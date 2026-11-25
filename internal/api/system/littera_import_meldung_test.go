package system

import (
	"fmt"
	"strings"
	"testing"

	"bibliothek/internal/pkg/csvutil"
	"bibliothek/internal/repository"
)

// Die Meldung nennt bis zu zehn Zeilen mit ihrer Nummer und zählt den Rest: Bei einer falsch
// zugeordneten Spalte trägt jede Zeile einen Ausweis, und die Meldung bliebe lesbar.
func TestMeldungImportNummerIstAusweis_NenntZeilenUndZaehltDenRest(t *testing.T) {
	var exemplare []repository.ImportExemplar
	for i := range 12 {
		exemplare = append(exemplare, repository.ImportExemplar{Barcode: fmt.Sprintf("A%d", i), Zeile: i + 2})
	}
	got := meldungImportNummerIstAusweis(exemplare)
	const will = "Nichts importiert: Die Zeilen 2 mit A0, 3 mit A1, 4 mit A2, 5 mit A3, 6 mit A4, " +
		"7 mit A5, 8 mit A6, 9 mit A7, 10 mit A8, 11 mit A9 und 2 weitere tragen als Buchnummer die Nummer eines " +
		"Leserausweises. Eine Nummer ist entweder ein Buch oder ein Ausweis. Bitte die Buchnummer " +
		"in der Datei berichtigen und die Datei erneut hochladen."
	if got != will {
		t.Errorf("Meldung\n%q\nerwartet\n%q", got, will)
	}
}

// Littera- und Bestandsimport lesen eine CSV nur bis zur festen Grenze, gezählt beim Lesen: 100 MB
// aus kurzen Zeilen stünden sonst als Millionen Zeilen im Speicher. Die Meldung nennt die Grenze.
func TestLeseCSVZeilen_BrichtUeberDerGrenzeAb(t *testing.T) {
	_, err := leseCSVZeilen([]byte(strings.Repeat("Barcode;Titel\n", csvutil.HoechstensZeilen+2)))
	if meldung := csvLesefehlerMeldung(err); !strings.Contains(meldung, fmt.Sprintf("höchstens %d", csvutil.HoechstensZeilen)) {
		t.Errorf("Meldung %q (%v), erwartet die Grenze", meldung, err)
	}
}
