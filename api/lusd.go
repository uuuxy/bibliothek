package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/closeutil"
)

// StudentDiff represents a single student-level change for the qualitative LUSD
// preview. ID holds the LUSD-ID (from the export file), not the internal DB-UUID —
// it is a stable key for the frontend list and avoids leaking internal identifiers.
type StudentDiff struct {
	ID         string `json:"id"`
	Vorname    string `json:"vorname"`
	Nachname   string `json:"nachname"`
	AlteKlasse string `json:"alte_klasse,omitempty"`
	NeueKlasse string `json:"neue_klasse,omitempty"`
}

// LusdPreviewResult contains the detailed diff lists for the frontend preview,
// so the Sekretariat can visually verify names and class changes before committing.
// ActiveDbStudents ist die Bezugsgröße für die Abgänger-Quote (aktive Schüler mit
// LUSD-ID in der DB) — NICHT die CSV-Zeilenzahl.
type LusdPreviewResult struct {
	NewStudents      []StudentDiff `json:"new_students"`
	ClassChanges     []StudentDiff `json:"class_changes"`
	Graduates        []StudentDiff `json:"graduates"` // Abgänger (missing in CSV but in DB)
	TotalCsvRecords  int           `json:"total_csv_records"`
	ActiveDbStudents int           `json:"active_db_students"`
	SkippedNoID      int           `json:"skipped_no_id"` // CSV-Zeilen ohne LUSD-ID — werden nie importiert
}

// massGraduationThresholdPct: Ab diesem Anteil an Abgängern (bezogen auf die
// aktiven DB-Schüler) verweigert der Import ohne explizite Bestätigung — die
// Abgänger-Behandlung anonymisiert irreversibel. Schutz gegen versehentliche
// Teilexporte (z. B. nur eine Jahrgangsstufe in der Datei).
const massGraduationThresholdPct = 30

// minStudentsForThreshold verhindert, dass der Schwellen-Schutz winzige
// Bestände (Erstinstallation, Testsysteme) blockiert.
const minStudentsForThreshold = 10

// errMassGraduation trägt die Zahlen für die 409-Antwort ans Frontend.
type errMassGraduation struct {
	Graduates int
	Active    int
}

func (e *errMassGraduation) Error() string {
	return fmt.Sprintf(
		"%d von %d aktiven Schülern würden als Abgänger anonymisiert (Schwelle: %d%%). Datei prüfen — falls der Massenabgang beabsichtigt ist (Schuljahreswechsel), Import mit Bestätigung wiederholen.",
		e.Graduates, e.Active, massGraduationThresholdPct)
}

// readLusdUpload liest die hochgeladene CSV und parst sie mit dem getesteten
// LUSD-Parser (lusd_parser.go): exakte Spaltennamen, Dedupe per LUSD-ID
// (letzte Zeile gewinnt), echte Datums-Validierung, harte Fehler statt
// stillem Überspringen.
func readLusdUpload(r *http.Request) ([]parsedStudentRow, error) {
	file, _, err := r.FormFile("csvFile")
	if err != nil {
		return nil, fmt.Errorf("CSV-Datei fehlt: %w", err)
	}
	defer closeutil.LogClose(file, "lusd upload")

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("CSV konnte nicht gelesen werden: %w", err)
	}

	rows, _, err := parseLUSDCSV(content)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// generateImportBarcode liefert einen eindeutigen vorläufigen Barcode für im
// LUSD-Import neu angelegte Schüler. Der laufende Zähler macht die Barcodes
// INNERHALB eines Imports garantiert kollisionsfrei (barcode_id ist UNIQUE —
// die frühere Nanosekunden-Variante kollidierte per Geburtstagsparadoxon ab
// ~50 Neuzugängen regelmäßig und brach den gesamten Import ab).
func generateImportBarcode(counter int) string {
	return fmt.Sprintf("S-%06d%04d", time.Now().Unix()%1000000, counter)
}

