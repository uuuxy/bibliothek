package service

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// Stubs für das präfixlose Fallback-Routing: Buch → Ausweis → Volltextsuche.
// Alle nicht überschriebenen Interface-Methoden stammen aus dem eingebetteten
// Nil-Interface und dürfen in diesen Tests nicht aufgerufen werden.
//
// Seit Migration 125 gibt es EINE Ausweis-Stufe: Der Scan sucht einen LESER, und ob das ein
// Schüler oder ein Kollege ist, steht als Art an ihm. Vorher waren es zwei Stufen über zwei
// Tabellen, und ihre Reihenfolge entschied bei einer doppelt vergebenen Nummer still, wen
// die Theke lädt.

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

func (r *routingStudentRepo) GetLeserByBarcode(_ context.Context, barcode string) (*repository.Student, error) {
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

	res, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "20240001737", StaffID: "staff", StaffRole: "admin"})
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

	res, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "gibtesnicht", StaffID: "staff", StaffRole: "admin"})
	if err != nil {
		t.Fatalf("ProcessQuery: %v", err)
	}
	if res.Type != "search_results" {
		t.Fatalf("unbekannte Eingabe muss in der Suche landen, got Type=%q", res.Type)
	}
}

// GetLeserByID bedient die Auswahl aus der Trefferliste: Dort steht die ID, nicht die
// Ausweisnummer — ein Kollege ohne gedruckten Ausweis hat gar keine.
func (r *routingStudentRepo) GetLeserByID(_ context.Context, id string) (*repository.Student, error) {
	for _, s := range r.students {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

// TestProcessQuery_LeserAusDerTrefferliste: Wer an der Theke einen Namen tippt und einen
// Treffer anklickt, muss diesen Leser geladen bekommen — auch wenn er keine
// Ausweisnummer hat. Bis zum 16.09.2026 setzte die Auswahl die Ausweisnummer in die
// Scanleiste und schickte sie los; bei einem Kollegen aus der Selbstanmeldung war die
// leer, und der Klick tat nichts.
func TestProcessQuery_LeserAusDerTrefferliste(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo: &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{
			"ohne-ausweis": {ID: "l1", Vorname: "Hendrik", Nachname: "Wendlandt", Art: "liv"},
		}},
	}

	res, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "leser:l1", StaffID: "staff", StaffRole: "admin"})
	if err != nil {
		t.Fatalf("ProcessQuery: %v", err)
	}
	if res.Type != "student" || res.Student == nil || res.Student.ID != "l1" {
		t.Fatalf("Auswahl aus der Trefferliste muss den Leser öffnen, got Type=%q Student=%v", res.Type, res.Student)
	}
}

// TestProcessQuery_LeserIDUnbekannt: Eine ID, zu der es niemanden gibt, ist ein lauter
// Fehler — sie darf nicht in die Buchtitel-Volltextsuche durchfallen. Dort stünde dann
// „keine Treffer", und an der Theke sähe es aus, als sei der Klick ins Leere gegangen.
func TestProcessQuery_LeserIDUnbekannt(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo:    &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{}},
	}

	if _, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "leser:fehlt", StaffID: "staff", StaffRole: "admin"}); err == nil {
		t.Fatal("unbekannte Leser-ID muss einen Fehler melden, nicht in die Suche fallen")
	}
}
