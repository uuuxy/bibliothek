package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"bibliothek/repository"
)

// TestErkenneScan: Die globale Suche fragt „was ist das?" und springt hin. Sie löst einen Wert
// in der Reihenfolge der Theke auf: Exemplar, Littera-Etikett (EAN-13 → Mediennummer), dann
// Ausweis. Ein Fehler der Datenbank kommt als Fehler an; verschluckt sähe er aus wie eine
// unbekannte Nummer, und die Volltextsuche liefe los.
func TestErkenneScan(t *testing.T) {
	dbFehler := errors.New("verbindung weg")
	exemplar := &repository.BookCopy{ID: "ex-1", TitelID: "titel-1", BarcodeID: "B-12345"}
	altesEtikett := &repository.BookCopy{ID: "ex-2", TitelID: "titel-2", BarcodeID: "58968"}
	leser := &repository.Student{ID: "leser-1", BarcodeID: "S-7"}

	// nurUnter liefert das Exemplar genau für diesen Barcode.
	nurUnter := func(barcode string, ex *repository.BookCopy) func(context.Context, string) (*repository.BookCopy, error) {
		return func(_ context.Context, gefragt string) (*repository.BookCopy, error) {
			if gefragt == barcode {
				return ex, nil
			}
			return nil, nil
		}
	}

	faelle := []struct {
		name       string
		wert       string
		exemplare  func(context.Context, string) (*repository.BookCopy, error)
		leser      *repository.Student
		leserErr   error
		soll       *ScanTreffer
		sollFehler error
	}{
		{
			name: "Barcode eines Exemplars", wert: "B-12345", exemplare: nurUnter("B-12345", exemplar),
			soll: &ScanTreffer{Typ: "exemplar", ID: "ex-1", TitelID: "titel-1", Barcode: "B-12345"},
		},
		{
			name: "Littera-Etikett: die EAN-13 führt zur Mediennummer", wert: "5896800039556",
			exemplare: nurUnter("58968", altesEtikett),
			soll:      &ScanTreffer{Typ: "exemplar", ID: "ex-2", TitelID: "titel-2", Barcode: "58968"},
		},
		{
			name: "das Exemplar geht vor dem Ausweis", wert: "B-12345",
			exemplare: nurUnter("B-12345", exemplar), leser: leser,
			soll: &ScanTreffer{Typ: "exemplar", ID: "ex-1", TitelID: "titel-1", Barcode: "B-12345"},
		},
		{
			name: "kein Exemplar, aber ein Ausweis", wert: "S-7", leser: leser,
			soll: &ScanTreffer{Typ: "schueler", ID: "leser-1", Barcode: "S-7"},
		},
		{name: "weder Exemplar noch Ausweis", wert: "unbekannt"},
		{
			name: "Fehler beim Exemplar", wert: "B-12345",
			exemplare:  func(context.Context, string) (*repository.BookCopy, error) { return nil, dbFehler },
			sollFehler: dbFehler,
		},
		{name: "Fehler beim Ausweis", wert: "S-7", leserErr: dbFehler, sollFehler: dbFehler},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			treffer, err := ErkenneScan(context.Background(),
				&mockBookRepo{mockGetCopyByBarcode: f.exemplare},
				&stubLeserRepo{leser: f.leser, err: f.leserErr}, f.wert)
			if !errors.Is(err, f.sollFehler) {
				t.Fatalf("Fehler %v, erwartet %v", err, f.sollFehler)
			}
			if !reflect.DeepEqual(treffer, f.soll) {
				t.Errorf("Treffer %+v, erwartet %+v", treffer, f.soll)
			}
		})
	}
}
