package bestandsbuch

import (
	"testing"
	"time"

	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
)

func TestBarcodeText(t *testing.T) {
	for _, f := range []struct {
		name     string
		barcodes []string
		want     string
	}{
		{"einer", []string{"B-00100"}, "B-00100"},
		{"zwei fortlaufende bleiben einzeln", []string{"B-00101", "B-00100"}, "B-00100, B-00101"},
		{"drei fortlaufende sind ein Bereich", []string{"B-00102", "B-00100", "B-00101"}, "B-00100 – B-00102"},
		{"Lücke teilt", []string{"B-00100", "B-00101", "B-00102", "B-00104", "B-00105"},
			"B-00100 – B-00102, B-00104, B-00105"},
		{"nach der Zahl, nicht nach dem Text", []string{"B-99", "B-100", "B-101", "B-9"}, "B-9, B-99, B-100, B-101"},
		{"andere Vorsilbe folgt nicht", []string{"A-00100", "B-00101", "B-00102"}, "A-00100, B-00101, B-00102"},
		{"andere Stellenzahl folgt nicht", []string{"B-0099", "B-00100", "B-00101"}, "B-0099, B-00100, B-00101"},
		{"ohne Ziffern am Ende", []string{"XYZ", "B-1"}, "B-1, XYZ"},
		{"Littera-Nummern ohne Vorsilbe", []string{"012347", "012345", "012346"}, "012345 – 012347"},
	} {
		if got := barcodeText(f.barcodes); got != f.want {
			t.Errorf("%s: barcodeText(%q) = %q, erwartet %q", f.name, f.barcodes, got, f.want)
		}
	}
}

// Eine Zeile je Tag, Titel und Lieferant; ein anderer Lieferant oder ein anderer Tag ist eine
// eigene Zeile. Die Reihenfolge ist die der Abfrage.
func TestZugaenge_FasstTagTitelUndLieferantZusammen(t *testing.T) {
	tag := time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC)
	land := mitteltopf.Land
	zeilen := []repository.ZugangsZeile{
		{Datum: tag, Barcode: "B-00101", Titel: "Atlas", Lieferant: "Naacher", Topf: land},
		{Datum: tag, Barcode: "B-00100", Titel: "Atlas", Lieferant: "Naacher", Topf: land},
		{Datum: tag, Barcode: "B-00102", Titel: "Atlas", Lieferant: "Naacher", Topf: land},
		{Datum: tag, Barcode: "B-00200", Titel: "Atlas", Lieferant: "Schulbuch-Center", Topf: land},
		{Datum: tag, Barcode: "B-00300", Titel: "Lesebuch", Lieferant: "Naacher", Topf: land},
		{Datum: tag.AddDate(0, 0, 1), Barcode: "B-00103", Titel: "Atlas", Lieferant: "Naacher", Topf: land},
	}
	want := []Zugang{
		{Datum: tag, Titel: "Atlas", Lieferant: "Naacher", Anzahl: 3, Barcodes: "B-00100 – B-00102", Topf: land},
		{Datum: tag, Titel: "Atlas", Lieferant: "Schulbuch-Center", Anzahl: 1, Barcodes: "B-00200", Topf: land},
		{Datum: tag, Titel: "Lesebuch", Lieferant: "Naacher", Anzahl: 1, Barcodes: "B-00300", Topf: land},
		{Datum: tag.AddDate(0, 0, 1), Titel: "Atlas", Lieferant: "Naacher", Anzahl: 1, Barcodes: "B-00103", Topf: land},
	}
	if diff := cmp.Diff(want, Zugaenge(zeilen)); diff != "" {
		t.Errorf("Zugaenge (-erwartet +bekommen):\n%s", diff)
	}
}

// Im Abgangsbuch trennen auch Grund und Signatur; der Tag ist der Kalendertag, nicht der
// Zeitpunkt.
func TestAbgaenge_FasstTagTitelSignaturUndGrundZusammen(t *testing.T) {
	morgens := time.Date(2026, time.April, 12, 8, 0, 0, 0, time.UTC)
	abends := time.Date(2026, time.April, 12, 18, 0, 0, 0, time.UTC)
	topf := mitteltopf.Schultraeger
	zeilen := []repository.AbgangsZeile{
		{Datum: morgens, Barcode: "B-1", Titel: "Atlas", Signatur: "Erd 1", Grund: "BESCHAEDIGUNG", GrundText: "Beschädigt", Topf: topf},
		{Datum: abends, Barcode: "B-2", Titel: "Atlas", Signatur: "Erd 1", Grund: "BESCHAEDIGUNG", GrundText: "Beschädigt", Topf: topf},
		{Datum: abends, Barcode: "B-3", Titel: "Atlas", Signatur: "Erd 1", Grund: "VERLUST", GrundText: "Verlust", Topf: topf},
		{Datum: abends, Barcode: "B-4", Titel: "Atlas", Signatur: "Erd 2", Grund: "VERLUST", GrundText: "Verlust", Topf: topf},
	}
	want := []Abgang{
		{Datum: morgens, Titel: "Atlas", Signatur: "Erd 1", Grund: "BESCHAEDIGUNG", GrundText: "Beschädigt", Anzahl: 2, Barcodes: "B-1, B-2", Topf: topf},
		{Datum: abends, Titel: "Atlas", Signatur: "Erd 1", Grund: "VERLUST", GrundText: "Verlust", Anzahl: 1, Barcodes: "B-3", Topf: topf},
		{Datum: abends, Titel: "Atlas", Signatur: "Erd 2", Grund: "VERLUST", GrundText: "Verlust", Anzahl: 1, Barcodes: "B-4", Topf: topf},
	}
	if diff := cmp.Diff(want, Abgaenge(zeilen)); diff != "" {
		t.Errorf("Abgaenge (-erwartet +bekommen):\n%s", diff)
	}
}
