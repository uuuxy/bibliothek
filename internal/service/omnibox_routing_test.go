package service

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// Attrappen für die Weiche ohne Vorsilbe: Buch → Ausweis → Volltextsuche. Alle nicht
// überschriebenen Methoden stammen aus der eingebetteten leeren Schnittstelle und dürfen in
// diesen Tests nicht aufgerufen werden.
//
// Es gibt eine Ausweis-Stufe: Der Scan sucht einen Leser, und ob das ein Schüler oder ein
// Kollege ist, steht als Art an ihm. Zwei Stufen über zwei Tabellen ließen ihre Reihenfolge bei
// einer doppelt vergebenen Nummer still entscheiden, wen die Theke lädt.

type weicheKatalogAttrappe struct {
	repository.KatalogAbfragen
	exemplare map[string]*repository.Exemplar
}

func (r *weicheKatalogAttrappe) FindeExemplar(_ context.Context, barcode string) (*repository.Exemplar, error) {
	return r.exemplare[barcode], nil
}

func (r *weicheKatalogAttrappe) FindeTitel(_ context.Context, _ string) ([]repository.Titel, error) {
	return nil, nil
}

type weicheLeserAttrappe struct {
	repository.LeserAbfragen
	leser map[string]*repository.Leser
}

func (r *weicheLeserAttrappe) FindeZuBarcode(_ context.Context, barcode string) (*repository.Leser, error) {
	return r.leser[barcode], nil
}

// TestVerarbeiteEingabe_AusweisOhnePraefix bildet die Littera-Altbestand-Ausweise ab:
// nackte, längere Nummern ohne "S-"-Präfix müssen am Pult den Schüler öffnen,
// ohne das bestehende Buch-Routing oder die Volltextsuche zu stören.
func TestVerarbeiteEingabe_AusweisOhnePraefix(t *testing.T) {
	svc := &OmniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen: &weicheLeserAttrappe{leser: map[string]*repository.Leser{
			"20240001737": {ID: "s1", BarcodeID: "20240001737", Vorname: "Mia", Nachname: "Muster"},
		}},
	}

	res, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "20240001737", BearbeiterID: "staff"})
	if err != nil {
		t.Fatalf("VerarbeiteEingabe: %v", err)
	}
	if res.Art != "student" || res.Leser == nil || res.Leser.ID != "s1" {
		t.Fatalf("VerarbeiteEingabe(20240001737) = Art %q, Leser %v, erwartet den Schüler s1", res.Art, res.Leser)
	}
}

// TestVerarbeiteEingabe_UnbekannteNummerFaelltAufSuche stellt sicher, dass Eingaben,
// die weder Buch- noch Ausweis-Barcode sind, weiterhin in der Volltextsuche landen.
func TestVerarbeiteEingabe_UnbekannteNummerFaelltAufSuche(t *testing.T) {
	svc := &OmniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen:   &weicheLeserAttrappe{leser: map[string]*repository.Leser{}},
	}

	res, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "gibtesnicht", BearbeiterID: "staff"})
	if err != nil {
		t.Fatalf("VerarbeiteEingabe: %v", err)
	}
	if res.Art != "search_results" {
		t.Fatalf("VerarbeiteEingabe(gibtesnicht) = Art %q, erwartet die Suche", res.Art)
	}
}

// Finde bedient die Auswahl aus der Trefferliste: Dort steht die ID, nicht die
// Ausweisnummer — ein Kollege ohne gedruckten Ausweis hat gar keine.
func (r *weicheLeserAttrappe) Finde(_ context.Context, id string) (*repository.Leser, error) {
	for _, s := range r.leser {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

// TestVerarbeiteEingabe_LeserAusDerTrefferliste: Wer an der Theke einen Namen tippt und einen
// Treffer anklickt, muss diesen Leser geladen bekommen — auch wenn er keine Ausweisnummer hat.
// Die Auswahl schickt deshalb die Kennung, nicht die Ausweisnummer: Bei einem Kollegen aus der
// Selbstanmeldung ist die leer, und der Klick täte nichts.
func TestVerarbeiteEingabe_LeserAusDerTrefferliste(t *testing.T) {
	svc := &OmniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen: &weicheLeserAttrappe{leser: map[string]*repository.Leser{
			"ohne-ausweis": {ID: "l1", Vorname: "Hendrik", Nachname: "Wendlandt", Art: "liv"},
		}},
	}

	res, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "leser:l1", BearbeiterID: "staff"})
	if err != nil {
		t.Fatalf("VerarbeiteEingabe: %v", err)
	}
	if res.Art != "student" || res.Leser == nil || res.Leser.ID != "l1" {
		t.Fatalf("VerarbeiteEingabe(leser:l1) = Art %q, Leser %v, erwartet den Leser l1", res.Art, res.Leser)
	}
}

// TestVerarbeiteEingabe_LeserIDUnbekannt: Eine ID, zu der es niemanden gibt, ist ein lauter
// Fehler — sie darf nicht in die Buchtitel-Volltextsuche durchfallen. Dort stünde dann
// „keine Treffer", und an der Theke sähe es aus, als sei der Klick ins Leere gegangen.
func TestVerarbeiteEingabe_LeserIDUnbekannt(t *testing.T) {
	svc := &OmniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen:   &weicheLeserAttrappe{leser: map[string]*repository.Leser{}},
	}

	if _, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "leser:fehlt", BearbeiterID: "staff"}); err == nil {
		t.Fatal("unbekannte Leser-ID muss einen Fehler melden, nicht in die Suche fallen")
	}
}
