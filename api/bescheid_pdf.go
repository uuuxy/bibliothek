package api

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"bibliothek/pdf"

	"github.com/jung-kurt/gofpdf"
)

// Der Schadensersatz-Bescheid als PDF — der Nachbau des Musteranschreibens, das die
// Schule vorgelegt hat (Wortlaut aus der Vorlage, Layout nach dem Original).
//
// Warum wörtlich und fest im Code: Der Brief ist ein Bescheid. Eine frei editierbare
// Vorlage könnte einen Satz verlieren, ohne dass es auffällt — und ein Bescheid ohne
// seinen Schlussabsatz ist fehlerhaft. Deshalb steht der Text an EINER Stelle hier;
// austauschbar ist er trotzdem, es ist eine Konstante je Absatz.
//
// Bewusst NICHT übernommen: die Ausfüllhilfe unter der Referenznummer („SSANr.
// (4stellig)+ Jahr (4stellig) …"). Sie erklärt dem Ausfüller, wie die Nummer entsteht,
// nicht dem Empfänger, wie er zahlt — im fertigen Brief wäre sie ein Fremdkörper.
//
// Zwei Fallgruppen, wie im Formular: nicht zurückgegeben und so stark beschädigt, dass
// eine Nutzung nicht möglich ist. Gezeigt wird eine Gruppe nur, wenn sie Positionen hat;
// ein leeres Kästchen mit leerer Tabelle wäre ein unausgefülltes Formular, kein Brief.

// Maße nach DIN 5008 Form A, wie beim Eltern-Mahnbrief (reports_pdf.go).
const (
	bescheidRandLinks  = 25.0
	bescheidRandRechts = 20.0
	bescheidBreite     = 210.0 - bescheidRandLinks - bescheidRandRechts
	// Anschriftfeld und Infoblock beginnen auf derselben Höhe.
	bescheidYAnschrift = 45.0
	bescheidXInfoblock = 125.0
	// Der Betreff sitzt unter dem Anschriftfeld (Form A: 98,46 mm).
	bescheidYBetreff = 98.0
)

// Der Wortlaut. Jede Konstante ist ein Absatz der Vorlage.
const (
	bescheidBetreff = "Aushändigung von Lernmaterial im Rahmen des öffentlich-rechtlichen Nutzungsverhältnisses/ " +
		"Schadenersatzforderung entsprechend §§ 823 Abs. 1, 832 Abs. 1 BGB wegen Verletzung von Pflichten " +
		"des öffentlich-rechtlichen Nutzungsverhältnisses"
	bescheidAbsatzEigentum = "die Schule hat Ihnen/Ihrem Kind im Rahmen der Lernmittelfreiheit Lernmittel zur " +
		"Verfügung gestellt. Gemäß § 153 Abs. 2 Hessisches Schulgesetz (HSchG) bleiben die Lernmittel im " +
		"Eigentum des Landes Hessen. Sie sind pfleglich zu behandeln und nach Ablauf der Verwendungszeit " +
		"wieder zurückzugeben."
	bescheidAbsatzHaftung = "Bei Verlust oder Beschädigung haften die Schülerinnen und Schüler oder ihre Eltern, " +
		"§ 153 Abs.2, 3 HSchG i.V.m. § 9 der Verordnung über die Durchführung der Lernmittelfreiheit."
	bescheidEinleitung    = "Sie haben/Ihr Kind hat die Lernmittel"
	bescheidKastenOffen   = "nicht ordnungsgemäß zurückgegeben."
	bescheidKastenSchaden = "so stark beschädigt zurückgegeben, dass eine Nutzung nicht mehr möglich ist."
	bescheidBetroffen     = "Folgende Lernmittel sind betroffen:"
	bescheidBitteOffen    = "Ich bitte um Rückgabe entsprechend § 985 BGB oder soweit Sie oder Ihr Kind nicht mehr " +
		"dazu in der Lage sein sollten, um Ersatzbeschaffung oder Erstattung des Wiederbeschaffungspreises, " +
		"entsprechend § 823 Abs. 1, § 832 BGB bis zum %s."
	bescheidBitteSchaden = "Ich bitte um Ersatzbeschaffung oder Erstattung des Wiederbeschaffungspreises, " +
		"entsprechend §§ 823 Abs. 1, 832 BGB bis zum %s."
	bescheidZahlsatz  = "Der Gesamtbetrag i.H.v. %s ist auf das Konto %s zu zahlen:"
	bescheidReferenz  = "unter Angabe der folgenden Referenz-Nr. "
	bescheidAndrohung = "Sollte bis zum oben genannten Zeitpunkt keine oder nur eine teilweise Rückgabe oder " +
		"Ersatzbeschaffung erfolgt sein oder Schadenersatzpflicht nicht geleistet worden sein, werde ich die " +
		"Angelegenheit dem %s übergeben, das den Schadensersatzanspruch im Wege des " +
		"Verwaltungsvollstreckungsverfahrens gegen Sie verfolgen wird."
	bescheidGruss       = "Mit freundlichen Grüßen"
	bescheidRechtsTitel = "Rechtsbehelfsbelehrung"
	bescheidRechtsText  = "Gegen diesen Bescheid kann innerhalb eines Monats nach Bekanntgabe Widerspruch erhoben " +
		"werden. Der Widerspruch ist bei der %s schriftlich oder zur Niederschrift einzulegen. Die Frist wird " +
		"auch durch Einlegung bei dem %s als Behörde, die den Widerspruchsbescheid zu erlassen hat, gewahrt."
	// Die Zeile über dem Namen im Anschriftfeld, wenn der Schuldner minderjährig ist.
	bescheidAnAnErzieher = "An die Erziehungsberechtigten des/der Schülers/in"
)

