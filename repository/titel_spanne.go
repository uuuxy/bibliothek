package repository

import "errors"

// ErrMehrjahresband lehnt ein Mehrjahresband ab, das die Regel verletzt. Ein Mehrjahresband ist
// ein Lernmittel, das über die Spanne „im Unterricht von Jahrgang … bis" beim Kind bleibt. Die Jahreszahl, bis zu der es bleibt, ist jahrgang_bis; eine zweite gibt es
// nicht. Die Frist rechnet bis zum Stichtag des Schuljahres, in dem das Kind diesen Jahrgang
// beendet (internal/service/frist.go).
//
// Der Schalter gilt nur an einem Lernmittel (bei einem Buch der Schülerbücherei liest ihn keine
// Regel) und nur mit einer Spanne über mehr als einen Jahrgang: Ein „Mehrjahresband 7 bis 7"
// hätte keine Wirkung, die Maske sagte aber „bleibt beim Kind". Dieselbe Regel steht als CHECK
// chk_mehrjahresband_spanne in der Datenbank; hier wird sie zu einer lesbaren Antwort (400),
// bevor die Datenbank sie zu einem 500 macht.
var ErrMehrjahresband = errors.New("mehrjahresband: nur bei einem Lernmittel und nur mit einer Spanne über mehr als einen Jahrgang (bis über von, beide 1 bis 13)")

// ErrJahrgangsSpanne lehnt eine halbe Spanne ab: „im Unterricht von Jahrgang … bis" ist ganz
// angegeben oder gar nicht. 0 und 0 heißt „unbekannt" und ist eine eigene Angabe; eine Vorgabe
// gibt es nicht. Dieselbe Regel steht als CHECK chk_jahrgang_spanne in der Datenbank.
var ErrJahrgangsSpanne = errors.New("jahrgang: „von“ und „bis“ gehören zusammen (beide 1 bis 13, „von“ nicht über „bis“) oder bleiben beide leer")

// PruefeJahrgangsSpanne prüft die Spanne für Anlegen und Ändern.
func PruefeJahrgangsSpanne(von, bis int) error {
	if von == 0 && bis == 0 {
		return nil
	}
	if von < 1 || bis > 13 || bis < von {
		return ErrJahrgangsSpanne
	}
	return nil
}

// PruefeMehrjahresband prüft die Angabe für Anlegen und Ändern — dieselbe Regel an beiden
// Türen.
func PruefeMehrjahresband(istLernmittel, mehrjahresband bool, von, bis int) error {
	if !mehrjahresband {
		return nil
	}
	if !istLernmittel || von < 1 || bis > 13 || bis <= von {
		return ErrMehrjahresband
	}
	return nil
}
