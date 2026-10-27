package katalog

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/katalogdienste"
	"bibliothek/pkg/isbnutil"
	"bibliothek/pkg/lmf"
	"bibliothek/pkg/safego"
	"bibliothek/pkg/xlsxgrenze"
	"bibliothek/repository"

	"github.com/xuri/excelize/v2"
)

func leseImportZeilen(w http.ResponseWriter, r *http.Request) ([][]string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100 MB
		return nil, apierrors.BadRequest("Die Datei ließ sich nicht hochladen. Möglich sind höchstens 100 MB.", err)
	}

	datei, dateiKopf, err := r.FormFile("file")
	if err != nil {
		return nil, apierrors.BadRequest("Bitte eine Datei auswählen", err)
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
		return nil, apierrors.BadRequest("Die CSV-Datei ließ sich nicht lesen", err)
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
		return nil, apierrors.BadRequest("Die CSV-Datei ließ sich nicht lesen", err)
	}
	if len(zeilen) == 0 {
		return nil, apierrors.BadRequest("Die Datei ist leer", nil)
	}
	return zeilen, nil
}

// leseExcelZeilen liest die erste Tabelle einer Excel-Datei, hinter der Schranke aus
// xlsxgrenze: Eine kaputte oder verschlüsselte Datei wird ein Bedienfehler mit Meldung, kein
// Absturz.
func leseExcelZeilen(datei io.Reader) ([][]string, error) {
	zeilen, err := xlsxgrenze.LeseMappe(datei, func(f *excelize.File) ([][]string, error) {
		blaetter := f.GetSheetList()
		if len(blaetter) == 0 {
			return nil, apierrors.BadRequest("Die Excel-Datei ist leer", nil)
		}
		zeilen, err := f.GetRows(blaetter[0])
		if err != nil || len(zeilen) < 1 {
			return nil, apierrors.BadRequest("Die Excel-Datei enthält keine Daten", nil)
		}
		return zeilen, nil
	})
	var meldung *apierrors.Fehler
	if err != nil && !errors.As(err, &meldung) {
		return nil, apierrors.BadRequest("Die Excel-Datei ließ sich nicht lesen", err)
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

func (t *Tueren) verarbeiteImportZeilen(ctx context.Context, datenZeilen [][]string, spalten map[string]int) ([]repository.KatalogTitel, int32, error) {
	var gescheitert int32 = 0
	var ersterFehler error
	var fehlerSperre sync.Mutex
	var wg sync.WaitGroup
	begrenzung := make(chan struct{}, 10)

	var zuSchreiben []repository.KatalogTitel
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

			eintrag, err := verarbeiteImportZeile(ctx, importZeile{
				Zeile:     zeile,
				Spalten:   spalten,
				Metadaten: t.katalogdienste(),
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

// bereiteImportZeilenVor prüft die Datei, ermittelt die Spaltenzuordnung und liefert die
// Datenzeilen (ohne Kopfzeile). ok=false bedeutet: die Fehlerantwort wurde bereits
// geschrieben.
func (t *Tueren) bereiteImportZeilenVor(w http.ResponseWriter, r *http.Request) (datenZeilen [][]string, spalten map[string]int, ok bool) {
	zeilen, err := leseImportZeilen(w, r)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		return nil, nil, false
	}

	spalten, mitKopfzeile := bestimmeSpalten(zeilen[0])
	if spalten["isbn"] == -1 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Spalte ISBN fehlt in der Datei. Bitte eine Spalte ISBN anlegen.", nil)
		return nil, nil, false
	}

	datenZeilen = zeilen
	if mitKopfzeile {
		datenZeilen = zeilen[1:]
	}

	const maxImportZeilen = 100000
	if len(datenZeilen) > maxImportZeilen {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, fmt.Sprintf("Die Datei hat %d Zeilen, höchstens %d sind möglich", len(datenZeilen), maxImportZeilen), nil)
		return nil, nil, false
	}

	return datenZeilen, spalten, true
}

// speichereImportierteTitel schreibt die eingelesenen Bücher per Batch-Upsert; schlägt der
// Batch fehl, wird zeilenweise als Fallback eingefügt. Liefert die aktualisierten
// Zähler zurück.
func (t *Tueren) speichereImportierteTitel(ctx context.Context, zuSchreiben []repository.KatalogTitel, gescheitert int32, ersterFehler error) (importiert, gescheitertAus int32, fehlerAus error) {
	if len(zuSchreiben) == 0 {
		return 0, gescheitert, ersterFehler
	}

	anzahl, err := t.titelAbfragen().SchreibeTitelStapel(ctx, zuSchreiben)
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
func (t *Tueren) speichereTitelEinzeln(ctx context.Context, eintraege []repository.KatalogTitel, gescheitert int32, ersterFehler error) (importiert, gescheitertAus int32, fehlerAus error) {
	for _, eintrag := range eintraege {
		_, einzelFehler := t.titelAbfragen().SchreibeTitelAusListe(ctx, eintrag)
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
func (t *Tueren) ExcelImport() http.HandlerFunc {
	return t.importiereListe
}

// importiereListe ist der Rumpf von ExcelImport. Er steht auf der obersten Ebene, weil eine
// Closure für die Komplexitätsmessung als eigene Ebene zählt.
func (t *Tueren) importiereListe(w http.ResponseWriter, r *http.Request) {
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
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "Der Schlüssel des Imports ließ sich nicht prüfen", err)
			return
		}
		if !frisch {
			tuer.AntworteJSON(w, alt.Status, alt.Daten)
			return
		}
	}

	zuSchreiben, gescheitert, ersterFehler := t.verarbeiteImportZeilen(r.Context(), datenZeilen, spalten)
	importiert, gescheitert, ersterFehler := t.speichereImportierteTitel(r.Context(), zuSchreiben, gescheitert, ersterFehler)

	status, antwort := importAntwort(importiert, gescheitert, ersterFehler)
	if ersterFehler != nil {
		slog.Warn("Listenimport: Zeilen nicht gespeichert", "gescheitert", gescheitert, "fehler", ersterFehler)
	}
	if schluessel != "" {
		t.schliesseImportLauf(r.Context(), schluessel, importiert > 0, status, antwort)
	}
	tuer.AntworteJSON(w, status, antwort)
}

// importAntwort formt das Ergebnis eines Laufs — eine Stelle, weil dieselbe Antwort
// beim Wiederholungsaufruf aus dem Idempotenz-Gedächtnis kommt. Der Text des Fehlers ist für das
// Log geschrieben und geht nicht hinaus (docs/STIL.md 11); die Meldung nennt nur einen Grund, den
// repository für die Oberfläche formuliert hat.
func importAntwort(importiert, gescheitert int32, ersterFehler error) (int, map[string]any) {
	if importiert == 0 && gescheitert > 0 {
		return http.StatusBadRequest, map[string]any{"error": repository.MitGrund("Keine Zeile der Liste ließ sich importieren", ersterFehler)}
	}

	zusammenfassung := fmt.Sprintf("%d Titel importiert", importiert)
	if gescheitert > 0 {
		zusammenfassung += fmt.Sprintf(", %d fehlgeschlagen", gescheitert)
	}
	return http.StatusOK, map[string]any{
		"message":  zusammenfassung,
		"imported": importiert,
		"failed":   gescheitert,
	}
}

// errateFachAusTitel rät das Fach aus dem Titel — über dieselbe Liste wie der
// ISBN-Lookup (pkg/lmf), damit beide Wege dasselbe Fach registrieren.
func errateFachAusTitel(titelText string) string {
	return lmf.FachAusText(strings.TrimSpace(titelText))
}

func feldZurKopfzeile(kopf string) string {
	roh := strings.ToLower(strings.TrimSpace(kopf))

	// Ein Durchgang mit strings.Builder statt mehrerer strings.ReplaceAll: Leerzeichen,
	// Unterstriche, Bindestriche und Schrägstriche fallen aus der Kopfzeile, ohne unnötige
	// Zwischenzeichenketten.
	var b strings.Builder
	b.Grow(len(roh))
	for i := 0; i < len(roh); i++ {
		c := roh[i]
		if c != ' ' && c != '_' && c != '-' && c != '/' {
			b.WriteByte(c)
		}
	}
	name := b.String()

	if strings.Contains(name, "isbn") {
		return "isbn"
	}
	if strings.Contains(name, "titel") || strings.Contains(name, "title") || strings.Contains(name, "band") || strings.Contains(name, "ausgabe") {
		return "titel"
	}
	if strings.Contains(name, "autor") || strings.Contains(name, "author") {
		return "autor"
	}
	if strings.Contains(name, "fach") || strings.Contains(name, "subject") {
		return "fach"
	}
	if strings.Contains(name, "klasse") || strings.Contains(name, "stufe") || strings.Contains(name, "grade") {
		return "klasse"
	}
	if strings.Contains(name, "bestand") || strings.Contains(name, "anzahl") || strings.Contains(name, "stock") || strings.Contains(name, "menge") || strings.Contains(name, "stueck") || strings.Contains(name, "stück") {
		return "bestand"
	}

	return ""
}

// importZeile bündelt die Parameter für die Verarbeitung einer Importzeile.
type importZeile struct {
	Zeile     []string
	Spalten   map[string]int
	Metadaten *katalogdienste.MetadatenClient
}

// verarbeiteImportZeile verarbeitet eine einzelne Excel-Zeile und erstellt das
// entsprechende Book-Objekt für den Batch-Import.
func verarbeiteImportZeile(ctx context.Context, konfig importZeile) (*repository.KatalogTitel, error) {
	spalteLesen := func(name string) string {
		stelle := konfig.Spalten[name]
		if stelle >= 0 && stelle < len(konfig.Zeile) {
			return strings.TrimSpace(konfig.Zeile[stelle])
		}
		return ""
	}

	isbn := spalteLesen("isbn")
	if isbn == "" {
		return nil, nil // Leere Zeile überspringen
	}

	titelText := spalteLesen("titel")
	autor := spalteLesen("autor")
	// Die Spalte des Fachs ist Freitext, buecher_titel.subject aber ein Fremdschlüssel auf die
	// Systematik: Was hier durchgeht, wird eine Kategorie und steht danach in der Auswahl der Fächer
	// in der Maske und als Fach im Portal. Deshalb dieselbe Prüfung wie im Import des Bestands aus
	// CSV (fachDerZeile): nur ein echter Fachname oder eine bekannte Schreibvariante, sonst leer.
	//
	// Ein Rückfall auf das wörtliche Fach „Unbekannt" legte eine Kategorie dieses Namens an, die kein
	// Fach ist. Leer heißt überall sonst „ohne Fach", und genau das ist hier gemeint.
	fach := lmf.FachExakt(spalteLesen("fach"))

	// Die Spalte „klasse" nennt einen Jahrgang. Er steht als Spanne N bis N am Titel: Mit ihr
	// rechnen Inventur und Portal, und die Titelliste zeigt sie. 0 heißt unbekannt.
	klasse := leseKlassenstufe(spalteLesen("klasse"))

	eintrag := repository.KatalogTitel{
		ISBN:            isbn,
		Titel:           titelText,
		Autor:           autor,
		Fach:            fach,
		JahrgangVon:     klasse,
		JahrgangBis:     klasse,
		Bestand:         leseBestand(spalteLesen("bestand")),
		ZuletztGezaehlt: nil,
	}

	ergaenzeMetadaten(ctx, konfig.Metadaten, &eintrag)

	if eintrag.Fach == "" || strings.EqualFold(eintrag.Fach, "unbekannt") {
		if erratenesFach := errateFachAusTitel(eintrag.Titel); erratenesFach != "" {
			eintrag.Fach = erratenesFach
		}
	}

	return &eintrag, nil
}

// leseKlassenstufe liest den Jahrgang aus der Spalte „klasse" (lmf.IstJahrgang). Fehlt die
// Spalte, ist sie leer oder liegt der Wert daneben, ist der Jahrgang unbekannt (0), und beide
// Upserts schreiben NULL in die Spanne. Geraten wird nicht: Ein geratener Jahrgang ist in der
// Datenbank von einem gepflegten nicht zu unterscheiden, und das Upsert behält eine vorhandene
// Spanne.
func leseKlassenstufe(stufeText string) int {
	stufe, err := strconv.Atoi(stufeText)
	if err != nil || !lmf.IstJahrgang(stufe) {
		return 0
	}
	return stufe
}

// leseBestand liest den Bestand aus der Import-Spalte. Nur Werte aus [0, MaxInt32]
// werden übernommen: negative Bestände sind Datenfehler, und die DB-Spalte ist int4 —
// der Bulk-Upsert konvertiert nach int32. Ungültiges wird wie ein Parse-Fehler
// behandelt (Bestand 0).
func leseBestand(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > math.MaxInt32 {
		return 0
	}
	return n
}

// ergaenzeMetadaten ergänzt fehlende Buch-Metadaten über externe APIs.
func ergaenzeMetadaten(ctx context.Context, metadaten *katalogdienste.MetadatenClient, eintrag *repository.KatalogTitel) {
	if eintrag.Titel != "" && eintrag.Autor != "" && eintrag.CoverURL != "" {
		return
	}

	nachschlagen, _ := metadaten.SucheNachISBN(ctx, eintrag.ISBN) //nolint:errcheck
	if nachschlagen == nil {
		return
	}

	if eintrag.Titel == "" || eintrag.Titel == "Unbekannter Titel" {
		eintrag.Titel = nachschlagen.Titel
	}
	if eintrag.Autor == "" || eintrag.Autor == "Unbekannter Autor" {
		eintrag.Autor = nachschlagen.Autor
	}
	if eintrag.CoverURL == "" {
		eintrag.CoverURL = nachschlagen.CoverURL
	}

	if eintrag.Titel == "" {
		eintrag.Titel = "Unbekannter Titel"
	}
	if eintrag.Autor == "" {
		eintrag.Autor = "Unbekannter Autor"
	}
}
