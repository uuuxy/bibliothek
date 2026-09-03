package inventur

import (
	"fmt"
	"net/http"
	"strings"

	"bibliothek/pkg/csvutil"

	"github.com/xuri/excelize/v2"
)

// SchulbuecherAlsExcel baut das Blatt: Kopfzeile fett, Spalten Fach, Titel, Autor, ISBN,
// Gesamt, Verliehen, Verfügbar, Cover (Link). Text-Zellen gehen durch SanitizeCell —
// Titel und Autor stammen aus Importen und dürfen in Excel keine Formel auslösen.
func SchulbuecherAlsExcel(titel []LernmittelTitel, coverBasis string) (*excelize.File, error) {
	f := excelize.NewFile()
	const blatt = "Schulbücher"
	if err := f.SetSheetName("Sheet1", blatt); err != nil {
		return nil, err
	}
	kopf := []any{"Fach", "Titel", "Autor", "ISBN", "Jahrgang", "Schulzweig", "Gesamt", "Verliehen", "Verfügbar", "Cover"}
	if err := f.SetSheetRow(blatt, "A1", &kopf); err != nil {
		return nil, err
	}
	fett, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	if err := f.SetCellStyle(blatt, "A1", "J1", fett); err != nil {
		return nil, err
	}
	for i, t := range titel {
		fach := t.Subject
		if fach == "" {
			fach = "ohne Fach"
		}
		zeile := []any{csvutil.SanitizeCell(fach), csvutil.SanitizeCell(t.Title), csvutil.SanitizeCell(t.Autor),
			csvutil.SanitizeCell(t.ISBN), jahrgangText(t), csvutil.SanitizeCell(t.Track),
			t.Gesamt, t.Verliehen, t.Verfuegbar, coverLink(coverBasis, t)}
		if err := f.SetSheetRow(blatt, fmt.Sprintf("A%d", i+2), &zeile); err != nil {
			return nil, err
		}
	}
	for spalte, breite := range map[string]float64{"A": 16, "B": 48, "C": 24, "D": 16, "F": 14, "J": 40} {
		if err := f.SetColWidth(blatt, spalte, spalte, breite); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// coverLink: ein Link statt eines eingebetteten Bilds — Bilder machten die Datei groß,
// der Link öffnet das Cover über den Proxy der Anwendung.
func coverLink(basis string, t LernmittelTitel) string {
	if t.CoverURL == "" {
		return ""
	}
	if strings.HasPrefix(t.CoverURL, "/") {
		return basis + t.CoverURL
	}
	return basis + "/api/images/cover?isbn=" + t.ISBN + "&url=" + t.CoverURL
}

func coverBasis(r *http.Request) string {
	schema := "https"
	if r.TLS == nil && !strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		schema = "http"
	}
	return schema + "://" + r.Host
}

func dateinamenTeil(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b.WriteRune(c)
		case c == 'ä':
			b.WriteString("ae")
		case c == 'ö':
			b.WriteString("oe")
		case c == 'ü':
			b.WriteString("ue")
		case c == 'ß':
			b.WriteString("ss")
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// jahrgangText: „7", „12–13"; die Spalten-Vorgabe 5–10 (= unbekannt) und 0 bleiben leer.
func jahrgangText(t LernmittelTitel) string {
	if t.JahrgangVon == 0 || (t.JahrgangVon == 5 && t.JahrgangBis == 10) {
		return ""
	}
	if t.JahrgangVon == t.JahrgangBis {
		return fmt.Sprint(t.JahrgangVon)
	}
	return fmt.Sprintf("%d–%d", t.JahrgangVon, t.JahrgangBis)
}
