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
// wird als echte Datumszelle geschrieben (Excel-Serienzahl). Die Reihenfolge der Blätter ist die
// der Map, also zufällig; wer sie braucht, nimmt BaueInReihenfolge.
func Baue(t *testing.T, blaetter map[string][][]any) []byte {
	t.Helper()
	var liste []Blatt
	for name, zeilen := range blaetter {
		liste = append(liste, Blatt{Name: name, Zeilen: zeilen})
	}
	return BaueInReihenfolge(t, liste...)
}

// Blatt ist ein Blatt der Arbeitsmappe: sein Name und die Zellen je Zeile.
type Blatt struct {
	Name   string
	Zeilen [][]any
}

// BaueInReihenfolge erzeugt eine Arbeitsmappe mit den Blättern in der genannten Reihenfolge. Ein
// Test, dessen Aussage am ersten oder letzten Blatt hängt, wäre mit der Map von Baue nur
// manchmal rot.
func BaueInReihenfolge(t *testing.T, blaetter ...Blatt) []byte {
	t.Helper()
	mappe := excelize.NewFile()
	for i, blatt := range blaetter {
		legeBlattAn(t, mappe, blatt.Name, i == 0)
		fuelleBlatt(t, mappe, blatt.Name, blatt.Zeilen)
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
