package uebernahme

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"bibliothek/pkg/isbnutil"
)

var reNichtISBN = regexp.MustCompile(`[^0-9Xx]`)

// NormalisiereISBN entfernt Bindestriche und Leerzeichen und schreibt das Prüfzeichen groß.
func NormalisiereISBN(roh string) string {
	return strings.ToUpper(reNichtISBN.ReplaceAllString(roh, ""))
}

// PruefeISBN liefert die normalisierte ISBN und ob ihre Prüfziffer stimmt.
// Eine leere Eingabe gilt als gültig — buecher_titel.isbn ist nullbar.
func PruefeISBN(roh string) (normalisiert string, ok bool) {
	if roh == "" {
		return "", true
	}
	n := NormalisiereISBN(roh)
	switch len(n) {
	case 10:
		return n, pruefeISBN10(n)
	case 13:
		return n, pruefeISBN13(n)
	default:
		return n, false
	}
}

func pruefeISBN13(isbn string) bool {
	summe := 0
	for i, ch := range isbn {
		if !unicode.IsDigit(ch) {
			return false
		}
		d := int(ch - '0')
		if i%2 == 0 {
			summe += d
		} else {
			summe += d * 3
		}
	}
	return summe%10 == 0
}

func pruefeISBN10(isbn string) bool {
	summe := 0
	for i, ch := range isbn {
		var d int
		switch {
		case i == 9 && (ch == 'X' || ch == 'x'):
			d = 10
		case unicode.IsDigit(ch):
			d = int(ch - '0')
		default:
			return false
		}
		summe += d * (10 - i)
	}
	return summe%11 == 0
}

// VormerkSchluessel ist die Form, unter der eine ISBN vorgemerkt wird: die, in der die
// Datenbank sie speichert (isbn_normalform). Eine zehnstellige ISBN wird dort
// dreizehnstellig; unter der zehnstelligen vorgemerkt, träfe sie den Titel, der sie
// dreizehnstellig trägt, erst am UNIQUE-Index.
func VormerkSchluessel(roh string) string {
	return isbnutil.Normalform(NormalisiereISBN(roh))
}

// KlaereISBN ermittelt die ISBN, mit der ein Titel geschrieben wird, und meldet jede
// Abwertung als Warnung. Rückgabe "" heißt: Titel ohne ISBN übernehmen — das ist besser,
// als ihn wegen einer kaputten Prüfziffer ganz zu verlieren.
//
// gesehen bildet ISBN → Quell-ID ab und wird fortgeschrieben, unter dem VormerkSchluessel.
// buecher_titel.isbn ist UNIQUE; ohne diese Vormerkung liefe jede Dublette in einen 23505
// und kostete den zweiten Titel. Im Littera-Altbestand betrifft das 1.100 Titel.
func KlaereISBN(p *Protokoll, quellID, roh string, gesehen map[string]string) string {
	norm, ok := PruefeISBN(roh)
	if roh != "" && !ok {
		p.Warnung(quellID, roh, "ungültige ISBN-Prüfziffer – Titel wird ohne ISBN übernommen")
		return ""
	}
	if norm == "" {
		return ""
	}
	norm = VormerkSchluessel(norm)
	if vorher, belegt := gesehen[norm]; belegt {
		p.Warnung(quellID, norm, fmt.Sprintf(
			"doppelte ISBN – bereits übernommen als %s=%s; Titel wird ohne ISBN übernommen", p.idFeld, vorher))
		return ""
	}
	gesehen[norm] = quellID
	return norm
}
