package inventur

import "errors"

// Mehrjahresband (Migration 134, Antwort der Schule vom 22.09.2026, docs/OFFEN.md 9.6):
// Ein Lernmittel, das über die Spanne „im Unterricht von Jahrgang … bis" beim Kind bleibt.
// Die Jahreszahl, bis zu der es bleibt, ist jahrgang_bis — eine zweite gibt es nicht. Die
// Frist rechnet bis zum Stichtag des Schuljahres, in dem das Kind diesen Jahrgang beendet
// (internal/service/loan_rules.go).
//
// Der Schalter gilt nur an einem Lernmittel (bei einem Buch der Schülerbücherei liest ihn
// keine Regel) und nur mit einer Spanne über mehr als einen Jahrgang — ein „Mehrjahresband
// 7 bis 7" hätte keine Wirkung, die Maske sagte aber „bleibt beim Kind". Dieselbe Regel
// steht als CHECK chk_mehrjahresband_spanne in der Datenbank; hier wird sie zu einer
// lesbaren Antwort (400), bevor die Datenbank sie zu einem 500 macht.
var errMehrjahresband = errors.New("mehrjahresband: nur bei einem Lernmittel und nur mit einer Spanne über mehr als einen Jahrgang (bis über von, beide 1 bis 13)")

// pruefeMehrjahresband prüft die Angabe für Anlegen und Ändern — dieselbe Regel an beiden
// Türen.
func pruefeMehrjahresband(istLernmittel, mehrjahresband bool, von, bis int) error {
	if !mehrjahresband {
		return nil
	}
	if !istLernmittel || von < 1 || bis > 13 || bis <= von {
		return errMehrjahresband
	}
	return nil
}
