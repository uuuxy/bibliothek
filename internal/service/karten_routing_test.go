package service

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// TestEchteKartennummerFindetDenSchueler prueft die GEMESSENE Nummer eines echten
// Schuelerausweises: aufgedruckt "[0395] 37", gescannt "B97601826457".
//
// Die faengt mit "B" an — und "B-" ist bei uns das Buch-Praefix. Der Abstand zwischen
// "funktioniert" und "jeder Ausweis landet im Buch-Handler" ist genau ein Bindestrich.
func TestEchteKartennummerFindetDenSchueler(t *testing.T) {
	const karte = "B97601826457"
	svc := &defaultOmniboxService{
		bookRepo: &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		userRepo: &routingUserRepo{lehrer: map[string]*repository.User{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{
			karte: {ID: "s1", Vorname: "Peter", Nachname: "Flasch"},
		}},
	}
	resp, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: karte})
	if err != nil {
		t.Fatalf("die echte Kartennummer muss aufloesen: %v", err)
	}
	if resp.Type != "student" || resp.Student == nil {
		t.Fatalf("Schueler erwartet, geliefert: Type=%q Student=%v", resp.Type, resp.Student)
	}
}

// TestLehrerausweisOhnePraefix ist die Luecke, die lange offen war.
//
// Lehrkraefte stehen in `benutzer`, Schueler in `schueler`. Die praefixlose Aufloesung
// fragte nur Buecher und Schueler ab — ein gescannter Lehrerausweis lief deshalb bis in
// die Volltextsuche und meldete „keine Treffer". Nicht weil die Karte falsch waere:
// Littera kennt gar keinen Unterschied zwischen Schueler- und Lehrerausweis, es steht nur
// ein anderes Wort auf dem Aufdruck.
func TestLehrerausweisOhnePraefix(t *testing.T) {
	const karte = "B97601826458"
	svc := &defaultOmniboxService{
		bookRepo:    &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{}},
		userRepo: &routingUserRepo{lehrer: map[string]*repository.User{
			karte: {ID: "l1", Vorname: "Anna", Nachname: "Berg"},
		}},
	}
	resp, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: karte})
	if err != nil {
		t.Fatalf("der Lehrerausweis muss ohne Praefix aufloesen: %v", err)
	}
	if resp.Type != "teacher" || resp.Teacher == nil {
		t.Fatalf("Lehrkraft erwartet, geliefert: Type=%q Teacher=%v", resp.Type, resp.Teacher)
	}
	if resp.Teacher.Nachname != "Berg" {
		t.Errorf("falsche Lehrkraft geladen: %+v", resp.Teacher)
	}
}

// TestUnbekannteKarteLandetInDerSuche: die neue Stufe darf den Rueckfall nicht schlucken.
// ErrNotFound aus handleTeacherAction ist hier kein Fehler, sondern der Uebergang.
func TestUnbekannteKarteLandetInDerSuche(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo:    &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{}},
		userRepo:    &routingUserRepo{lehrer: map[string]*repository.User{}},
	}
	resp, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "Goethe"})
	if err != nil {
		t.Fatalf("unbekannte Eingabe muss zur Volltextsuche werden: %v", err)
	}
	if resp.Type == "teacher" {
		t.Fatal("ohne Treffer darf keine Lehrkraft gemeldet werden")
	}
}