// BescheidPosition ist eine Zeile der Tabelle: ein Buch mit seinem Betrag.
type BescheidPosition struct {
	SchuelerName string
	Titel        string
	ISBN         string
	Betrag       float64
}

// BescheidEmpfaenger ist die Anschrift zum Briefdatum (Snapshot).
type BescheidEmpfaenger struct {
	// Anrede: die Zeile nach dem Betreff, z. B. „Sehr geehrte Erziehungsberechtigte,".
	Anrede string
	// AnZeile: die Zeile über dem Namen im Anschriftfeld; leer bei volljährigen Schuldnern.
	AnZeile string
	Name    string
	Strasse string
	PLZ     string
	Ort     string
}

// BescheidBrief bündelt alles, was auf dem Blatt steht.
type BescheidBrief struct {
	Schule            pdf.SchuleInfo
	Empfaenger        BescheidEmpfaenger
	Geschaeftszeichen string
	Bearbeiter        string
	Durchwahl         string
	BriefDatum        time.Time
	FristBis          time.Time
	// Die beiden Fallgruppen des Formulars.
	NichtZurueckgegeben []BescheidPosition
	Beschaedigt         []BescheidPosition
	Gesamtbetrag        float64
	Zahlstelle          string
	Bankverbindung      string
	Referenznummer      string
	Aufsicht            string
	Schulleitung        string
}

// GenerateBescheidPDF zeichnet den Bescheid.
func GenerateBescheidPDF(b BescheidBrief) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	roh := p.UnicodeTranslatorFromDescriptor("")
	// EIN Ort für die Zeichenersetzung: Jede Zeichenkette, die auf das Papier geht, läuft
	// hier durch. Ohne sie kam „Ayşe" als „Ay.e" aus dem Drucker — cp1252 kennt das ş
	// nicht (dieselbe Falle wie beim Schüler-Etikett, siehe cp1252Ersatz). Ein Bescheid
	// mit entstelltem Namen ist ein fehlerhafter Bescheid.
	tr := func(text string) string { return roh(cp1252Ersatz.Replace(text)) }
	// Ab der zweiten Seite die Seitenzahl mittig oben, wie im Original („- 2 -").
	p.SetHeaderFunc(func() {
		if p.PageNo() < 2 {
			return
		}
		p.SetY(15)
		p.SetFont("Times", "", 11)
		p.CellFormat(0, 6, fmt.Sprintf("- %d -", p.PageNo()), "", 1, "C", false, 0, "")
		p.SetY(25)
	})
	p.SetMargins(bescheidRandLinks, 20, bescheidRandRechts)
	p.SetAutoPageBreak(true, 20)
	p.AddPage()

	bescheidKopf(p, tr, b)
	bescheidFallgruppen(p, tr, b)
	bescheidZahlung(p, tr, b)
	bescheidSchluss(p, tr, b)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// bescheidKopf zeichnet Schulname, Anschriftfeld, Infoblock, Betreff, Anrede und die
