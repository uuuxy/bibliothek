// Package pdfzeichen bringt Text aufs Papier: in den Schriften der Ausdrucke (NewDokument), in
// der Form, die diese drucken (Ersetze), und gekürzt auf die Breite einer Zelle.
//
// Die Schriften sind ins Dokument eingebettet und kennen jeden lateinischen Buchstaben; „Ayşe"
// steht auf dem Etikett wie am Bildschirm. Ein Zeichen jenseits der Basisebene von Unicode, etwa
// ein Emoji im Buchtitel, kann gofpdf aber nicht setzen und verweigert dann das ganze Dokument.
//
// Jedes Dokument mit gofpdf entsteht über NewDokument, jeder Text an gofpdf und maroto geht durch
// Ersetze; pruefung/pdfzeichen_ratsche_test.go hält beides.
package pdfzeichen

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Ersetze bringt Text in die Form, die die Schriften drucken. Ein Buchstabe mit getrennt
// gespeichertem Akzent wird zu einem Zeichen (NFC): So misst gofpdf seine Breite richtig, und der
// Akzent sitzt, wo die Schrift ihn zeichnet. Ein Zeichen jenseits der Basisebene wird zu „�“.
func Ersetze(text string) string {
	return strings.Map(func(r rune) rune {
		if r > 0xFFFF {
			return utf8.RuneError
		}
		return r
	}, norm.NFC.String(text))
}
