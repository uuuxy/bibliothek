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
