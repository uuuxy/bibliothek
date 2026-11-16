package isbnutil

import "regexp"

// eingabeForm ist die Form, die eine ISBN als Eingabe haben muss: Ziffern, die letzte darf ein X
// sein. Sie ist weiter als isbnForm, weil sie nur prüft, ob eine Nummer wie eine ISBN aussieht.
var eingabeForm = regexp.MustCompile(`^[0-9]{9,13}[0-9xX]?$`)

// HatForm sagt, ob die Eingabe die Form einer ISBN hat: ohne Leerraum und Bindestriche zehn bis
// dreizehn Zeichen, Ziffern und am Ende auch ein X. Das Prüfzeichen prüft sie nicht. Eine Anfrage
// an die Katalogdienste geht nur mit einer solchen Nummer hinaus, und eine Maske lehnt eine ISBN
// ohne diese Form ab.
func HatForm(isbn string) bool {
	sauber := ohneLeerraumUndBindestrich(isbn)
	if len(sauber) < 10 || len(sauber) > 13 {
		return false
	}
	return eingabeForm.MatchString(sauber)
}

// ohneLeerraumUndBindestrich entfernt Leerzeichen, Bindestriche, Tabulatoren und Zeilenumbrüche
// in einem Durchgang. Anders als OhneTrenner nimmt sie auch den Leerraum einer eingefügten Zeile.
func ohneLeerraumUndBindestrich(s string) string {
	mussBereinigen := false
	behalten := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			mussBereinigen = true
		} else {
			behalten++
		}
	}
	if !mussBereinigen {
		return s
	}
	puffer := make([]byte, behalten)
	stelle := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '-' && c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			puffer[stelle] = c
			stelle++
		}
	}
	return string(puffer)
}
