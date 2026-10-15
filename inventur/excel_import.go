package inventur

import (
	"bibliothek/pkg/xlsxgrenze"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"bibliothek/pkg/isbnutil"
	"bibliothek/pkg/safego"

	"github.com/xuri/excelize/v2"
)

func leseImportZeilen(w http.ResponseWriter, r *http.Request) ([][]string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100 MB
		return nil, errors.New("datei zu groß oder ungültig")
	}

	datei, dateiKopf, err := r.FormFile("file")
	if err != nil {
		return nil, errors.New("keine datei gefunden")
	}
	defer func() { _ = datei.Close() }() //nolint:errcheck

	if strings.HasSuffix(strings.ToLower(dateiKopf.Filename), ".csv") {
		return leseCSVZeilen(datei)
	}
	return leseExcelZeilen(datei)
}

// leseCSVZeilen liest eine CSV-Datei mit heuristischer Trennzeichenerkennung (, oder ;).
func leseCSVZeilen(datei io.Reader) ([][]string, error) {
	inhalt, err := io.ReadAll(datei)
	if err != nil {
		return nil, errors.New("fehler beim lesen der csv-datei")
	}
	inhaltText := string(inhalt)
	trenner := ','
	if strings.Count(inhaltText, ";") > strings.Count(inhaltText, ",") {
		trenner = ';'
	}
	leser := csv.NewReader(strings.NewReader(inhaltText))
	leser.Comma = trenner
	leser.LazyQuotes = true
	zeilen, err := leser.ReadAll()
	if err != nil {
		return nil, errors.New("ungültige csv-datei")
	}
	if len(zeilen) == 0 {
		return nil, errors.New("datei ist leer")
	}
	return zeilen, nil
}

// leseExcelZeilen liest die erste Tabelle einer Excel-Datei, hinter der Schranke aus
// xlsxgrenze: Eine kaputte oder verschlüsselte Datei wird eine „ungültige excel-datei",
// kein Absturz.
func leseExcelZeilen(datei io.Reader) ([][]string, error) {
	zeilen, err := xlsxgrenze.LeseMappe(datei, func(f *excelize.File) ([][]string, error) {
		blaetter := f.GetSheetList()
		if len(blaetter) == 0 {
			return nil, errors.New("excel-datei ist leer")
		}
		zeilen, err := f.GetRows(blaetter[0])
		if err != nil || len(zeilen) < 1 {
			return nil, errors.New("keine daten gefunden")
		}
		return zeilen, nil
	})
	if xlsxgrenze.IstUnlesbar(err) {
		return nil, errors.New("ungültige excel-datei")
	}
	return zeilen, err
}

func bestimmeSpalten(kopf []string) (map[string]int, bool) {
	spalten := map[string]int{"isbn": -1, "titel": -1, "autor": -1, "fach": -1, "klasse": -1, "bestand": -1}
	mitKopfzeile := true

	// Die Kopfzeile erkennen.
	for i, spalte := range kopf {
		if feld := feldZurKopfzeile(spalte); feld != "" {
			spalten[feld] = i
		}
	}

	// Ohne erkannte Kopfzeile (keine Spalte ISBN) die Spalten aus dem Inhalt erraten.
	if spalten["isbn"] == -1 {
		mitKopfzeile = false
		errateSpaltenAusInhalt(kopf, spalten)
	}
	return spalten, mitKopfzeile
}

// errateSpaltenAusInhalt rät die Spaltenzuordnung anhand des Zeileninhalts, wenn keine
// Kopfzeile erkannt wurde: ISBN am 978/979-Präfix, reine Zahlen als Bestand, längere
// Texte als Titel. Mutiert colIdx.
func errateSpaltenAusInhalt(kopf []string, spalten map[string]int) {
	for i, spalte := range kopf {
		wert := strings.TrimSpace(spalte)
		bereinigt := isbnutil.OhneTrenner(wert)
		if (strings.HasPrefix(wert, "978") || strings.HasPrefix(wert, "979")) && len(bereinigt) >= 10 {
			spalten["isbn"] = i
			continue
		}
		if _, err := strconv.Atoi(wert); err == nil {
			if spalten["bestand"] == -1 {
				spalten["bestand"] = i
			}
			continue
		}
		if spalten["titel"] == -1 && len(wert) > 2 {
			spalten["titel"] = i
		}
	}
}

