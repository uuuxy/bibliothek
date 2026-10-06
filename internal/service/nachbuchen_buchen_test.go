package service

import (
	"reflect"
	"testing"
	"time"

	"bibliothek/repository"
)

func TestLoanResultAlsOmnibox(t *testing.T) {
	now := time.Now()
	loanID := "test-loan-id"
	vorbesitzer := &repository.Student{ID: "vorbesitzer-id"}

	tests := []struct {
		name string
		in   *LoanResult
		want *OmniboxResult
	}{
		{
			name: "nil input",
			in:   nil,
			want: &OmniboxResult{},
		},
		{
			name: "fully populated",
			in: &LoanResult{
				Type: "ausleihe",
				Book: &repository.BookCopy{
					ID:        "book-id",
					TitelID:   "titel-id",
					BarcodeID: "B123",
					Titel:     "Test Book",
				},
				Student: &repository.Student{
					ID:        "student-id",
					BarcodeID: "S123",
					Vorname:   "Max",
					Nachname:  "Mustermann",
				},
				DueDate:              &now,
				LoanID:               &loanID,
				Fremdrueckgabe:       true,
				Vorbesitzer:          vorbesitzer,
				HasVormerkung:        true,
				VormerkungTitel:      "Vorgemerktes Buch",
				VormerkungUser:       "Anna Schmidt",
				RegalfreigabeBarcode: "R456",
				AuflagenHinweis:      &repository.AuflagenMischung{Klasse: "10A", Auflage: "1. Auflage"},
			},
			want: &OmniboxResult{
				Type: "ausleihe",
				Book: &repository.BookCopy{
					ID:        "book-id",
					TitelID:   "titel-id",
					BarcodeID: "B123",
					Titel:     "Test Book",
				},
				Student: &repository.Student{
					ID:        "student-id",
					BarcodeID: "S123",
					Vorname:   "Max",
					Nachname:  "Mustermann",
				},
				DueDate:              &now,
				LoanID:               &loanID,
				Fremdrueckgabe:       true,
				Vorbesitzer:          vorbesitzer,
				HasVormerkung:        true,
				VormerkungTitel:      "Vorgemerktes Buch",
				VormerkungUser:       "Anna Schmidt",
				RegalfreigabeBarcode: "R456",
				AuflagenHinweis:      &repository.AuflagenMischung{Klasse: "10A", Auflage: "1. Auflage"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LoanResultAlsOmnibox(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoanResultAlsOmnibox() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
