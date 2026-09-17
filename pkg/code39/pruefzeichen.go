// Package code39 kennt das Prüfzeichen, das bis zum 17.09.2026 auf jedem Ausweis und
// jedem Buchetikett dieser Anwendung stand — und rechnet es wieder heraus.
//
// Die Geschichte: `api/barcode_generate.go` druckte mit
// `code39.Encode(strings.ToUpper(inhalt), true, true)`. Das mittlere `true` heißt
// „Prüfzeichen anhängen", und dieses Zeichen steht IN den Strichcode-Daten. Ein Leser
// gibt es als Teil der Nummer zurück, solange er es nicht selbst prüft und entfernt —
// und die Voreinstellung fast aller Geräte ist „nicht prüfen".
//
// Auf dem Ausweis stand also „A-10003" und im Strichcode darüber „A-100037", auf dem
// Buchetikett „B-10001" gegen „B-100016". Gedruckt wird seit dem 17.09.2026 Code 128 ohne
// Prüfzeichen, aber die Karten und Etiketten von VORHER sind im Umlauf und sollen weiter
// funktionieren — eine Anforderung, die von Anfang an stand. Ohne diese Umkehrung müsste
// die Schule jeden Ausweis und jedes Etikett neu drucken.
//
// Die Rechnung ist Mod 43 über den Zeichensatz von Code 39, Zeichen für Zeichen dieselbe
// wie in github.com/boombuler/barcode/code39 (getChecksum). Nachgerechnet an den beiden
// gemessenen Werten: „B-10001" ergibt '6', „A-10003" ergibt '7'.
package code39

import "strings"

// zeichensatz ist die Wertetabelle von Code 39: Index = Wert des Zeichens.
// Reihenfolge und Inhalt entsprechen der Norm; '*' ist nur Start/Stopp und steht deshalb
// nicht darin.
const zeichensatz = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ-. $/+%"

// Pruefzeichen liefert das Mod-43-Prüfzeichen für einen Inhalt.
// Das zweite Ergebnis ist falsch, wenn der Inhalt ein Zeichen enthält, das Code 39 nicht
// kennt — dann gab es auch nie einen solchen Aufdruck.
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

// OhnePruefzeichen trennt ein angehängtes Mod-43-Prüfzeichen ab.
//
// Das zweite Ergebnis ist nur dann wahr, wenn das LETZTE Zeichen genau das Prüfzeichen
// des Restes ist. Jeder andere Scan kommt unverändert zurück.
//
// WICHTIG für die Verwendung: Das ist kein Beweis, dass ein Prüfzeichen gemeint war.
// Bei 43 möglichen Zeichen sieht im Schnitt jeder 43. gültige Code zufällig so aus, als
// hinge eines dran — „B-10001" selbst könnte theoretisch der gekürzte Wert von etwas
// anderem sein. Deshalb darf diese Funktion NUR als zweiter Versuch benutzt werden,
// nachdem der Scan so, wie er kam, nichts gefunden hat. Dann ist ein falscher Treffer
// nur möglich, wenn der gekürzte Wert existiert und der volle nicht — und genau das ist
// der Fall, den sie auflösen soll.
//
// Die Mindestlänge von drei Zeichen hält Unsinn heraus: Ein Barcode dieser Anwendung ist
// nie kürzer, und ein Scan aus zwei Zeichen wäre nach dem Kürzen ein einzelnes.
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
