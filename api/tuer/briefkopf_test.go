package tuer

import (
	"testing"

	"bibliothek/pdf"
	"bibliothek/repository"
)

func TestSchuleAus(t *testing.T) {
	e := &repository.SystemEinstellungen{
		SchuleName: "Gesamtschule am Park", SchuleStrasse: "Parkweg 3", SchulePLZ: "35390", SchuleOrt: "Gießen",
	}
	want := pdf.SchuleInfo{Name: "Gesamtschule am Park", Strasse: "Parkweg 3", PLZ: "35390", Ort: "Gießen"}
	if got := SchuleAus(e); got != want {
		t.Errorf("SchuleAus(…) = %+v, erwartet %+v", got, want)
	}
}
