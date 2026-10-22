package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
)

// TestErkenneExemplar: Die Suche im Medienkatalog fragt „welches Exemplar ist das?" in der
// Reihenfolge der Theke: erst der Barcode, dann das Etikett von Littera (EAN-13 →
// Mediennummer). Ein Fehler der Datenbank kommt als Fehler an; verschluckt sähe er aus wie eine
// unbekannte Nummer.
func TestErkenneExemplar(t *testing.T) {
	dbFehler := errors.New("verbindung weg")
	exemplar := &repository.Exemplar{ID: "ex-1", TitelID: "titel-1", BarcodeID: "B-12345"}
	altesEtikett := &repository.Exemplar{ID: "ex-2", TitelID: "titel-2", BarcodeID: "58968"}
	ausgesondert := &repository.Exemplar{ID: "ex-3", TitelID: "titel-3", BarcodeID: "B-99", IstAusgesondert: true}

	// nurUnter liefert das Exemplar genau für diesen Barcode.
	nurUnter := func(barcode string, ex *repository.Exemplar) func(context.Context, string) (*repository.Exemplar, error) {
		return func(_ context.Context, gefragt string) (*repository.Exemplar, error) {
			if gefragt == barcode {
				return ex, nil
			}
			return nil, nil
		}
	}

	faelle := []struct {
		name       string
		wert       string
		exemplare  func(context.Context, string) (*repository.Exemplar, error)
		soll       *ExemplarTreffer
		sollFehler error
	}{
		{
			name: "Barcode eines Exemplars", wert: "B-12345", exemplare: nurUnter("B-12345", exemplar),
			soll: &ExemplarTreffer{ID: "ex-1", TitelID: "titel-1", Barcode: "B-12345"},
		},
		{
			name: "Littera-Etikett: die EAN-13 führt zur Mediennummer", wert: "5896800039556",
			exemplare: nurUnter("58968", altesEtikett),
			soll:      &ExemplarTreffer{ID: "ex-2", TitelID: "titel-2", Barcode: "58968"},
		},
		{
			name: "ausgesondertes Exemplar", wert: "B-99", exemplare: nurUnter("B-99", ausgesondert),
			soll: &ExemplarTreffer{ID: "ex-3", TitelID: "titel-3", Barcode: "B-99", Ausgesondert: true},
		},
		{name: "kein Exemplar", wert: "unbekannt", exemplare: nurUnter("B-12345", exemplar)},
		{
			name: "Fehler der Datenbank", wert: "B-12345",
			exemplare:  func(context.Context, string) (*repository.Exemplar, error) { return nil, dbFehler },
			sollFehler: dbFehler,
		},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			treffer, err := ErkenneExemplar(context.Background(), &katalogAttrappe{findeExemplar: f.exemplare}, f.wert)
			if !errors.Is(err, f.sollFehler) {
				t.Fatalf("Fehler %v, erwartet %v", err, f.sollFehler)
			}
			if !cmp.Equal(treffer, f.soll) {
				t.Errorf("Treffer %+v, erwartet %+v", treffer, f.soll)
			}
		})
	}
}