func (t *Tueren) verarbeiteImportZeilen(ctx context.Context, datenZeilen [][]string, spalten map[string]int) ([]Titel, int32, error) {
	var gescheitert int32 = 0
	var ersterFehler error
	var fehlerSperre sync.Mutex
	var wg sync.WaitGroup
	begrenzung := make(chan struct{}, 10)

	var zuSchreiben []Titel
	var titelSperre sync.Mutex

	for _, zeile := range datenZeilen {
		wg.Add(1)
		// Semaphore vor dem Start der Goroutine belegen, damit die Schleife blockiert und
		// nicht für jede Zeile sofort eine Goroutine erzeugt. Bei großen Importen (z. B. 20k
		// Zeilen) entstünden sonst zehntausende gleichzeitig lebende Goroutinen, die alle nur
		// auf einen Semaphor-Slot warten — unnötiger Speicher- und Scheduler-Druck.
		begrenzung <- struct{}{}
		go func(zeile []string) {
			defer wg.Done()
			defer func() { <-begrenzung }()
			defer safego.FangePanic("excel-import-zeile")

			eintrag, err := verarbeiteImportZeile(importZeile{
				Kontext:   ctx,
				Zeile:     zeile,
				Spalten:   spalten,
				Abfragen:  t.abfragen,
				Metadaten: t.metadaten,
			})
			if err != nil {
				atomic.AddInt32(&gescheitert, 1)
				fehlerSperre.Lock()
				if ersterFehler == nil {
					ersterFehler = err
				}
				fehlerSperre.Unlock()
			} else if eintrag != nil {
				titelSperre.Lock()
				zuSchreiben = append(zuSchreiben, *eintrag)
				titelSperre.Unlock()
			}
		}(zeile)
	}
	wg.Wait()

	return zuSchreiben, gescheitert, ersterFehler
}

// bereiteImportZeilenVor validiert Methode und Datei, ermittelt die Spaltenzuordnung und
// liefert die Datenzeilen (ohne Kopfzeile). ok=false bedeutet: die Fehlerantwort wurde
// bereits geschrieben.
func (t *Tueren) bereiteImportZeilenVor(w http.ResponseWriter, r *http.Request) (datenZeilen [][]string, spalten map[string]int, ok bool) {
	if r.Method != http.MethodPost {
		schreibeFehler(w, http.StatusMethodNotAllowed, "Nur Anfragen mit POST")
		return nil, nil, false
	}

	zeilen, err := leseImportZeilen(w, r)
	if err != nil {
		schreibeFehler(w, http.StatusBadRequest, err.Error())
		return nil, nil, false
	}

	spalten, mitKopfzeile := bestimmeSpalten(zeilen[0])
	if spalten["isbn"] == -1 {
		schreibeFehler(w, http.StatusBadRequest, "Die Spalte ISBN fehlt in der Datei. Bitte eine Spalte ISBN anlegen.")
		return nil, nil, false
	}

	datenZeilen = zeilen
	if mitKopfzeile {
		datenZeilen = zeilen[1:]
	}

	const maxImportZeilen = 100000
	if len(datenZeilen) > maxImportZeilen {
		schreibeFehler(w, http.StatusBadRequest, fmt.Sprintf("zu viele zeilen (%d), maximal %d erlaubt", len(datenZeilen), maxImportZeilen))
		return nil, nil, false
	}

	return datenZeilen, spalten, true
}

