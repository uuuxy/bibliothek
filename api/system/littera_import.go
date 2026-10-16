package system

import (
	"bibliothek/pkg/xlsxgrenze"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/closeutil"
	"bibliothek/repository"

	"github.com/xuri/excelize/v2"
)

// Stelle im Log, wenn das Schließen der hochgeladenen Datei scheitert.
const litteraImportQuelle = "littera import file"

// LitteraImportAntwort meldet zurück, was ein Littera-Import bewegt hat. Die Zähler
// tragen omitempty, weil je nach Dateiart nur ein Teil davon gefüllt wird: Das
// Katalogisat-XML bringt Titel, die Bestands-CSV bringt Exemplare.
type LitteraImportAntwort struct {
	NeueTitel            int    `json:"new_titles_count,omitempty"`
	ImportierteExemplare int    `json:"imported_copies_count,omitempty"`
	GeaenderteTitel      int    `json:"updated_titles_count,omitempty"`
	Art                  string `json:"type"` // "csv", "xml" or "xlsx"
}

// leseUploadInhalt begrenzt die Anfrage, zerlegt die Mehrteil-Form und liest die
// hochgeladene Datei. Bei Fehler ist schon geantwortet, und ok ist false.
func (t *Tueren) leseUploadInhalt(w http.ResponseWriter, r *http.Request, maxBytes int64) (inhalt []byte, dateiname string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungHochladen(maxBytes), err)
		return nil, "", false
	}

	datei, dateiKopf, err := r.FormFile("file")
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungKeineDatei, err)
		return nil, "", false
	}
	defer closeutil.Schliesse(datei, litteraImportQuelle)

	inhalt, err = io.ReadAll(datei)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return nil, "", false
	}
	return inhalt, dateiKopf.Filename, true
}

// istLitteraXML erkennt ein MAB2-XML-Katalogisat an Endung oder Inhalt.
func istLitteraXML(dateiname, inhaltText string) bool {
	nameKlein := strings.ToLower(dateiname)
	inhaltKlein := strings.ToLower(inhaltText)
	return strings.HasSuffix(nameKlein, ".xml") ||
		strings.Contains(inhaltKlein, "<?xml") ||
		strings.Contains(inhaltKlein, "<katalogisat")
}

// leseTabellarischeDaten liest die Zeilen aus einer XLSX- oder CSV-Datei.
func leseTabellarischeDaten(dateiname string, inhalt []byte, inhaltText string) (zeilen [][]string, istXLSX bool, err error) {
	if strings.HasSuffix(strings.ToLower(dateiname), ".xlsx") {
		zeilen, err := xlsxgrenze.LeseMappe(bytes.NewReader(inhalt), func(f *excelize.File) ([][]string, error) {
			blatt := f.GetSheetName(f.GetActiveSheetIndex())
			return f.GetRows(blatt, excelize.Options{RawCellValue: true})
		})
		switch {
		case xlsxgrenze.IstUnlesbar(err):
			return nil, true, apierrors.BadRequest("Die Excel-Datei ließ sich nicht öffnen. Bitte als .xlsx oder CSV neu speichern.", err)
		case err != nil:
			return nil, true, apierrors.BadRequest("Die Excel-Datei ließ sich nicht lesen", err)
		}
		return zeilen, true, nil
	}

	csvLeser := csv.NewReader(strings.NewReader(inhaltText))
	csvLeser.Comma = service.ImportTrennzeichen(inhaltText)
	csvLeser.LazyQuotes = true
	zeilen, err = csvLeser.ReadAll()
	if err != nil {
		return nil, false, apierrors.BadRequest(csvLesefehlerMeldung(err), err)
	}
	return zeilen, false, nil
}

// importKopfMitPflichtspalten baut die Spaltenzuordnung und prüft die
// Pflichtspalten (Titel, Barcode).
func importKopfMitPflichtspalten(zeilen [][]string) (map[string]int, error) {
	if len(zeilen) < 1 {
		return nil, apierrors.BadRequest("Die Datei ist leer", nil)
	}
	kopf := service.ImportKopfzeile(zeilen[0])
	if _, ok := kopf["titel"]; !ok {
		return nil, apierrors.BadRequest("In der Kopfzeile fehlt die Pflichtspalte „titel“", nil)
	}
	if _, ok := kopf["barcode"]; !ok {
		return nil, apierrors.BadRequest("In der Kopfzeile fehlt die Pflichtspalte „barcode“ oder „exemplarnummer“", nil)
	}
	return kopf, nil
}

// protokolliereImport schreibt einen Eintrag ins Protokoll, wenn jemand angemeldet ist.
func (t *Tueren) protokolliereImport(r *http.Request, aktion, details string) {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		return
	}
	t.SchreibeAdminProtokoll(r.Context(), claims.KontoID, aktion, clientip.AusAnfrage(r), details)
}

// verarbeiteLitteraXML importiert ein MAB2-XML-Katalogisat und antwortet.
// Eine Datei, die kein Katalogisat ist (z. B. Schlagwort-/Systematik-Export),
// ist ein Nutzer-Formatfehler und wird als 400 gemeldet, nicht als 500.
func (t *Tueren) verarbeiteLitteraXML(w http.ResponseWriter, r *http.Request, inhalt []byte) {
	importDienst := service.NewImportDienst(repository.NewKatalogAbfragen(t.DB.Pool), t.DB.Pool)
	geaenderteTitel, err := importDienst.ImportiereLitteraXML(r.Context(), bytes.NewReader(inhalt))
	if err != nil {
		if errors.Is(err, service.ErrKeinKatalogisat) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Datei ist kein Katalogisat-Export aus "+
				"Littera. Erwartet wird MAB2-XML mit der Wurzel <Katalogisate>.", err)
			return
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	t.protokolliereImport(r, "LITTERA_IMPORT", fmt.Sprintf(`{"updated_titles":%d,"type":"xml"}`, geaenderteTitel))

	// Dieselbe typisierte Antwort wie CSV und XLSX: Mit einer losen Map bliebe GeaenderteTitel
	// ungesetzt, omitempty nähme den Zähler still weg, und die Oberfläche zeigte „undefined
	// bestehende Titel".
	tuer.AntworteJSON(w, http.StatusOK, LitteraImportAntwort{
		GeaenderteTitel: geaenderteTitel,
		Art:             "xml",
	})
}

