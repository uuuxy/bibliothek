package inventur

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"bibliothek/pkg/coverdatei"

	"github.com/jung-kurt/gofpdf"
)

// Der Schulbuch-Export ist ein PDF, kein Excel (Peter, 03.09.2026): „es rechnet niemand,
// also können wir Excel löschen." Entscheidend war das Coverbild — in einer .xlsx schwebt
// ein Bild über der Zelle statt darin, verrutscht beim Sortieren und bläht die Datei.
// Im PDF steht es fest an seiner Zeile, druckt sauber und öffnet sich ohne Excel.
//
// Gezeigt wird, was der Reiter zeigt: Bild, Titel, Autor, ISBN, Jahrgang, Schulzweig,
// Gesamt, Verliehen, Verfügbar.

// Spaltenbreiten in mm. Die Summe ist immer 178 = A4-Breite (210) minus zweimal 16 mm
// Rand; die Titelspalte nimmt auf, was die optionale Zählspalte übrig lässt.
const (
	spCover    = 12.0
	spAutor    = 24.0
	spISBN     = 23.0
	spJg       = 12.0
	spZweig    = 21.0
	spGezaehlt = 17.0
	spZahl     = 8.0
	nutzBreite = 178.0
	randLinks  = 16.0
	zeilenH    = 17.0
	coverH     = 15.0
	// coverBrt ist nur die ANGENOMMENE Breite fürs Zentrieren (Cover sind 2:3); die
	// echte Breite bestimmt gofpdf beim Skalieren über die Höhe.
	coverBrt = 10.0
)

// titelBreite füllt den Rest der Zeile.
func titelBreite(mitGezaehlt bool) float64 {
	belegt := spCover + spAutor + spISBN + spJg + spZweig + 3*spZahl
	if mitGezaehlt {
		belegt += spGezaehlt
	}
	return nutzBreite - belegt
}

// SchulbuecherAlsPDF baut die Bestandsliste. fachName ist die Überschrift („Englisch"
// oder "" für alle Fächer), zusatz die aktive Einschränkung als Klartext.
func SchulbuecherAlsPDF(titel []LernmittelTitel, fachName, zusatz string) ([]byte, error) {
	// Die Zählspalte erscheint nur, wenn in dieser Auswahl überhaupt etwas gezählt wurde
	// (Peter, 03.09.2026). Eine Spalte, die auf jedem Blatt leer bliebe, nähme der
	// Titelspalte 17 mm weg und behauptete zugleich eine Angabe, die es nicht gibt.
	mitGezaehlt := false
	for _, t := range titel {
		if t.Gezaehlt != "" {
			mitGezaehlt = true
			break
		}
	}
	breiteTitel := titelBreite(mitGezaehlt)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(randLinks, 16, randLinks)
	pdf.SetAutoPageBreak(true, 16)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	ueberschrift := "Schulbücher"
	if fachName != "" {
		ueberschrift += " · " + fachName
	}
	// Die Kopfzeile wiederholt sich auf jeder Seite — eine Bestandsliste wird ausgedruckt
	// und weitergereicht, und Blatt 3 ohne Spaltennamen ist wertlos.
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 9, tr(ueberschrift))
		pdf.Ln(7)
		pdf.SetFont("Arial", "", 9)
		pdf.SetTextColor(120, 120, 120)
		hinweis := fmt.Sprintf("%d Titel · Stand %s", len(titel), time.Now().Format("02.01.2006"))
		if zusatz != "" {
			hinweis = zusatz + " · " + hinweis
		}
		pdf.Cell(0, 5, tr(hinweis))
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(9)
		zeichneKopfzeile(pdf, tr, breiteTitel, mitGezaehlt)
	})

	pdf.AddPage()
	// Beim Export über alle Fächer trennt eine Zwischenüberschrift die Fächer, statt das
	// Fach in jede Zeile zu schreiben: Das spart der Titelspalte 21 mm, und auf Papier
	// findet man das eigene Fach so schneller. Die Abfrage liefert nach Fach sortiert.
	letztesFach := "\x00"
	for _, t := range titel {
		if fachName == "" && t.Subject != letztesFach {
			letztesFach = t.Subject
			zeichneFachTrenner(pdf, tr, fachAnzeige(t.Subject))
		}
		zeichneSchulbuchZeile(pdf, tr, t, breiteTitel, mitGezaehlt)
	}
	if len(titel) == 0 {
		pdf.SetFont("Arial", "I", 10)
		pdf.Ln(4)
		pdf.Cell(0, 6, tr("Keine Schulbücher in dieser Auswahl."))
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("schulbuecher-pdf: %w", err)
	}
	return out.Bytes(), nil
}