// speichereImportierteTitel schreibt die eingelesenen Bücher per Batch-Upsert; schlägt der
// Batch fehl, wird zeilenweise als Fallback eingefügt. Liefert die aktualisierten
// Zähler zurück.
func (t *Tueren) speichereImportierteTitel(ctx context.Context, zuSchreiben []Titel, gescheitert int32, ersterFehler error) (importiert, gescheitertAus int32, fehlerAus error) {
	if len(zuSchreiben) == 0 {
		return 0, gescheitert, ersterFehler
	}

	anzahl, err := t.abfragen.SchreibeTitelStapel(ctx, zuSchreiben)
	if err != nil {
		return t.speichereTitelEinzeln(ctx, zuSchreiben, gescheitert, ersterFehler)
	}

	// #nosec G115 - die Zahl ist durch maxImportZeilen (5000) begrenzt
	importiert += int32(anzahl)
	if importiert == 0 {
		// Falls count 0 ist wegen z.B. nur Updates in manchen DB Versionen
		// #nosec G115 - die Länge ist durch maxImportZeilen (5000) begrenzt
		importiert = int32(len(zuSchreiben))
	}
	return importiert, gescheitert, ersterFehler
}

// speichereTitelEinzeln führt einzelne Inserts als Fallback aus, wenn der Batch-Insert fehlschlägt.
func (t *Tueren) speichereTitelEinzeln(ctx context.Context, eintraege []Titel, gescheitert int32, ersterFehler error) (importiert, gescheitertAus int32, fehlerAus error) {
	for _, eintrag := range eintraege {
		_, einzelFehler := t.abfragen.SchreibeTitel(ctx, eintrag)
		if einzelFehler != nil {
			gescheitert++
			if ersterFehler == nil {
				ersterFehler = einzelFehler
			}
		} else {
			importiert++
		}
	}
	return importiert, gescheitert, ersterFehler
}

// ExcelImport ist die Tür POST /api/books/import: Sie übernimmt eine Liste von Titeln aus
// Excel oder CSV.
func (t *Tueren) ExcelImport(w http.ResponseWriter, r *http.Request) {
	schluessel, ok := importSchluessel(w, r)
	if !ok {
		return
	}
	datenZeilen, spalten, ok := t.bereiteImportZeilenVor(w, r)
	if !ok {
		return
	}
	// Erst nach der Dateiprüfung reservieren: Ein Formfehler ist deterministisch und
	// braucht kein Gedächtnis; ein reservierter Schlüssel dagegen sperrt den zweiten
	// Aufruf, solange dieser läuft.
	if schluessel != "" {
		frisch, alt, err := t.reserviereImportLauf(r.Context(), schluessel)
		if err != nil {
			schreibeFehler(w, http.StatusInternalServerError, "import-schlüssel konnte nicht geprüft werden")
			return
		}
		if !frisch {
			schreibeJSON(w, alt.Status, alt.Daten)
			return
		}
	}

	zuSchreiben, gescheitert, ersterFehler := t.verarbeiteImportZeilen(r.Context(), datenZeilen, spalten)
	importiert, gescheitert, ersterFehler := t.speichereImportierteTitel(r.Context(), zuSchreiben, gescheitert, ersterFehler)

	status, antwort := importAntwort(importiert, gescheitert, ersterFehler)
	if schluessel != "" {
		t.schliesseImportLauf(r.Context(), schluessel, importiert > 0, status, antwort)
	}
	schreibeJSON(w, status, antwort)
}

// importAntwort formt das Ergebnis eines Laufs — eine Stelle, weil dieselbe Antwort
// beim Wiederholungsaufruf aus dem Idempotenz-Gedächtnis kommt.
func importAntwort(importiert, gescheitert int32, ersterFehler error) (int, map[string]any) {
	if importiert == 0 && gescheitert > 0 {
		meldung := "keine bücher konnten importiert werden."
		if ersterFehler != nil {
			meldung += " fehler: " + ersterFehler.Error()
		}
		return http.StatusBadRequest, map[string]any{"error": meldung}
	}

	zusammenfassung := fmt.Sprintf("%d bücher importiert", importiert)
	if gescheitert > 0 {
		zusammenfassung += fmt.Sprintf(", %d fehlgeschlagen", gescheitert)
	}
	return http.StatusOK, map[string]any{
		"message":  zusammenfassung,
		"imported": importiert,
		"failed":   gescheitert,
	}
}
