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

// StudentDiff ist ein Schüler-Eintrag der Vorschau. ID ist der Listenschlüssel fürs
// Frontend — die LUSD-ID (ID-Modus) oder die schueler-UUID, bei Neuzugängen im
// Namensmodus die Zeilennummer. Nie eine interne Kennung, die es nicht ohnehin gibt.
type StudentDiff struct {
	ID         string `json:"id"`
	Vorname    string `json:"vorname"`
	Nachname   string `json:"nachname"`
	AlteKlasse string `json:"alte_klasse,omitempty"`
	NeueKlasse string `json:"neue_klasse,omitempty"`
}

// AdoptionDiff beschreibt eine geplante ADOPTION (nur ID-Modus): Eine CSV-Zeile, deren
// LUSD-ID im Bestand fehlt, trifft über Name+Geburtsdatum auf einen bestehenden Schüler
// OHNE LUSD-ID (Handanlage/Littera-Import). Statt ihn zu duplizieren, wird die LUSD-ID
// nachgetragen. SchuelerID ist der bestehende Datensatz, LusdID die anzuheftende Kennung.
type AdoptionDiff struct {
	SchuelerID   string `json:"schueler_id"`
	LusdID       string `json:"lusd_id"`
	Vorname      string `json:"vorname"`
	Nachname     string `json:"nachname"`
	Geburtsdatum string `json:"geburtsdatum"`
	AlteKlasse   string `json:"alte_klasse,omitempty"`
	NeueKlasse   string `json:"neue_klasse,omitempty"`
}

// LusdPreviewResult ist die Vorschau, anhand derer das Sekretariat Namen und
// Klassenwechsel prüft, bevor es bestätigt. ActiveDbStudents ist die Bezugsgröße der
// Abgänger-Quote (abgleichbare aktive Schüler) — NICHT die CSV-Zeilenzahl.
type LusdPreviewResult struct {
	Modus            string         `json:"modus"` // "lusd_id" oder "name_geburtsdatum" (lusdModus.String)
	NewStudents      []StudentDiff  `json:"new_students"`
	ClassChanges     []StudentDiff  `json:"class_changes"`
	Adoptions        []AdoptionDiff `json:"adoptions"`         // ID-Modus: ID-lose Bestandsschüler bekommen ihre LUSD-ID
	Rueckkehrer      []StudentDiff  `json:"rueckkehrer"`       // Abgänger, die wieder im Export stehen — werden reaktiviert
	Graduates        []StudentDiff  `json:"graduates"`         // Abgänger (im Bestand, fehlen in der Datei)
	NichtImExport    []StudentDiff  `json:"nicht_im_export"`   // Namensmodus: nie bestätigte Handanlagen — bleiben unverändert
	NichtAbgleichbar []StudentDiff  `json:"nicht_abgleichbar"` // Namensmodus: ohne Geburtsdatum — bleiben unverändert
	Mehrdeutig       []StudentDiff  `json:"mehrdeutig"`        // gleicher Schlüssel mehrfach — wird nicht angefasst
	TotalCsvRecords  int            `json:"total_csv_records"`
	ActiveDbStudents int            `json:"active_db_students"`
	SkippedNoID      int            `json:"skipped_no_id"`      // ID-Modus: CSV-Zeilen ohne LUSD-ID — werden nie importiert
	DublettenInDatei int            `json:"dubletten_in_datei"` // Zeilen mit demselben Schlüssel, letzte gewann
}

// lusdImportLockKey serialisiert gleichzeitige LUSD-Importe (Advisory-Lock). Eigener
// Nummernkreis, überschneidet sich nicht mit anderen Advisory-Keys im Projekt.
const lusdImportLockKey int64 = 750_2026

// massGraduationThresholdPct: Ab diesem Anteil an Abgängern (bezogen auf die
// abgleichbaren aktiven DB-Schüler) verweigert der Import ohne explizite Bestätigung —
// die Abgänger-Behandlung anonymisiert irreversibel. Schutz gegen versehentliche
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
// LUSD-Parser (lusd_parser.go): Modus-Erkennung, Dedupe, echte Datums-Validierung,
// harte Fehler statt stillem Überspringen.
func readLusdUpload(r *http.Request) (lusdDatei, error) {
	file, _, err := r.FormFile("csvFile")
	if err != nil {
		return lusdDatei{}, fmt.Errorf("CSV-Datei fehlt: %w", err)
	}
	defer closeutil.LogClose(file, "lusd upload")

	content, err := io.ReadAll(file)
	if err != nil {
		return lusdDatei{}, fmt.Errorf("CSV konnte nicht gelesen werden: %w", err)
	}
	return parseLusdDatei(content)
}

