package api

import (
	"bibliothek/auth"
	"bibliothek/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	// Umbenennungen: Abgänger + Neuzugang, die nach Geburtsdatum/Schuleintritt/Klasse/
	// Anschrift dieselbe Person sind (lusd_paarung.go). Der Admin bestätigt je Paar.
	Umbenennungen []UmbenennungDiff `json:"umbenennungen"`
	// KarenzTage: so lange bleiben Abgänger ohne offene Vorgänge nur gesperrt, bevor
	// sie anonymisiert werden (Einstellung abgaenger_karenz_tage; 0 = sofort).
	KarenzTage       int `json:"karenz_tage"`
	TotalCsvRecords  int `json:"total_csv_records"`
	ActiveDbStudents int `json:"active_db_students"`
	SkippedNoID      int `json:"skipped_no_id"`      // ID-Modus: CSV-Zeilen ohne LUSD-ID — werden nie importiert
	DublettenInDatei int `json:"dubletten_in_datei"` // Zeilen mit demselben Schlüssel, letzte gewann
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
		"%d von %d aktiven Schülern würden zu Abgängern (Schwelle: %d%%). Datei prüfen — falls der Massenabgang beabsichtigt ist (Schuljahreswechsel), Import mit Bestätigung wiederholen.",
		e.Graduates, e.Active, massGraduationThresholdPct)
}

// lusdLauf sind die Vorgaben eines Laufs: Vorschau oder Anwenden, ob der Massenabgang
// bestätigt wurde, und welche Umbenennungs-Paare der Admin gewählt hat.
type lusdLauf struct {
	apply, allowMassGraduation bool
	umbenennungen              []umbenennungWahl
}

// abgaengerKarenzTage liest die Karenzzeit aus den Einstellungen; ohne lesbare
// Einstellungen gilt die Vorgabe — derselbe Rückfall wie im nächtlichen Job.
func (s *Server) abgaengerKarenzTage(ctx context.Context) int {
	einst, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		return repository.StandardAbgaengerKarenzTage
	}
	return repository.AbgaengerKarenzTageOderStandard(einst)
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

// computeLusdLauf vergleicht die Datei mit dem Bestand in einer Transaktion und liefert
// entweder die Vorschau oder wendet die Änderungen an — samt der Umbenennungs-Wahl des
// Admins. (Die Kurzform ohne Wahl, computeLusd, lebt nur noch als Testhelfer.)
func (s *Server) computeLusdLauf(ctx context.Context, datei lusdDatei, lauf lusdLauf) (*LusdPreviewResult, error) {
	apply := lauf.apply
	// Die Karenzzeit vor der Transaktion lesen: eine Einstellung, keine Bestandsdaten —
	// sie gehört nicht in den Snapshot des Laufs, und die Mocks der Bremsen-Tests sehen
	// so eine feste Reihenfolge (Einstellungen → Begin → Lock → Bestand).
	karenzTage := s.abgaengerKarenzTage(ctx)
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
		Umbenennungen:    []UmbenennungDiff{},
		TotalCsvRecords:  len(datei.Zeilen),
		ActiveDbStudents: len(idx.aktiv),
		DublettenInDatei: datei.DublettenInDatei,
		KarenzTage:       karenzTage,
	}

	// Erster Durchlauf: nur klassifizieren — der Schwellen-Check muss VOR dem
	// ersten destruktiven Statement entscheiden.
	zuordnung := klassifiziereLusd(datei, idx, res)
	res.Adoptions = append(res.Adoptions, zuordnung.adoptionen...)
	// Umbenennungen aus Abgängern und Neuzugängen paaren; beim Anwenden die Wahl des
	// Admins einarbeiten (bestätigte Paare verlassen beide Listen).
	res.Umbenennungen = append(res.Umbenennungen, findeUmbenennungen(datei, bestand, idx, zuordnung)...)
	if apply {
		if err := uebernimmUmbenennungen(datei, lauf.umbenennungen, res.Umbenennungen, &zuordnung, res); err != nil {
			return nil, err
		}
	}

	if !apply {
		return res, nil
	}

	// Serverseitige Massenabgang-Bremse: Die Abgänger-Behandlung sperrt bzw. (Karenz 0)
	// anonymisiert IRREVERSIBEL. Die Schwelle wird hier durchgesetzt, wo es passiert.
	if !lauf.allowMassGraduation &&
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
	if err := behandleAbgaenger(ctx, tx, zuordnung.abgaengerIDs, res.KarenzTage); err != nil {
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
		lauf := lusdLauf{apply: apply, allowMassGraduation: apply && r.FormValue("confirm_graduates") == "true"}
		if apply {
			if lauf.umbenennungen, err = leseUmbenennungsWahl(r.FormValue("umbenennungen")); err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
				return
			}
		}
		res, err := s.computeLusdLauf(r.Context(), datei, lauf)
		if err == nil && apply {
			// Der einzige Pfad, der Schülernamen irreversibel anonymisiert, hinterließ bis
			// 22.08.2026 keinen Audit-Eintrag — weder Akteur noch Zahlen (Prüfung 22.08., B).
			// Zähler und Modus, keine Namen (Rechenschaft ohne neue PII).
			s.protokolliereLusdImport(r, res, lauf.allowMassGraduation)
		}
		if err != nil {
			var massErr *errMassGraduation
			var wahlErr *errUmbenennungUngueltig
			if errors.As(err, &massErr) {
				apierrors.SendHTTPError(w, http.StatusConflict, err)
				return
			}
			if errors.As(err, &wahlErr) {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, res)
	}
}

