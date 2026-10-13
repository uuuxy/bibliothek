package uebernahme

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"bibliothek/pkg/isbnutil"
)

var keinISBNZeichen = regexp.MustCompile(`[^0-9Xx]`)

// normalisiereISBN entfernt Bindestriche und Leerzeichen und schreibt das Prüfzeichen groß.
func normalisiereISBN(roh string) string {
	return strings.ToUpper(keinISBNZeichen.ReplaceAllString(roh, ""))
}

// pruefeISBN liefert die normalisierte ISBN und ob ihre Prüfziffer stimmt. Eine leere Eingabe
// gilt als gültig, weil buecher_titel.isbn NULL sein darf.
func pruefeISBN(roh string) (normalisiert string, ok bool) {
	if roh == "" {
		return "", true
	}
	n := normalisiereISBN(roh)
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
	for i, zeichen := range isbn {
		if !unicode.IsDigit(zeichen) {
			return false
		}
		ziffer := int(zeichen - '0')
		if i%2 == 0 {
			summe += ziffer
		} else {
			summe += ziffer * 3
		}
	}
	return summe%10 == 0
}

func pruefeISBN10(isbn string) bool {
	summe := 0
	for i, zeichen := range isbn {
		var ziffer int
		switch {
		case i == 9 && (zeichen == 'X' || zeichen == 'x'):
			ziffer = 10
		case unicode.IsDigit(zeichen):
			ziffer = int(zeichen - '0')
		default:
			return false
		}
		summe += ziffer * (10 - i)
	}
	return summe%11 == 0
}

// VormerkSchluessel ist die Form, unter der eine ISBN vorgemerkt wird: die, in der die
// Datenbank sie speichert (isbn_normalform). Eine zehnstellige ISBN wird dort
// dreizehnstellig; unter der zehnstelligen vorgemerkt, träfe sie den Titel, der sie
// dreizehnstellig trägt, erst am UNIQUE-Index.
func VormerkSchluessel(roh string) string {
	return isbnutil.Normalform(normalisiereISBN(roh))
}

// KlaereISBN liefert die ISBN, mit der ein Titel geschrieben wird, und meldet jede Abwertung
// als Warnung. "" heißt: den Titel ohne ISBN übernehmen; das ist besser, als ihn wegen einer
// falschen Prüfziffer zu verlieren.
//
// gesehen bildet die ISBN unter ihrem VormerkSchluessel auf die ID der Quelle ab und wird
// fortgeschrieben. buecher_titel.isbn ist UNIQUE; ohne die Vormerkung liefe jede Dublette in
// einen Fehler 23505 und kostete den zweiten Titel.
func KlaereISBN(p *Protokoll, quellID, roh string, gesehen map[string]string) string {
	norm, ok := pruefeISBN(roh)
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
