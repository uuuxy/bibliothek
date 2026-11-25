package isbnutil

import "strings"

// Formen liefert die Schreibweisen, unter denen dieselbe ISBN im Bestand stehen kann: ohne
// Trenner, das Prüfzeichen X groß, zehn- und dreizehnstellig (978). Leer heißt: keine ISBN.
// Das Prüfzeichen der Eingabe prüft Formen nicht, sonst bliebe ein krumm eingetippter Bestand
// unauffindbar. Zwilling von isbnFormen in frontend/src/lib/utils/isbnFormen.js; beide prüfen
// die Fälle in isbnFormen.faelle.json.
func Formen(roh string) []string {
	var b strings.Builder
	for _, r := range roh {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == 'x' || r == 'X':
			b.WriteByte('X')
		}
	}
	n := b.String()
	switch {
	case len(n) == 10:
		zwoelf := "978" + n[:9]
		return []string{n, zwoelf + pruefziffer13(zwoelf)}
	case len(n) == 13 && strings.HasPrefix(n, "978"):
		// Nur 978 lässt sich zurückrechnen: Der Bereich 979 hat keine zehnstellige Form.
		kern := n[3:12]
		return []string{n, kern + pruefzeichen10(kern)}
	case len(n) == 13 && strings.HasPrefix(n, "979"):
		return []string{n}
	}
	return nil
}
