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

// Zeilenteiler bricht einen Text in der gerade gesetzten Schrift auf eine Breite um, wie
// MultiCell ihn druckt; gofpdf erfüllt ihn mit SplitLines.
type Zeilenteiler interface {
	SplitLines(text []byte, breite float64) [][]byte
}

// KuerzeAufZeilen kürzt den Text, bis er umbrochen auf höchstens so viele Zeilen der Breite
// passt. Gezählt wird, wie MultiCell umbricht, nicht nach Zeichen: Großbuchstaben füllen eine
// Zeile mit halb so vielen Zeichen wie Ziffern, und eine Zeile zu viel druckt MultiCell unter die
// Zelle.
func KuerzeAufZeilen(pdf Zeilenteiler, messbar func(string) string, text string, breite float64, zeilen int) string {
	passt := func(t string) bool { return len(pdf.SplitLines([]byte(messbar(t)), breite)) <= zeilen }
	if passt(text) {
		return text
	}
	runen := []rune(text)
	for len(runen) > 1 {
		runen = runen[:len(runen)-1]
		gekuerzt := strings.TrimRight(string(runen), " ") + "…"
		if passt(gekuerzt) {
			return gekuerzt
		}
	}
	return "…"
}

// KuerzeAufZeichen kürzt den Text auf höchstens so viele Zeichen, das Auslassungszeichen
// eingerechnet; hoechstens ist damit die Länge der Ausgabe, an der sich eine Spaltenbreite
// ablesen lässt. Gezählt wird in Zeichen und nicht in Bytes: Ein Umlaut belegt in UTF-8 zwei,
// und ein Schnitt mitten durch „ä" hinterlässt ein halbes Zeichen, aus dem der Übersetzer von
// gofpdf Zeichensalat macht.
func KuerzeAufZeichen(text string, hoechstens int) string {
	runen := []rune(text)
	if len(runen) <= hoechstens {
		return text
	}
	return string(runen[:hoechstens-1]) + "…"
}