// computeLusdChanges compares the CSV records with the database inside a transaction
// and either returns the preview stats or actually applies the changes.
func (s *Server) computeLusdChanges(ctx context.Context, records []parsedStudentRow, apply bool, allowMassGraduation bool) (*LusdPreviewResult, error) {
	// Start transaction. Everything is done within a single TX to ensure atomicity.
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// Always rollback on panic or early return. If we apply, we commit at the very end.
	defer db.SafeRollback(ctx, tx)

	// Read all current students. deleted_at IS NULL ist zwingend: sonst „matcht"
	// eine soft-gelöschte, aber noch auf der LUSD-Liste stehende Zeile, der aktive
	// Schüler würde nie neu angelegt und bliebe unsichtbar. Re-Insert einer zuvor
	// gelöschten lusd_id ist dank partiellem Unique-Index (Migration 035) sicher.
	// Vorname/Nachname werden mitgelesen, damit Abgänger-Diffs den Namen VOR der
	// DSGVO-Anonymisierung anzeigen können.
	rows, err := tx.Query(ctx, "SELECT id, lusd_id, klasse, vorname, nachname FROM schueler WHERE ist_abgaenger = false AND deleted_at IS NULL")
	if err != nil {
		return nil, err
	}
	dbStudents := make(map[string]struct {
		ID       string
		Klasse   string
		Vorname  string
		Nachname string
	})
	for rows.Next() {
		var id, klasse, vorname, nachname string
		var lusdID *string
		if err := rows.Scan(&id, &lusdID, &klasse, &vorname, &nachname); err != nil {
			rows.Close()
			return nil, err
		}
		if lusdID != nil && *lusdID != "" {
			dbStudents[*lusdID] = struct {
				ID       string
				Klasse   string
				Vorname  string
				Nachname string
			}{id, klasse, vorname, nachname}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	res := &LusdPreviewResult{
		NewStudents:      make([]StudentDiff, 0),
		ClassChanges:     make([]StudentDiff, 0),
		Graduates:        make([]StudentDiff, 0),
		TotalCsvRecords:  len(records),
		ActiveDbStudents: len(dbStudents),
	}
	csvLusdIDs := make(map[string]bool)

	// Erster Durchlauf: nur klassifizieren (neu / Klassenwechsel / Abgänger),
	// noch nichts schreiben — der Schwellen-Check unten muss VOR dem ersten
	// destruktiven Statement entscheiden.
	for _, rec := range records {
		if rec.LusdID == "" {
			// Ohne LUSD-ID gibt es keinen stabilen Schlüssel: jeder Import würde
			// dieselbe Person erneut anlegen. Sichtbar zählen statt still anlegen.
			res.SkippedNoID++
			continue
		}
		csvLusdIDs[rec.LusdID] = true
		if dbRec, exists := dbStudents[rec.LusdID]; exists {
			if dbRec.Klasse != rec.Klasse {
				res.ClassChanges = append(res.ClassChanges, StudentDiff{
					ID:         rec.LusdID,
					Vorname:    rec.Vorname,
					Nachname:   rec.Nachname,
					AlteKlasse: dbRec.Klasse,
					NeueKlasse: rec.Klasse,
				})
			}
		} else {
			res.NewStudents = append(res.NewStudents, StudentDiff{
				ID:         rec.LusdID,
				Vorname:    rec.Vorname,
				Nachname:   rec.Nachname,
				NeueKlasse: rec.Klasse,
			})
		}
	}
	for lusdID, dbRec := range dbStudents {
		if !csvLusdIDs[lusdID] {
			res.Graduates = append(res.Graduates, StudentDiff{
				ID:         lusdID,
				Vorname:    dbRec.Vorname,
				Nachname:   dbRec.Nachname,
				AlteKlasse: dbRec.Klasse,
			})
		}
	}

	if !apply {
		return res, nil
	}

	// Serverseitige Massenabgang-Bremse: Die Abgänger-Behandlung anonymisiert
	// Namen IRREVERSIBEL. Ein UI-Banner allein schützt nicht — die Schwelle wird
	// hier durchgesetzt, wo die Destruktion passiert.
	if !allowMassGraduation &&
		res.ActiveDbStudents >= minStudentsForThreshold &&
		len(res.Graduates)*100 >= res.ActiveDbStudents*massGraduationThresholdPct {
		return nil, &errMassGraduation{Graduates: len(res.Graduates), Active: res.ActiveDbStudents}
	}

	// Zweiter Durchlauf: anwenden.
	barcodeCounter := 0
	for _, rec := range records {
		if rec.LusdID == "" {
			continue
		}
		if dbRec, exists := dbStudents[rec.LusdID]; exists {
			if dbRec.Klasse != rec.Klasse {
				if _, err := tx.Exec(ctx, "UPDATE schueler SET klasse = $1, aktualisiert_am = NOW() WHERE id = $2", rec.Klasse, dbRec.ID); err != nil {
					return nil, err
				}
			}
		} else {
			barcodeCounter++
			year := time.Now().Year() + 5 // Default abgang
			if _, err := tx.Exec(ctx,
				"INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, geburtsdatum) VALUES ($1, $2, $3, $4, $5, $6, $7)",
				generateImportBarcode(barcodeCounter), rec.Vorname, rec.Nachname, rec.Klasse, year, rec.LusdID, rec.GebDatum); err != nil {
				return nil, err
			}
		}
	}

	for _, grad := range res.Graduates {
		dbRec := dbStudents[grad.ID]
		// Check if they have active loans
		var pending int
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL", dbRec.ID).Scan(&pending); err != nil {
			return nil, err
		}
		if pending > 0 {
			// Mark as inactive/abgänger — Name bleibt fürs Mahnwesen sichtbar
			_, err = tx.Exec(ctx, "UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true, aktualisiert_am = NOW() WHERE id = $1", dbRec.ID)
		} else {
			// DSGVO compliant anonymization
			// Append internal DB UUID to avoid unique constraint violations
			anonymisiertName := fmt.Sprintf("Anonymisiert-%s", dbRec.ID)
			_, err = tx.Exec(ctx, "UPDATE schueler SET vorname = 'Abgänger', nachname = $1, klasse = 'ABG', ist_abgaenger = true, ist_gesperrt = true, aktualisiert_am = NOW() WHERE id = $2", anonymisiertName, dbRec.ID)
		}
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return res, nil
}

// PostLusdPreviewHandler parses the CSV and returns a preview of changes.
func (s *Server) PostLusdPreviewHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		records, err := readLusdUpload(r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		res, err := s.computeLusdChanges(r.Context(), records, false, false)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, res)
	}
}

// PostLusdImportHandler parses the CSV and applies the changes transactionally.
// Ab massGraduationThresholdPct Abgängern verlangt er das Formularfeld
// confirm_graduates=true (HTTP 409 sonst) — zweite, bewusste Bestätigung.
func (s *Server) PostLusdImportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		records, err := readLusdUpload(r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		allowMass := r.FormValue("confirm_graduates") == "true"
		res, err := s.computeLusdChanges(r.Context(), records, true, allowMass)
		if err != nil {
			var massErr *errMassGraduation
			if errors.As(err, &massErr) {
				apierrors.SendHTTPError(w, http.StatusConflict, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, res)
	}
}