// zeichneKopfzeile schreibt die Spaltennamen.
func zeichneKopfzeile(pdf *gofpdf.Fpdf, tr func(string) string, breiteTitel float64, mitGezaehlt bool) {
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(226, 232, 240)
	pdf.CellFormat(spCover, 7, "", "1", 0, "C", true, 0, "")
	pdf.CellFormat(breiteTitel, 7, tr("Titel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(spAutor, 7, tr("Autor"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(spISBN, 7, tr("ISBN"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(spJg, 7, tr("Jahrg."), "1", 0, "C", true, 0, "")
	pdf.CellFormat(spZweig, 7, tr("Schulzweig"), "1", 0, "L", true, 0, "")
	if mitGezaehlt {
		pdf.CellFormat(spGezaehlt, 7, tr("Gezählt"), "1", 0, "C", true, 0, "")
	}
	pdf.CellFormat(spZahl, 7, tr("Ges."), "1", 0, "R", true, 0, "")
	pdf.CellFormat(spZahl, 7, tr("Verl."), "1", 0, "R", true, 0, "")
	pdf.CellFormat(spZahl, 7, tr("Verf."), "1", 1, "R", true, 0, "")
}

// zeichneFachTrenner setzt eine Zwischenüberschrift über die Bücher eines Fachs.
func zeichneFachTrenner(pdf *gofpdf.Fpdf, tr func(string) string, fach string) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(241, 245, 249)
	pdf.CellFormat(nutzBreite, 7, tr(" "+fach), "1", 1, "L", true, 0, "")
}

// zeichneSchulbuchZeile rendert eine Zeile samt Cover. Fehlt das Bild, bleibt die
// Rahmenzelle leer — die Zeile darf deshalb nicht ausfallen.
func zeichneSchulbuchZeile(pdf *gofpdf.Fpdf, tr func(string) string, t LernmittelTitel, breiteTitel float64, mitGezaehlt bool) {
	oben := pdf.GetY()
	bindeCoverEin(pdf, t.CoverURL, randLinks+(spCover-coverBrt)/2, oben+(zeilenH-coverH)/2)

	pdf.SetFont("Arial", "", 8)
	pdf.SetXY(randLinks, oben)
	pdf.CellFormat(spCover, zeilenH, "", "1", 0, "", false, 0, "")
	pdf.CellFormat(breiteTitel, zeilenH, tr(kuerze(t.Title, int(breiteTitel/1.6))), "1", 0, "L", false, 0, "")
	pdf.CellFormat(spAutor, zeilenH, tr(kuerze(t.Autor, 16)), "1", 0, "L", false, 0, "")
	pdf.CellFormat(spISBN, zeilenH, tr(t.ISBN), "1", 0, "L", false, 0, "")
	pdf.CellFormat(spJg, zeilenH, tr(jahrgangText(t)), "1", 0, "C", false, 0, "")
	pdf.CellFormat(spZweig, zeilenH, tr(kuerze(t.Track, 13)), "1", 0, "L", false, 0, "")
	if mitGezaehlt {
		pdf.CellFormat(spGezaehlt, zeilenH, tr(t.Gezaehlt), "1", 0, "C", false, 0, "")
	}
	pdf.CellFormat(spZahl, zeilenH, fmt.Sprint(t.Gesamt), "1", 0, "R", false, 0, "")
	pdf.CellFormat(spZahl, zeilenH, fmt.Sprint(t.Verliehen), "1", 0, "R", false, 0, "")
	pdf.CellFormat(spZahl, zeilenH, fmt.Sprint(t.Verfuegbar), "1", 1, "R", false, 0, "")
}

// bindeCoverEin bettet das lokale Cover als JPEG ein. Alle Fehler bleiben still: Ein
// defektes Cover darf nie die ganze Liste kosten (die Lehre aus dem Mahnwesen, wo ein
// einziges WebP den Fehlerzustand des PDF-Objekts setzte und den Lauf mit 500 beendete).
// gofpdf hält registrierte Bilder unter ihrem Namen vor — derselbe Titel in zwei Fächern
// wird nur einmal dekodiert.
func bindeCoverEin(pdf *gofpdf.Fpdf, coverURL string, x, y float64) {
	opt := gofpdf.ImageOptions{ImageType: "JPG"}
	pfad := coverdatei.Pfad(coverURL)
	if pfad == "" {
		return
	}
	if pdf.GetImageInfo(pfad) == nil {
		jpg, _, ok := coverdatei.AlsJPEG(coverURL)
		if !ok {
			return
		}
		pdf.RegisterImageOptionsReader(pfad, opt, bytes.NewReader(jpg))
	}
	// Nur die Höhe vorgeben, Breite 0: gofpdf skaliert dann seitenverhältnistreu. Mit
	// beiden Maßen wurde jedes Cover auf 10×15 mm gequetscht — bei einem Buchrücken
	// sieht man das sofort.
	pdf.ImageOptions(pfad, x, y, 0, coverH, false, opt, 0, "")
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

func fachAnzeige(fach string) string {
	if fach == "" {
		return "ohne Fach"
	}
	return fach
}

func kuerze(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max-1]) + "…"
}

// dateinamenTeil macht aus einem Fachnamen einen Dateinamen-Bestandteil.
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
	name := strings.Trim(b.String(), "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	if name == "" {
		return "ohne-fach"
	}
	return name
}