// beiden einleitenden Absätze.
func bescheidKopf(p *gofpdf.Fpdf, tr func(string) string, b BescheidBrief) {
	p.SetFont("Arial", "B", 11)
	p.SetXY(bescheidRandLinks, 20)
	p.Cell(0, 6, tr(b.Schule.Name))

	// Absenderzeile klein über dem Anschriftfeld (Fensterkuvert).
	p.SetFont("Arial", "", 7)
	p.SetXY(bescheidRandLinks, bescheidYAnschrift-5)
	p.Cell(0, 4, tr(b.Schule.Absenderzeile()))

	// Anschriftfeld.
	p.SetFont("Times", "", 11)
	p.SetXY(bescheidRandLinks, bescheidYAnschrift)
	for _, zeile := range bescheidAnschriftZeilen(b.Empfaenger) {
		p.SetX(bescheidRandLinks)
		p.CellFormat(85, 5.5, tr(zeile), "", 1, "L", false, 0, "")
	}

	bescheidInfoblock(p, tr, b)

	// Betreff.
	p.SetXY(bescheidRandLinks, bescheidYBetreff)
	p.SetFont("Times", "B", 11)
	p.MultiCell(bescheidBreite, 5.5, tr(bescheidBetreff), "", "L", false)
	p.Ln(6)

	p.SetFont("Times", "", 11)
	bescheidAbsatz(p, tr, b.Empfaenger.Anrede)
	bescheidAbsatz(p, tr, bescheidAbsatzEigentum)
	bescheidAbsatz(p, tr, bescheidAbsatzHaftung)
}

// bescheidAnschriftZeilen baut das Anschriftfeld. Fehlt die Anschrift, sagt der Brief
// das an der Stelle, an der sie stehen müsste — ein Bescheid ohne Empfänger darf nicht
// aussehen wie ein vollständiger (dieselbe Regel wie beim Eltern-Mahnbrief).
func bescheidAnschriftZeilen(e BescheidEmpfaenger) []string {
	zeilen := make([]string, 0, 4)
	if e.AnZeile != "" {
		zeilen = append(zeilen, e.AnZeile)
	}
	zeilen = append(zeilen, e.Name)
	if strings.TrimSpace(e.Strasse) == "" || strings.TrimSpace(e.Ort) == "" {
		return append(zeilen, "(keine Adresse hinterlegt)")
	}
	return append(zeilen, e.Strasse, strings.TrimSpace(e.PLZ+" "+e.Ort))
}

// bescheidInfoblock setzt Geschäftszeichen, Bearbeiter, Durchwahl und Datum rechts neben
// das Anschriftfeld. Leere Angaben lassen ihre Zeile weg — im Muster sind es Formularfelder.
func bescheidInfoblock(p *gofpdf.Fpdf, tr func(string) string, b BescheidBrief) {
	p.SetFont("Arial", "", 9)
	y := bescheidYAnschrift
	for _, paar := range [][2]string{
		{"Geschäftszeichen", b.Geschaeftszeichen},
		{"Bearbeiter", b.Bearbeiter},
		{"Durchwahl", b.Durchwahl},
	} {
		if paar[1] == "" {
			continue
		}
		p.SetXY(bescheidXInfoblock, y)
		p.Cell(30, 4.5, tr(paar[0]))
		p.SetXY(bescheidXInfoblock+30, y)
		p.Cell(35, 4.5, tr(paar[1]))
		y += 4.5
	}
	// Das Datum steht mit Abstand darunter, wie im Original.
	p.SetXY(bescheidXInfoblock, y+4.5)
	p.Cell(30, 4.5, tr("Datum"))
	p.SetXY(bescheidXInfoblock+30, y+4.5)
	p.Cell(35, 4.5, b.BriefDatum.Format(dateFormatDE))
}

// bescheidFallgruppen zeichnet die zutreffenden Kästchen samt Tabelle und Fristsatz.
func bescheidFallgruppen(p *gofpdf.Fpdf, tr func(string) string, b BescheidBrief) {
	frist := b.FristBis.Format(dateFormatDE)
	gruppen := []struct {
		positionen []BescheidPosition
		kasten     string
		bitte      string
	}{
		{b.NichtZurueckgegeben, bescheidKastenOffen, fmt.Sprintf(bescheidBitteOffen, frist)},
		{b.Beschaedigt, bescheidKastenSchaden, fmt.Sprintf(bescheidBitteSchaden, frist)},
	}
	for _, g := range gruppen {
		if len(g.positionen) == 0 {
			continue
		}
		// Einleitung, Kästchen, „Folgende Lernmittel sind betroffen:", Kopfzeile und eine
		// Datenzeile gehören zusammen — sonst steht die Ankündigung am Fuß der einen und
		// die Tabelle am Kopf der nächsten Seite.
		bescheidPlatzOderNeueSeite(p, 3*7.5+bescheidKopfHoehe+bescheidZeileHoehe)
		bescheidAbsatz(p, tr, bescheidEinleitung)
		bescheidKaestchen(p, tr, g.kasten)
		bescheidAbsatz(p, tr, bescheidBetroffen)
		bescheidTabelle(p, tr, g.positionen)
		p.Ln(4)
		bescheidAbsatz(p, tr, g.bitte)
	}
}