// generateImportBarcode liefert einen eindeutigen vorläufigen Barcode für im
// LUSD-Import neu angelegte Schüler. Der laufende Zähler macht die Barcodes
// INNERHALB eines Imports garantiert kollisionsfrei (barcode_id ist UNIQUE —
// die frühere Nanosekunden-Variante kollidierte per Geburtstagsparadoxon ab
// ~50 Neuzugängen regelmäßig und brach den gesamten Import ab).
func generateImportBarcode(counter int) string {
	return fmt.Sprintf("S-%06d%04d", time.Now().Unix()%1000000, counter)
}

// computeLusd vergleicht die Datei mit dem Bestand in einer Transaktion und liefert
// entweder die Vorschau oder wendet die Änderungen an.
func (s *Server) computeLusd(ctx context.Context, datei lusdDatei, apply bool, allowMassGraduation bool) (*LusdPreviewResult, error) {
	// Alles in einer TX für Atomarität. Bei Panic/frühem Return wird zurückgerollt.
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	// Beim ANWENDEN alle LUSD-Läufe hart serialisieren (Advisory-Lock, transaktions-
	// gebunden). Zwei gleichzeitige Importe arbeiteten sonst auf sich überholenden
	// Snapshots: kollidierende Import-Barcodes/lusd_ids reißen den zweiten Import
	// komplett ab (23505 → Rollback), und im ungünstigsten Fall anonymisiert der eine
	// einen Abgänger, den der andere gerade wieder aktiviert. Die Vorschau (apply=false)
	// nimmt den Lock NICHT — sie liest nur.
	if apply {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lusdImportLockKey); err != nil {
			return nil, fmt.Errorf("lusd-import sperren fehlgeschlagen: %w", err)
		}
	}

	bestand, err := ladeLusdBestand(ctx, tx)
	if err != nil {
		return nil, err
	}
	idx := baueLusdIndex(bestand, datei.Modus)

	res := &LusdPreviewResult{
		Modus:            datei.Modus.String(),
		NewStudents:      []StudentDiff{},
		ClassChanges:     []StudentDiff{},
		Adoptions:        []AdoptionDiff{},
		Rueckkehrer:      []StudentDiff{},
		Graduates:        []StudentDiff{},
		NichtImExport:    []StudentDiff{},
		NichtAbgleichbar: []StudentDiff{},
		Mehrdeutig:       []StudentDiff{},
		TotalCsvRecords:  len(datei.Zeilen),
		ActiveDbStudents: len(idx.aktiv),
		DublettenInDatei: datei.DublettenInDatei,
	}

	// Erster Durchlauf: nur klassifizieren — der Schwellen-Check muss VOR dem
	// ersten destruktiven Statement entscheiden.
	zuordnung := klassifiziereLusd(datei, idx, res)
	res.Adoptions = append(res.Adoptions, zuordnung.adoptionen...)

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

	// Adoptionen ZUERST: die LUSD-ID an den bestehenden ID-losen Schüler heften. Danach
	// findet ihn wendeLusdAenderungenAn über findeAktivenSchuelerNachLusdID und übernimmt
	// Klasse + Kontaktdaten wie bei jedem Bestandsschüler — statt ein Duplikat anzulegen.
	if err := adoptiereWaisen(ctx, tx, zuordnung.adoptionen); err != nil {
		return nil, err
	}
	if err := wendeLusdAenderungenAn(ctx, tx, datei, zuordnung); err != nil {
		return nil, err
	}
	if err := behandleAbgaenger(ctx, tx, zuordnung.abgaengerIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

// lusdUploadHandler bündelt, was Vorschau und Import gemeinsam haben: Upload-Grenze,
// Parsen, Fehlerabbildung. apply unterscheidet die beiden Routen.
func (s *Server) lusdUploadHandler(apply bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		datei, err := readLusdUpload(r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		allowMass := apply && r.FormValue("confirm_graduates") == "true"
		res, err := s.computeLusd(r.Context(), datei, apply, allowMass)
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

// PostLusdPreviewHandler parst die CSV und liefert die Vorschau der Änderungen.
func (s *Server) PostLusdPreviewHandler() http.HandlerFunc { return s.lusdUploadHandler(false) }

// PostLusdImportHandler parst die CSV und wendet die Änderungen transaktional an.
// Ab massGraduationThresholdPct Abgängern verlangt er das Formularfeld
// confirm_graduates=true (HTTP 409 sonst) — zweite, bewusste Bestätigung.
func (s *Server) PostLusdImportHandler() http.HandlerFunc { return s.lusdUploadHandler(true) }
