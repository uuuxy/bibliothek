package isbnutil

import "strconv"

// AndereForm ist dieselbe ISBN in der anderen Länge: zu einer ISBN-10 die ISBN-13 mit 978, zu
// einer ISBN-13 mit 978 die ISBN-10. Leer heißt: Es gibt keine — die Eingabe ist keine ISBN,
// oder sie beginnt mit 979, und dazu gibt es keine zehnstellige Form.
//
// Der Go-Zwilling von isbnFormen (frontend/src/lib/utils/isbnFormen.js); beide lesen dieselben
// Prüffälle (isbnFormen.faelle.json, andere_form_test.go). Die Normalform (Migration 133) trennt beide Längen bewusst — die Littera-Übernahme
// behält eine gültige ISBN-10 —, und ein Strichcode auf dem Buchrücken ist immer eine EAN-13.
// Zwischen beiden liegt eine feste Rechnung: gleicher Kern, neu berechnetes Prüfzeichen.
//
// Die Prüfziffer der Eingabe wird nicht kontrolliert, wie im Browser: Ein falsch gelesener
// Code trifft dann nichts. Wer über die andere Form einen Titel findet, schlägt ihn vor und
// übernimmt ihn nicht still (docs/OFFEN.md 4.18, 5.5): Am Testserver steht unter 3499500252
// ein anderes Buch als unter 9783499500251 — die ISBN-10 hat ein falsches Prüfzeichen, und
// die Rechnung führt von ihr trotzdem auf die ISBN-13.
func AndereForm(roh string) string {
	n := Normalform(roh)
	if !isbnForm.MatchString(n) {
		return ""
	}
	switch {
	case len(n) == 10:
		zwoelf := "978" + n[:9]
		return zwoelf + pruefziffer13(zwoelf)
	case n[:3] == "978":
		kern := n[3:12]
		return kern + pruefzeichen10(kern)
	default:
		return ""
	}
}

// pruefzeichen10 ist das Prüfzeichen einer ISBN-10 zu ihren ersten neun Ziffern.
func pruefzeichen10(neun string) string {
	summe := 0
	for i := 0; i < 9; i++ {
		summe += int(neun[i]-'0') * (10 - i)
	}
	rest := (11 - summe%11) % 11
	if rest == 10 {
		return "X"
	}
	return strconv.Itoa(rest)
}

// pruefziffer13 ist die Prüfziffer einer ISBN-13 zu ihren ersten zwölf Ziffern.
func pruefziffer13(zwoelf string) string {
	summe := 0
	for i := 0; i < 12; i++ {
		gewicht := 1
		if i%2 == 1 {
			gewicht = 3
		}
		summe += int(zwoelf[i]-'0') * gewicht
	}
	return strconv.Itoa((10 - summe%10) % 10)
}
