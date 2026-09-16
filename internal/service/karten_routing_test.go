package service

import (
	"context"
	"errors"
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
// Lehrkraefte standen in `benutzer`, Schueler in `schueler`. Die praefixlose Aufloesung
// fragte nur Buecher und Schueler ab — ein gescannter Lehrerausweis lief deshalb bis in
// die Volltextsuche und meldete „keine Treffer". Nicht weil die Karte falsch waere:
// Littera kennt gar keinen Unterschied zwischen Schueler- und Lehrerausweis, es steht nur
// ein anderes Wort auf dem Aufdruck. Seit Migration 125 stehen beide in EINER Tabelle; die
// Antwort heisst darum fuer beide „student" und traegt die Art.
func TestLehrerausweisOhnePraefix(t *testing.T) {
	const karte = "B97601826458"
	svc := &defaultOmniboxService{
		bookRepo: &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{
			karte: {ID: "l1", Vorname: "Anna", Nachname: "Berg", Art: "lehrkraft"},
		}},
	}
	resp, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: karte})
	if err != nil {
		t.Fatalf("der Lehrerausweis muss ohne Praefix aufloesen: %v", err)
	}
	if resp.Type != "student" || resp.Student == nil {
		t.Fatalf("Leser erwartet, geliefert: Type=%q Student=%v", resp.Type, resp.Student)
	}
	if resp.Student.Nachname != "Berg" || resp.Student.Art != "lehrkraft" {
		t.Errorf("falsche Person geladen: %+v", resp.Student)
	}
}

// TestUnbekannteKarteLandetInDerSuche: die Ausweis-Stufe darf den Rueckfall nicht
// schlucken. Ein getippter Titel ist genau dieser Fall — er ist weder Buchbarcode noch
// Ausweis und muss in der Volltextsuche landen (Peters Frage vom 16.09.2026).
func TestUnbekannteKarteLandetInDerSuche(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo:    &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{}},
	}
	for _, eingabe := range []string{"Goethe", "Die Leiden des jungen Werther"} {
		resp, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: eingabe})
		if err != nil {
			t.Fatalf("%q muss zur Volltextsuche werden: %v", eingabe, err)
		}
		if resp.Type != "search_results" {
			t.Errorf("%q: Volltextsuche erwartet, geliefert Type=%q", eingabe, resp.Type)
		}
		if resp.Student != nil {
			t.Errorf("%q: ohne Treffer darf keine Person gemeldet werden", eingabe)
		}
	}
}

// TestLitteraErsatznummerFindetDenSchueler: Die Littera-Übernahme gibt einem Schüler ohne
// eindeutige Ausweisnummer „L-<Littera-Nummer>" (internal/littera, ausweis). Bis zum
// 15.09.2026 schickte die Theke jedes „L-" in den Lehrkraft-Zweig — der Schüler war weder
// über den neu gedruckten Ausweis noch über die Namenssuche ladbar (OFFEN.md 5.15).
func TestLitteraErsatznummerFindetDenSchueler(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo: &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{
			"L-4711": {ID: "s1", BarcodeID: "L-4711", Vorname: "Ersatz", Nachname: "Nummer"},
		}},
	}
	resp, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "L-4711"})
	if err != nil {
		t.Fatalf("der Schüler mit der Ersatznummer muss laden: %v", err)
	}
	if resp.Type != "student" || resp.Student == nil || resp.Student.ID != "s1" {
		t.Fatalf("Schüler erwartet, geliefert: Type=%q Student=%v", resp.Type, resp.Student)
	}
}

// TestVorsilbeEntscheidetNichtDieTabelle: „S-" und „L-" sagen nur „Ausweis". Littera kennt
// die Vorsilben nicht, und auf dem Ausweis stehen sie auch nicht — eine Lehrkraft mit
// S-Nummer muss laden.
func TestVorsilbeEntscheidetNichtDieTabelle(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo: &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{
			"S-0815": {ID: "l1", Vorname: "Anna", Nachname: "Berg", Art: "lehrkraft"},
		}},
	}
	resp, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "S-0815"})
	if err != nil {
		t.Fatalf("die Lehrkraft mit S-Nummer muss laden: %v", err)
	}
	if resp.Type != "student" || resp.Student == nil || resp.Student.ID != "l1" {
		t.Fatalf("Leser erwartet, geliefert: Type=%q Student=%v", resp.Type, resp.Student)
	}
}

// TestUnbekannterAusweisMeldetSichLaut: Eine S-/L-Nummer ohne Person bleibt ein Fehler —
// sie verschwindet nicht leise in der Volltextsuche.
func TestUnbekannterAusweisMeldetSichLaut(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo:    &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{}},
	}
	for _, code := range []string{"S-404", "L-404"} {
		resp, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: code})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: ErrNotFound erwartet, geliefert %v (Type %q)", code, err, resp.Type)
		}
	}
}
