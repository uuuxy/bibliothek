package system

import (
	"fmt"
	"testing"

	"bibliothek/repository"
)

// Die Meldung nennt bis zu zehn Zeilen mit ihrer Nummer und zählt den Rest: Bei einer falsch
// zugeordneten Spalte trägt jede Zeile einen Ausweis, und die Meldung bliebe lesbar.
func TestMeldungImportNummerIstAusweis_NenntZeilenUndZaehltDenRest(t *testing.T) {
	var exemplare []repository.ImportExemplar
	for i := range 12 {
		exemplare = append(exemplare, repository.ImportExemplar{Barcode: fmt.Sprintf("A%d", i), Zeile: i + 2})
	}
	got := meldungImportNummerIstAusweis(exemplare)
	const will = "Nichts importiert: Die Zeilen 2 (A0), 3 (A1), 4 (A2), 5 (A3), 6 (A4), 7 (A5), " +
		"8 (A6), 9 (A7), 10 (A8), 11 (A9) und 2 weitere tragen als Buchnummer die Nummer eines " +
		"Leserausweises. Eine Nummer ist entweder ein Buch oder ein Ausweis. Bitte die Buchnummer " +
		"in der Datei berichtigen und die Datei erneut hochladen."
	if got != will {
		t.Errorf("Meldung\n%q\nerwartet\n%q", got, will)
	}
}