// bescheidKaestchen zeichnet ein angekreuztes Kästchen mit Text daneben. Gezeichnet und
// nicht als Zeichen gesetzt: ☒ gibt es in der Zeichentabelle dieser Schriften nicht,
// es käme als Fragezeichen aufs Papier.
func bescheidKaestchen(p *gofpdf.Fpdf, tr func(string) string, text string) {
	const seite = 3.6
	x := bescheidRandLinks
	y := p.GetY() + 1
	p.Rect(x, y, seite, seite, "D")
	// Das Kreuz, zwei Linien von Ecke zu Ecke mit einem halben Millimeter Luft.
	p.Line(x+0.5, y+0.5, x+seite-0.5, y+seite-0.5)
	p.Line(x+seite-0.5, y+0.5, x+0.5, y+seite-0.5)
	p.SetFont("Times", "", 11)
	p.SetXY(x+seite+2.5, y-1)
	p.MultiCell(bescheidBreite-seite-2.5, 5.5, tr(text), "", "L", false)
	p.Ln(3)
}

// Maße der Tabelle. Die Spaltenbreiten folgen dem Original: Name, Titel, ISBN, Preis.
var bescheidSpalten = []float64{45, 60, 37, 23}

const (
	bescheidKopfHoehe  = 11.0
	bescheidZeileHoehe = 7.0
	// Ab hier ist die Seite voll (unterer Rand 20 mm).
	bescheidSeitenende = 297.0 - 20.0
)

// bescheidTabelle zeichnet die Tabelle der betroffenen Lernmittel — vier Spalten wie im
// Original: Name des/der Schülers/Schülerin, Titel, ISBN, Preis.
//
// Der Platz wird VOR jedem Block geprüft und die Kopfzeile auf einer neuen Seite
// wiederholt. Ohne das riss gofpdf die Kopfzeile mitten entzwei: „Name des/der" stand
// unten auf Seite 1, „Schülers/Schülerin" oben auf Seite 2, die Spaltenüberschriften
// dahinter am Fuß der Folgeseite. Ein zerrissener Bescheid ist ein fehlerhafter.
func bescheidTabelle(p *gofpdf.Fpdf, tr func(string) string, positionen []BescheidPosition) {
	// Kopfzeile plus mindestens eine Datenzeile gehören zusammen — eine Kopfzeile allein
	// am Seitenfuß ist nichts wert.
	bescheidPlatzOderNeueSeite(p, bescheidKopfHoehe+bescheidZeileHoehe)
	bescheidTabellenkopf(p, tr)

	for _, pos := range positionen {
		if bescheidPlatzOderNeueSeite(p, bescheidZeileHoehe) {
			// Neue Seite: Die Tabelle bekommt ihren Kopf wieder, sonst stehen dort
			// Zahlen ohne Überschrift.
			bescheidTabellenkopf(p, tr)
		}
		y := p.GetY()
		x := bescheidRandLinks
		werte := []string{pos.SchuelerName, pos.Titel, pos.ISBN, euroBetrag(pos.Betrag)}
		p.SetFont("Times", "", 10)
		for i, wert := range werte {
			p.Rect(x, y, bescheidSpalten[i], bescheidZeileHoehe, "D")
			p.SetXY(x+1.5, y+1.4)
			// Kürzen statt umbrechen: Die Zeilenhöhe ist fest, ein zu langer Titel würde
			// sonst über den Rahmen laufen.
			p.CellFormat(bescheidSpalten[i]-3, 4.2, tr(kuerzeAufBreite(p, tr, wert, bescheidSpalten[i]-3)), "", 0, "L", false, 0, "")
			x += bescheidSpalten[i]
		}
		p.SetY(y + bescheidZeileHoehe)
	}
}

// bescheidTabellenkopf zeichnet die Kopfzeile. Die erste Spalte ist zweizeilig; beide
// Zeilen werden EINZELN gesetzt statt per MultiCell — MultiCell darf umbrechen, und
// genau dieser Umbruch zerriss die Kopfzeile.
func bescheidTabellenkopf(p *gofpdf.Fpdf, tr func(string) string) {
	y := p.GetY()
	x := bescheidRandLinks
	p.SetFont("Times", "", 10)
	p.SetFillColor(217, 217, 217)
	kopf := [][2]string{
		{"Name des/der", "Schülers/Schülerin"},
		{"Titel", ""},
		{"ISBN", ""},
		{"Preis", ""},
	}
	for i, zeilen := range kopf {
		p.Rect(x, y, bescheidSpalten[i], bescheidKopfHoehe, "FD")
		p.SetXY(x+1.5, y+1.0)
		p.CellFormat(bescheidSpalten[i]-3, 4.4, tr(zeilen[0]), "", 0, "L", false, 0, "")
		if zeilen[1] != "" {
			p.SetXY(x+1.5, y+5.4)
			p.CellFormat(bescheidSpalten[i]-3, 4.4, tr(zeilen[1]), "", 0, "L", false, 0, "")
		}
		x += bescheidSpalten[i]
	}
	p.SetY(y + bescheidKopfHoehe)
}

