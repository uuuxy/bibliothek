package service

import (
	"reflect"
	"testing"
	"time"

	"bibliothek/repository"
)

// TestLoanResultAlsOmnibox: Das Nachbuchen antwortet in der Form der Theke. Jedes Feld der
// Buchung kommt in der Antwort an; ohne Buchung ist die Antwort leer und nicht nil.
func TestLoanResultAlsOmnibox(t *testing.T) {
	if antwort := LoanResultAlsOmnibox(nil); !reflect.DeepEqual(antwort, &OmniboxResult{}) {
		t.Errorf("ohne Buchung: %+v, erwartet eine leere Antwort", antwort)
	}

	frist := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)
	ausleihe := "ausleihe-1"
	buch := &repository.BookCopy{ID: "ex-1", TitelID: "titel-1", BarcodeID: "B-1", Titel: "Natura 1"}
	leser := &repository.Student{ID: "leser-1", BarcodeID: "S-1", Vorname: "Ada", Nachname: "Beispiel"}
	vorbesitzer := &repository.Student{ID: "leser-2"}
	hinweis := &repository.AuflagenMischung{Klasse: "08G1", Auflage: "2. Auflage"}

	antwort := LoanResultAlsOmnibox(&LoanResult{
		Type:                 "rueckgabe",
		Book:                 buch,
		Student:              leser,
		DueDate:              &frist,
		LoanID:               &ausleihe,
		Fremdrueckgabe:       true,
		Vorbesitzer:          vorbesitzer,
		HasVormerkung:        true,
		VormerkungTitel:      "Natura 2",
		VormerkungUser:       "Kollegin Muster",
		RegalfreigabeBarcode: "B-2",
		AuflagenHinweis:      hinweis,
	})
	soll := &OmniboxResult{
		Type:                 "rueckgabe",
		Book:                 buch,
		Student:              leser,
		DueDate:              &frist,
		LoanID:               &ausleihe,
		Fremdrueckgabe:       true,
		Vorbesitzer:          vorbesitzer,
		HasVormerkung:        true,
		VormerkungTitel:      "Natura 2",
		VormerkungUser:       "Kollegin Muster",
		RegalfreigabeBarcode: "B-2",
		AuflagenHinweis:      hinweis,
	}
	if !reflect.DeepEqual(antwort, soll) {
		t.Errorf("Antwort %+v, erwartet %+v", antwort, soll)
	}
}
