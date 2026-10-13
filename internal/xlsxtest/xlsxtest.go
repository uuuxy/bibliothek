// Package xlsxtest baut Excel-Arbeitsmappen für Tests. Die Tests des LUSD-Lesers
// (internal/lusd) und die Tests der Türen in api/ brauchen dieselbe Mappe; nur Testdateien
// binden das Paket ein.
package xlsxtest

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Baue erzeugt eine Arbeitsmappe: je Blatt die Zellen je Zeile. Eine Zelle vom Typ time.Time
// wird als echte Datumszelle geschrieben (Excel-Serienzahl).
func Baue(t *testing.T, blaetter map[string][][]any) []byte {
	t.Helper()
	mappe := excelize.NewFile()
	erstes := true
	for name, zeilen := range blaetter {
		legeBlattAn(t, mappe, name, erstes)
		erstes = false
		fuelleBlatt(t, mappe, name, zeilen)
	}
	var puffer bytes.Buffer
	if err := mappe.Write(&puffer); err != nil {
		t.Fatal(err)
	}
	return puffer.Bytes()
}

// legeBlattAn gibt dem ersten Blatt den Namen des Blatts, das jede neue Mappe mitbringt, und
// legt jedes weitere an.
func legeBlattAn(t *testing.T, mappe *excelize.File, name string, erstes bool) {
	t.Helper()
	if erstes {
		if err := mappe.SetSheetName("Sheet1", name); err != nil {
			t.Fatal(err)
		}
		return
	}
	if _, err := mappe.NewSheet(name); err != nil {
		t.Fatal(err)
	}
}

func fuelleBlatt(t *testing.T, mappe *excelize.File, name string, zeilen [][]any) {
	t.Helper()
	for z, zeile := range zeilen {
		for s, wert := range zeile {
			zelle, err := excelize.CoordinatesToCellName(s+1, z+1)
			if err != nil {
				t.Fatal(err)
			}
			if err := mappe.SetCellValue(name, zelle, wert); err != nil {
				t.Fatal(err)
			}
		}
	}
}
