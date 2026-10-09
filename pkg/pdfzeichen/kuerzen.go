package pdfzeichen

import "strings"

// Breitenmesser nennt die Breite eines Texts in der gerade gesetzten Schrift. Ein
// gofpdf-Dokument erfüllt ihn, in beiden Fassungen, die im Haus im Einsatz sind.
type Breitenmesser interface {
	GetStringWidth(text string) float64
}

// KuerzeAufBreite kürzt den Text, bis er in die Breite (Millimeter) passt, gemessen in der
// gerade gesetzten Schrift und nicht nach Zeichenzahl: „Öztürk, Ali" und „MMMMMMMMMMM" sind
// beide elf Zeichen und über 50 % verschieden breit.
//
// messbar bringt den Text in die Form, die auch gedruckt wird (Uebersetzer): Gemessen wird,
// was auf dem Papier steht.
func KuerzeAufBreite(pdf Breitenmesser, messbar func(string) string, text string, breite float64) string {
	if pdf.GetStringWidth(messbar(text)) <= breite {
		return text
	}
	runen := []rune(text)
	for len(runen) > 1 {
		runen = runen[:len(runen)-1]
		gekuerzt := strings.TrimRight(string(runen), " ") + "…"
		if pdf.GetStringWidth(messbar(gekuerzt)) <= breite {
			return gekuerzt
		}
	}
	return "…"
}

// Zellenmesser nennt dazu den Rand, den das Dokument links und rechts vom Text einer Zelle lässt.
type Zellenmesser interface {
	Breitenmesser
	GetCellMargin() float64
}

// KuerzeAufZelle kürzt den Text, bis er in der gerade gesetzten Schrift in eine Zelle der Breite
// passt. gofpdf druckt Überlanges über die Nachbarzelle.
func KuerzeAufZelle(pdf Zellenmesser, messbar func(string) string, text string, breite float64) string {
	return KuerzeAufBreite(pdf, messbar, text, breite-2*pdf.GetCellMargin())
}

// KuerzeAufZeichen kürzt s auf höchstens max Zeichen, das Auslassungszeichen eingerechnet;
// max ist damit die Länge der Ausgabe, an der sich eine Spaltenbreite ablesen lässt. Gezählt
// wird in Zeichen und nicht in Bytes: Ein Umlaut belegt in UTF-8 zwei, und ein Schnitt mitten
// durch „ä" hinterlässt ein halbes Zeichen, aus dem der Übersetzer von gofpdf Zeichensalat
// macht.
func KuerzeAufZeichen(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
