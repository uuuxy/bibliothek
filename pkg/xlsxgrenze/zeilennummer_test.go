package xlsxgrenze_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"bibliothek/pkg/xlsxgrenze"

	"github.com/xuri/excelize/v2"
)

// Steht hinter einer gültigen Zeile eine mit einer Nummer jenseits der Blattgrenze von
// 1.048.576, zählte excelize jede fehlende Zeile einzeln ab (CVE-2026-107212): Nach der
// Meldung hält eine Datei von 1,5 KB mit der Nummer 231999999999940 GetRows elf Tage auf
// einem Prozessorkern fest. Die Schranke in MitMappe fängt eine Schleife nicht ab, und alle
// drei Importwege lesen mit GetRows.
//
// Die eingesetzte Fassung weist die Zeile ab. Geprüft wird die erste Nummer über der Grenze:
// An einer Fassung ohne die Korrektur wird der Test rot, statt tagelang zu laufen.

// zweiZeilen baut ein Blatt mit einer Zeile 1 und einer zweiten mit der gegebenen Nummer.
func zweiZeilen(t *testing.T, nummer int) []byte {
	t.Helper()
	return mappeMitZeilen(t, fmt.Sprintf(`<row r="1"><c r="A1" t="inlineStr"><is><t>erste</t></is></c></row>`+
		`<row r="%d"><c r="A%d" t="inlineStr"><is><t>zweite</t></is></c></row>`, nummer, nummer))
}

func TestMitMappe_ZeilennummerJenseitsDerBlattgrenzeWirdAbgewiesen(t *testing.T) {
	for _, roh := range []bool{false, true} {
		lies := func(f *excelize.File) ([][]string, error) {
			return f.GetRows("Tabelle1", excelize.Options{RawCellValue: roh})
		}

		// Dieselbe Mappe mit einer Nummer im Blatt liest sich: Der Fehler unten liegt an der
		// Nummer, nicht an der Bauart der Probe.
		zeilen, err := xlsxgrenze.MitMappe(bytes.NewReader(zweiZeilen(t, 3)), lies)
		if err != nil || len(zeilen) != 3 || zeilen[2][0] != "zweite" {
			t.Fatalf("roh=%v: Zeile 3 gab %v, Fehler %v — erwartet drei Zeilen", roh, zeilen, err)
		}

		zeilen, err = xlsxgrenze.MitMappe(bytes.NewReader(zweiZeilen(t, excelize.TotalRows+1)), lies)
		if !errors.Is(err, excelize.ErrMaxRows) {
			t.Errorf("roh=%v: %d Zeilen, Fehler %v — erwartet ErrMaxRows. Diese excelize-Fassung "+
				"zählt bis zu einer Zeilennummer jenseits der Blattgrenze (CVE-2026-107212)",
				roh, len(zeilen), err)
		}
	}
}