// bescheidPlatzOderNeueSeite beginnt eine neue Seite, wenn der Block nicht mehr passt,
// und meldet, ob umgebrochen wurde.
func bescheidPlatzOderNeueSeite(p *gofpdf.Fpdf, hoehe float64) bool {
	if p.GetY()+hoehe <= bescheidSeitenende {
		return false
	}
	p.AddPage()
	return true
}

// bescheidZahlung zeichnet Gesamtbetrag, Zahlstelle, Bankverbindung und Referenznummer.
func bescheidZahlung(p *gofpdf.Fpdf, tr func(string) string, b BescheidBrief) {
	bescheidAbsatz(p, tr, fmt.Sprintf(bescheidZahlsatz, euroBetrag(b.Gesamtbetrag), b.Zahlstelle))
	p.SetFont("Times", "", 11)
	for _, zeile := range strings.Split(b.Bankverbindung, "\n") {
		p.SetX(bescheidRandLinks)
		p.CellFormat(bescheidBreite, 5.5, tr(strings.TrimSpace(zeile)), "", 1, "L", false, 0, "")
	}
	p.Ln(4)

	// „Referenz-Nr." fett, die Nummer daneben — sie ist die Angabe, an der die Zahlung
	// zugeordnet wird, und darf nicht im Fließtext untergehen.
	p.SetX(bescheidRandLinks)
	p.SetFont("Times", "", 11)
	p.Write(5.5, tr(bescheidReferenz))
	p.SetFont("Times", "B", 11)
	p.Write(5.5, tr(b.Referenznummer))
	p.Ln(9)
}

// bescheidSchluss zeichnet Androhung, Grußformel, Unterschrift und Rechtsbehelfsbelehrung.
func bescheidSchluss(p *gofpdf.Fpdf, tr func(string) string, b BescheidBrief) {
	bescheidAbsatz(p, tr, fmt.Sprintf(bescheidAndrohung, b.Aufsicht))
	bescheidAbsatz(p, tr, bescheidGruss)
	p.Ln(10)
	p.SetX(bescheidRandLinks)
	p.SetFont("Times", "", 11)
	p.CellFormat(bescheidBreite, 5.5, tr(b.Schulleitung), "", 1, "L", false, 0, "")
	p.Ln(8)

	p.SetX(bescheidRandLinks)
	p.SetFont("Times", "B", 11)
	p.CellFormat(bescheidBreite, 5.5, tr(bescheidRechtsTitel), "", 1, "L", false, 0, "")
	p.Ln(2)
	p.SetFont("Times", "", 11)
	bescheidAbsatz(p, tr, fmt.Sprintf(bescheidRechtsText, bescheidSchulanschriftEinzeilig(b.Schule), b.Aufsicht))
}

// bescheidSchulanschriftEinzeilig nennt die Schule mit Anschrift in einem Satzteil —
// dort wird der Widerspruch eingelegt.
func bescheidSchulanschriftEinzeilig(s pdf.SchuleInfo) string {
	teile := []string{s.Name}
	if s.Strasse != "" {
		teile = append(teile, s.Strasse)
	}
	if ort := strings.TrimSpace(s.PLZ + " " + s.Ort); ort != "" {
		teile = append(teile, ort)
	}
	return strings.Join(teile, ", ")
}

// bescheidAbsatz setzt einen Fließtext-Absatz mit Abstand darunter.
func bescheidAbsatz(p *gofpdf.Fpdf, tr func(string) string, text string) {
	if text == "" {
		return
	}
	p.SetX(bescheidRandLinks)
	p.SetFont("Times", "", 11)
	p.MultiCell(bescheidBreite, 5.5, tr(text), "", "L", false)
	p.Ln(4)
}

// euroBetrag formatiert einen Betrag deutsch mit Euro-Zeichen.
func euroBetrag(betrag float64) string {
	s := fmt.Sprintf("%.2f", betrag)
	return strings.Replace(s, ".", ",", 1) + " €"
}
