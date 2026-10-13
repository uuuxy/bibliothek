// Package code39 rechnet das Prüfzeichen von Code 39 und trennt es von einem Scan ab.
//
// Ältere Ausweise und Buchetiketten tragen den Strichcode in Code 39 mit Prüfzeichen, und das
// Zeichen steht in den Daten: Auf dem Ausweis steht „A-10003", im Strichcode „A-100037". Ein
// Leser gibt es als Teil der Nummer zurück, solange er es nicht selbst prüft und entfernt, und
// die meisten Geräte sind so eingestellt. Gedruckt wird Code 128 ohne Prüfzeichen
// (pkg/strichcode); die älteren Karten und Etiketten sind im Umlauf und gelten weiter, ohne
// dass die Schule sie neu druckt.
//
// Die Rechnung ist Mod 43 über den Zeichensatz von Code 39, Zeichen für Zeichen wie in
// github.com/boombuler/barcode/code39 (getChecksum): „B-10001" ergibt '6', „A-10003" ergibt
// '7'.
package code39

import "strings"

// zeichensatz ist die Wertetabelle von Code 39: Die Stelle eines Zeichens ist sein Wert. '*'
// ist nur Start und Stopp und steht deshalb nicht darin.
const zeichensatz = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ-. $/+%"

// Pruefzeichen liefert das Prüfzeichen (Mod 43) für einen Inhalt. Das zweite Ergebnis ist
// false, wenn der Inhalt ein Zeichen enthält, das Code 39 nicht kennt; einen solchen Aufdruck
// gab es dann nie.
func Pruefzeichen(inhalt string) (byte, bool) {
	summe := 0
	for i := 0; i < len(inhalt); i++ {
		wert := strings.IndexByte(zeichensatz, inhalt[i])
		if wert < 0 {
			return 0, false
		}
		summe += wert
	}
	return zeichensatz[summe%43], true
}

// OhnePruefzeichen trennt ein angehängtes Prüfzeichen ab. Das zweite Ergebnis ist nur wahr,
// wenn das letzte Zeichen das Prüfzeichen des Rests ist; jeder andere Scan kommt unverändert
// zurück.
//
// Ein Beweis, dass ein Prüfzeichen gemeint war, ist das nicht: Bei 43 möglichen Zeichen sieht
// im Schnitt jeder 43. gültige Code so aus, als hinge eines daran. Der Rufer nimmt die
// Funktion deshalb nur als zweiten Versuch, nachdem der Scan, wie er kam, nichts gefunden hat.
// Ein falscher Treffer ist dann nur möglich, wenn es den gekürzten Wert gibt und den vollen
// nicht, und das ist der Fall, den sie auflösen soll.
//
// Die Mindestlänge von drei Zeichen hält Unsinn heraus: Kein Barcode des Programms ist kürzer,
// und ein Scan aus zwei Zeichen wäre nach dem Kürzen ein einzelnes.
func OhnePruefzeichen(scan string) (string, bool) {
	if len(scan) < 3 {
		return scan, false
	}
	kern := scan[:len(scan)-1]
	erwartet, ok := Pruefzeichen(kern)
	if !ok || erwartet != scan[len(scan)-1] {
		return scan, false
	}
	return kern, true
}
