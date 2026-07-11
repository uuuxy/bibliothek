package service

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// Stubs für das präfixlose Fallback-Routing: Buch → Schülerausweis → Volltextsuche.
// Alle nicht überschriebenen Interface-Methoden stammen aus dem eingebetteten
// Nil-Interface und dürfen in diesen Tests nicht aufgerufen werden.

type routingBookRepo struct {
	repository.BookRepository
	copies map[string]*repository.BookCopy
}

func (r *routingBookRepo) GetCopyByBarcode(_ context.Context, barcode string) (*repository.BookCopy, error) {
	return r.copies[barcode], nil
}

func (r *routingBookRepo) SearchTitles(_ context.Context, _ string) ([]repository.BookTitle, error) {
	return nil, nil
}

type routingStudentRepo struct {
	repository.StudentRepository
	students map[string]*repository.Student
}

func (r *routingStudentRepo) GetByBarcode(_ context.Context, barcode string) (*repository.Student, error) {
	return r.students[barcode], nil
}

// TestProcessQuery_AusweisOhnePraefix bildet die Littera-Altbestand-Ausweise ab:
// nackte, längere Nummern ohne "S-"-Präfix müssen am Pult den Schüler öffnen,
// ohne das bestehende Buch-Routing oder die Volltextsuche zu stören.
func TestProcessQuery_AusweisOhnePraefix(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo: &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{
			"20240001737": {ID: "s1", BarcodeID: "20240001737", Vorname: "Mia", Nachname: "Muster"},
		}},
	}

	res, err := svc.ProcessQuery(context.Background(), "20240001737", nil, nil, false, "staff", "admin", false)
	if err != nil {
		t.Fatalf("ProcessQuery: %v", err)
	}
	if res.Type != "student" || res.Student == nil || res.Student.ID != "s1" {
		t.Fatalf("Ausweisnummer ohne Präfix muss den Schüler öffnen, got Type=%q Student=%v", res.Type, res.Student)
	}
}

// TestProcessQuery_UnbekannteNummerFaelltAufSuche stellt sicher, dass Eingaben,
// die weder Buch- noch Ausweis-Barcode sind, weiterhin in der Volltextsuche landen.
func TestProcessQuery_UnbekannteNummerFaelltAufSuche(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo:    &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{}},
	}

	res, err := svc.ProcessQuery(context.Background(), "gibtesnicht", nil, nil, false, "staff", "admin", false)
	if err != nil {
		t.Fatalf("ProcessQuery: %v", err)
	}
	if res.Type != "search_results" {
		t.Fatalf("unbekannte Eingabe muss in der Suche landen, got Type=%q", res.Type)
	}
}
