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
	f := excelize.NewFile()
	erstes := true
	for name, rows := range blaetter {
		legeBlattAn(t, f, name, erstes)
		erstes = false
		fuelleBlatt(t, f, name, rows)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// legeBlattAn gibt dem ersten Blatt den Namen des Blatts, das jede neue Mappe mitbringt, und
// legt jedes weitere an.
func legeBlattAn(t *testing.T, f *excelize.File, name string, erstes bool) {
	t.Helper()
	if erstes {
		if err := f.SetSheetName("Sheet1", name); err != nil {
			t.Fatal(err)
		}
		return
	}
	if _, err := f.NewSheet(name); err != nil {
		t.Fatal(err)
	}
}

func fuelleBlatt(t *testing.T, f *excelize.File, name string, rows [][]any) {
	t.Helper()
	for r, row := range rows {
		for c, wert := range row {
			zelle, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.SetCellValue(name, zelle, wert); err != nil {
				t.Fatal(err)
			}
		}
	}
}
