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
	repository.KatalogAbfragen
	copies map[string]*repository.Exemplar
}

func (r *routingBookRepo) FindeExemplar(_ context.Context, barcode string) (*repository.Exemplar, error) {
	return r.copies[barcode], nil
}

func (r *routingBookRepo) FindeTitel(_ context.Context, _ string) ([]repository.Titel, error) {
	return nil, nil
}

type routingStudentRepo struct {
	repository.LeserAbfragen
	students map[string]*repository.Leser
}

func (r *routingStudentRepo) FindeZuBarcode(_ context.Context, barcode string) (*repository.Leser, error) {
	return r.students[barcode], nil
}

// TestVerarbeiteEingabe_AusweisOhnePraefix bildet die Littera-Altbestand-Ausweise ab:
// nackte, längere Nummern ohne "S-"-Präfix müssen am Pult den Schüler öffnen,
// ohne das bestehende Buch-Routing oder die Volltextsuche zu stören.
func TestVerarbeiteEingabe_AusweisOhnePraefix(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &routingBookRepo{copies: map[string]*repository.Exemplar{}},
		leserAbfragen: &routingStudentRepo{students: map[string]*repository.Leser{
			"20240001737": {ID: "s1", BarcodeID: "20240001737", Vorname: "Mia", Nachname: "Muster"},
		}},
	}

	res, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "20240001737", BearbeiterID: "staff"})
	if err != nil {
		t.Fatalf("VerarbeiteEingabe: %v", err)
	}
	if res.Art != "student" || res.Leser == nil || res.Leser.ID != "s1" {
		t.Fatalf("Ausweisnummer ohne Präfix muss den Schüler öffnen, got Type=%q Student=%v", res.Art, res.Leser)
	}
}

// TestVerarbeiteEingabe_UnbekannteNummerFaelltAufSuche stellt sicher, dass Eingaben,
// die weder Buch- noch Ausweis-Barcode sind, weiterhin in der Volltextsuche landen.
func TestVerarbeiteEingabe_UnbekannteNummerFaelltAufSuche(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &routingBookRepo{copies: map[string]*repository.Exemplar{}},
		leserAbfragen:   &routingStudentRepo{students: map[string]*repository.Leser{}},
	}

	res, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "gibtesnicht", BearbeiterID: "staff"})
	if err != nil {
		t.Fatalf("VerarbeiteEingabe: %v", err)
	}
	if res.Art != "search_results" {
		t.Fatalf("unbekannte Eingabe muss in der Suche landen, got Type=%q", res.Art)
	}
}

// Finde bedient die Auswahl aus der Trefferliste: Dort steht die ID, nicht die
// Ausweisnummer — ein Kollege ohne gedruckten Ausweis hat gar keine.
func (r *routingStudentRepo) Finde(_ context.Context, id string) (*repository.Leser, error) {
	for _, s := range r.students {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

// TestVerarbeiteEingabe_LeserAusDerTrefferliste: Wer an der Theke einen Namen tippt und einen
// Treffer anklickt, muss diesen Leser geladen bekommen — auch wenn er keine
// Ausweisnummer hat. Bis zum 16.09.2026 setzte die Auswahl die Ausweisnummer in die
// Scanleiste und schickte sie los; bei einem Kollegen aus der Selbstanmeldung war die
// leer, und der Klick tat nichts.
func TestVerarbeiteEingabe_LeserAusDerTrefferliste(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &routingBookRepo{copies: map[string]*repository.Exemplar{}},
		leserAbfragen: &routingStudentRepo{students: map[string]*repository.Leser{
			"ohne-ausweis": {ID: "l1", Vorname: "Hendrik", Nachname: "Wendlandt", Art: "liv"},
		}},
	}

	res, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "leser:l1", BearbeiterID: "staff"})
	if err != nil {
		t.Fatalf("VerarbeiteEingabe: %v", err)
	}
	if res.Art != "student" || res.Leser == nil || res.Leser.ID != "l1" {
		t.Fatalf("Auswahl aus der Trefferliste muss den Leser öffnen, got Type=%q Student=%v", res.Art, res.Leser)
	}
}

// TestVerarbeiteEingabe_LeserIDUnbekannt: Eine ID, zu der es niemanden gibt, ist ein lauter
// Fehler — sie darf nicht in die Buchtitel-Volltextsuche durchfallen. Dort stünde dann
// „keine Treffer", und an der Theke sähe es aus, als sei der Klick ins Leere gegangen.
func TestVerarbeiteEingabe_LeserIDUnbekannt(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &routingBookRepo{copies: map[string]*repository.Exemplar{}},
		leserAbfragen:   &routingStudentRepo{students: map[string]*repository.Leser{}},
	}

	if _, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "leser:fehlt", BearbeiterID: "staff"}); err == nil {
		t.Fatal("unbekannte Leser-ID muss einen Fehler melden, nicht in die Suche fallen")
	}
}