// LitteraImport nimmt eine Littera-Ausgabedatei entgegen und verzweigt anhand der
// Dateiendung auf den passenden Leser (CSV, MAB2-XML, XLSX). Die Grenze von 100 MB ist
// bewusst großzügig: Das Katalogisat der Schule liegt bei rund 14.000 Titeln.
func (t *Tueren) LitteraImport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inhalt, dateiname, ok := t.leseUploadInhalt(w, r, 100<<20) // höchstens 100 MB
		if !ok {
			return
		}

		inhaltText := string(inhalt)
		if istLitteraXML(dateiname, inhaltText) {
			t.verarbeiteLitteraXML(w, r, inhalt)
			return
		}

		zeilen, istXLSX, err := leseTabellarischeDaten(dateiname, inhalt, inhaltText)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}

		kopf, err := importKopfMitPflichtspalten(zeilen)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}

		importDienst := service.NewImportDienst(repository.NewKatalogAbfragen(t.DB.Pool), t.DB.Pool)
		neueTitel, importierteExemplare, err := importDienst.ImportiereZeilen(r.Context(), zeilen, kopf)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		t.protokolliereImport(r, "LITTERA_IMPORT", fmt.Sprintf(`{"new_titles":%d,"imported_copies":%d}`, neueTitel, importierteExemplare))

		art := "csv"
		if istXLSX {
			art = "xlsx"
		}

		tuer.AntworteJSON(w, http.StatusOK, LitteraImportAntwort{
			NeueTitel:            neueTitel,
			ImportierteExemplare: importierteExemplare,
			Art:                  art,
		})
	}
}

// BestandImport verarbeitet den Upload der finalen Bestands-CSV.
//
// Er nutzt denselben robusten Pfad wie der Littera-Import (Trennzeichen-Erkennung,
// namensbasiertes Spalten-Mapping, Titel-Dedup über ImportiereZeilen). Dadurch ist es
// egal, ob die Datei Komma- oder Semikolon-getrennt ist bzw. die Zustand-Spalte
// enthält — ein Formatfehler des Nutzers wird als 400 (nicht 500) gemeldet.
func (t *Tueren) BestandImport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungHochladen(20<<20), err)
		return
	}

	datei, dateiKopf, err := r.FormFile("file")
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungKeineDatei, err)
		return
	}
	defer closeutil.Schliesse(datei, litteraImportQuelle)

	if !strings.HasSuffix(strings.ToLower(dateiKopf.Filename), ".csv") {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Nur CSV-Dateien werden angenommen", nil)
		return
	}

	inhalt, err := io.ReadAll(datei)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	inhaltText := string(inhalt)

	csvLeser := csv.NewReader(strings.NewReader(inhaltText))
	csvLeser.Comma = service.ImportTrennzeichen(inhaltText)
	csvLeser.LazyQuotes = true
	zeilen, err := csvLeser.ReadAll()
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, csvLesefehlerMeldung(err), err)
		return
	}
	if len(zeilen) < 2 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die CSV-Datei enthält keine Datenzeilen", nil)
		return
	}

	// Dieselbe Prüfung der Pflichtspalten wie beim Import aus Littera.
	kopf, err := importKopfMitPflichtspalten(zeilen)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		return
	}

	katalogAbfragen := repository.NewKatalogAbfragen(t.DB.Pool)
	importDienst := service.NewImportDienst(katalogAbfragen, t.DB.Pool)

	neueTitel, importierteExemplare, err := importDienst.ImportiereZeilen(r.Context(), zeilen, kopf)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	if claims, ok := auth.ClaimsAusKontext(r.Context()); ok {
		details := fmt.Sprintf(`{"new_titles":%d,"imported_copies":%d}`, neueTitel, importierteExemplare)
		t.SchreibeAdminProtokoll(r.Context(), claims.KontoID, "BESTAND_IMPORT", clientip.AusAnfrage(r), details)
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]any{
		"new_titles_count":      neueTitel,
		"imported_copies_count": importierteExemplare,
		"message":               "Bestands-CSV erfolgreich importiert",
	})
}

// meldungKeineDatei antwortet auf ein Formular ohne Datei.
const meldungKeineDatei = "Bitte eine Datei auswählen"

// meldungHochladen antwortet auf ein Formular, das sich nicht lesen ließ, meist eine zu große
// Datei; sie nennt die Grenze.
func meldungHochladen(maxBytes int64) string {
	return fmt.Sprintf("Die Datei ließ sich nicht hochladen. Möglich sind höchstens %d MB.", maxBytes>>20)
}

// csvLesefehlerMeldung nennt die Zeile, an der das Lesen der CSV-Datei scheiterte; der Text von
// encoding/csv geht nur ins Log.
func csvLesefehlerMeldung(err error) string {
	var lesefehler *csv.ParseError
	if errors.As(err, &lesefehler) {
		return fmt.Sprintf("Die CSV-Datei ließ sich nicht lesen, Zeile %d", lesefehler.Line)
	}
	return "Die CSV-Datei ließ sich nicht lesen"
}
