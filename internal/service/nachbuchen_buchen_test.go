package service

import (
	"reflect"
	"testing"
	"time"

	"bibliothek/repository"
)

// TestOmniboxErgebnisAus: Das Nachbuchen antwortet in der Form der Theke. Jedes Feld der
// Buchung kommt in der Antwort an; ohne Buchung ist die Antwort leer und nicht nil.
func TestOmniboxErgebnisAus(t *testing.T) {
	if antwort := OmniboxErgebnisAus(nil); !reflect.DeepEqual(antwort, &OmniboxErgebnis{}) {
		t.Errorf("ohne Buchung: %+v, erwartet eine leere Antwort", antwort)
	}

	frist := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)
	ausleihe := "ausleihe-1"
	buch := &repository.Exemplar{ID: "ex-1", TitelID: "titel-1", BarcodeID: "B-1", Titel: "Natura 1"}
	leser := &repository.Leser{ID: "leser-1", BarcodeID: "S-1", Vorname: "Ada", Nachname: "Beispiel"}
	vorbesitzer := &repository.Leser{ID: "leser-2"}
	hinweis := &repository.AuflagenMischung{Klasse: "08G1", Auflage: "2. Auflage"}

	antwort := OmniboxErgebnisAus(&AusleihErgebnis{
		Art:                  "rueckgabe",
		Exemplar:             buch,
		Leser:                leser,
		Frist:                &frist,
		AusleiheID:           &ausleihe,
		Fremdrueckgabe:       true,
		Vorbesitzer:          vorbesitzer,
		HatVormerkung:        true,
		VormerkungTitel:      "Natura 2",
		VorgemerktFuer:       "Kollegin Muster",
		RegalfreigabeBarcode: "B-2",
		AuflagenHinweis:      hinweis,
	})
	soll := &OmniboxErgebnis{
		Art:                  "rueckgabe",
		Exemplar:             buch,
		Leser:                leser,
		Frist:                &frist,
		AusleiheID:           &ausleihe,
		Fremdrueckgabe:       true,
		Vorbesitzer:          vorbesitzer,
		HatVormerkung:        true,
		VormerkungTitel:      "Natura 2",
		VorgemerktFuer:       "Kollegin Muster",
		RegalfreigabeBarcode: "B-2",
		AuflagenHinweis:      hinweis,
	}
	if !reflect.DeepEqual(antwort, soll) {
		t.Errorf("Antwort %+v, erwartet %+v", antwort, soll)
	}
}
