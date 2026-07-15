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

	"github.com/jackc/pgx/v5"
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

// lusdDbStudent ist ein aktiver Schüler-Datensatz, indexiert über die LUSD-ID.
type lusdDbStudent struct {
	ID       string
	Klasse   string
	Vorname  string
	Nachname string
}

// ladeAktiveSchueler liest alle aktiven (nicht-abgegangenen, nicht soft-gelöschten)
// Schüler mit LUSD-ID. deleted_at IS NULL ist zwingend, sonst würde eine soft-
// gelöschte Zeile matchen und der aktive Schüler nie neu angelegt.
func ladeAktiveSchueler(ctx context.Context, tx pgx.Tx) (map[string]lusdDbStudent, error) {
	rows, err := tx.Query(ctx, "SELECT id, lusd_id, klasse, vorname, nachname FROM schueler WHERE ist_abgaenger = false AND deleted_at IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbStudents := make(map[string]lusdDbStudent)
	for rows.Next() {
		var id, klasse, vorname, nachname string
		var lusdID *string
		if err := rows.Scan(&id, &lusdID, &klasse, &vorname, &nachname); err != nil {
			return nil, err
		}
		if lusdID != nil && *lusdID != "" {
			dbStudents[*lusdID] = lusdDbStudent{ID: id, Klasse: klasse, Vorname: vorname, Nachname: nachname}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dbStudents, nil
}

// klassifiziereLusdRecords ordnet die CSV-Zeilen (rein klassifizierend, ohne
// Schreibzugriff) in Neuzugänge, Klassenwechsel und Abgänger ein.
func klassifiziereLusdRecords(records []parsedStudentRow, dbStudents map[string]lusdDbStudent, res *LusdPreviewResult) {
	csvLusdIDs := make(map[string]bool)
	for _, rec := range records {
		if rec.LusdID == "" {
			// Ohne LUSD-ID gibt es keinen stabilen Schlüssel — sichtbar zählen.
			res.SkippedNoID++
			continue
		}
		csvLusdIDs[rec.LusdID] = true
		dbRec, exists := dbStudents[rec.LusdID]
		if !exists {
			res.NewStudents = append(res.NewStudents, StudentDiff{
				ID:         rec.LusdID,
				Vorname:    rec.Vorname,
				Nachname:   rec.Nachname,
				NeueKlasse: rec.Klasse,
			})
			continue
		}
		if dbRec.Klasse != rec.Klasse {
			res.ClassChanges = append(res.ClassChanges, StudentDiff{
				ID:         rec.LusdID,
				Vorname:    rec.Vorname,
				Nachname:   rec.Nachname,
				AlteKlasse: dbRec.Klasse,
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
}

// computeLusdChanges compares the CSV records with the database inside a transaction
// and either returns the preview stats or actually applies the changes.
func (s *Server) computeLusdChanges(ctx context.Context, records []parsedStudentRow, apply bool, allowMassGraduation bool) (*LusdPreviewResult, error) {
	// Alles in einer TX für Atomarität. Bei Panic/frühem Return wird zurückgerollt.
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	dbStudents, err := ladeAktiveSchueler(ctx, tx)
	if err != nil {
		return nil, err
	}

	res := &LusdPreviewResult{
		NewStudents:      make([]StudentDiff, 0),
		ClassChanges:     make([]StudentDiff, 0),
		Graduates:        make([]StudentDiff, 0),
		TotalCsvRecords:  len(records),
		ActiveDbStudents: len(dbStudents),
	}

	// Erster Durchlauf: nur klassifizieren — der Schwellen-Check muss VOR dem
	// ersten destruktiven Statement entscheiden.
	klassifiziereLusdRecords(records, dbStudents, res)

	if !apply {
		return res, nil
	}

	// Serverseitige Massenabgang-Bremse: Die Abgänger-Behandlung anonymisiert
	// Namen IRREVERSIBEL. Die Schwelle wird hier durchgesetzt, wo die Destruktion passiert.
	if !allowMassGraduation &&
		res.ActiveDbStudents >= minStudentsForThreshold &&
		len(res.Graduates)*100 >= res.ActiveDbStudents*massGraduationThresholdPct {
		return nil, &errMassGraduation{Graduates: len(res.Graduates), Active: res.ActiveDbStudents}
	}

	if err := wendeLusdAenderungenAn(ctx, tx, records, dbStudents); err != nil {
		return nil, err
	}

	if err := behandleAbgaenger(ctx, tx, res.Graduates, dbStudents); err != nil {
		return nil, err
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
