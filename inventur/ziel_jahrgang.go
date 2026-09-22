package inventur

import "errors"

// Mehrjahresband (Antwort der Schule vom 22.09.2026, docs/OFFEN.md 9.6): Am Werk steht,
// bis zu welchem Jahrgang das Buch beim Kind bleibt. Die Frist ist dann der Stichtag des
// Schuljahres, in dem das Kind diesen Jahrgang beendet (internal/service/loan_rules.go).
//
// 0 heißt ein Schuljahr — die Vorgabe für jedes Buch. Ein Wert gibt es nur an einem
// Lernmittel: Bei einem Buch der Schülerbücherei liest die Fristregel ihn nie, und ein
// Wert, den keine Regel liest, wäre eine Behauptung ohne Wirkung.
const (
	zielJahrgangMin = 5
	zielJahrgangMax = 13
)

var errZielJahrgang = errors.New("zielJahrgang: nur bei einem Lernmittel, Jahrgang 5 bis 13; 0 heißt ein Schuljahr")

// pruefeZielJahrgang prüft die Angabe für Anlegen und Ändern — dieselbe Regel an beiden
// Türen.
func pruefeZielJahrgang(istLernmittel bool, ziel int) error {
	if ziel == 0 {
		return nil
	}
	if !istLernmittel || ziel < zielJahrgangMin || ziel > zielJahrgangMax {
		return errZielJahrgang
	}
	return nil
}