// leseUmbenennungsWahl liest das Formularfeld `umbenennungen` (JSON-Liste aus Zeile +
// schueler_id). Leer heißt: keine Paare bestätigt — dann läuft es wie bisher.
func leseUmbenennungsWahl(roh string) ([]umbenennungWahl, error) {
	if strings.TrimSpace(roh) == "" {
		return nil, nil
	}
	var wahl []umbenennungWahl
	if err := json.Unmarshal([]byte(roh), &wahl); err != nil {
		return nil, fmt.Errorf("Umbenennungs-Auswahl unlesbar: %w", err) //nolint:staticcheck // ST1005: nutzer-sichtbarer Text
	}
	return wahl, nil
}

// PostLusdPreviewHandler parst die CSV und liefert die Vorschau der Änderungen.
func (s *Server) PostLusdPreviewHandler() http.HandlerFunc { return s.lusdUploadHandler(false) }

// PostLusdImportHandler parst die CSV und wendet die Änderungen transaktional an.
// Ab massGraduationThresholdPct Abgängern verlangt er das Formularfeld
// confirm_graduates=true (HTTP 409 sonst) — zweite, bewusste Bestätigung.
func (s *Server) PostLusdImportHandler() http.HandlerFunc { return s.lusdUploadHandler(true) }

func zaehleBestaetigte(paare []UmbenennungDiff) int {
	n := 0
	for _, p := range paare {
		if p.Bestaetigt {
			n++
		}
	}
	return n
}

// protokolliereLusdImport schreibt den Apply-Lauf ins Admin-Audit: Modus, Zähler je
// Kategorie und ob der Massenabgang bestätigt wurde. Ohne Namen.
func (s *Server) protokolliereLusdImport(r *http.Request, res *LusdPreviewResult, massenabgangBestaetigt bool) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || s.DB == nil || s.DB.Pool == nil || res == nil {
		return
	}
	details := map[string]any{
		"modus":                    res.Modus,
		"zeilen":                   res.TotalCsvRecords,
		"neu":                      len(res.NewStudents),
		"klassenwechsel":           len(res.ClassChanges),
		"adoptionen":               len(res.Adoptions),
		"rueckkehrer":              len(res.Rueckkehrer),
		"abgaenger":                len(res.Graduates),
		"nicht_im_export":          len(res.NichtImExport),
		"nicht_abgleichbar":        len(res.NichtAbgleichbar),
		"mehrdeutig":               len(res.Mehrdeutig),
		"umbenennungen_bestaetigt": zaehleBestaetigte(res.Umbenennungen),
		"karenz_tage":              res.KarenzTage,
		"massenabgang_bestaetigt":  massenabgangBestaetigt,
	}
	if err := repository.NewAuditRepository(s.DB.Pool).
		LogAdminAktion(r.Context(), claims.UserID, "LUSD_IMPORT", getIP(r), details); err != nil {
		log.Printf("LUSD-Import: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}
