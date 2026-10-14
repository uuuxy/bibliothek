package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"
)

// TestEchteKartennummerFindetDenSchueler prüft die gemessene Nummer eines echten
// Schülerausweises: aufgedruckt "[0395] 37", gescannt "B97601826457".
//
// Sie fängt mit "B" an, und "B-" ist bei uns die Vorsilbe eines Buchs. Zwischen „funktioniert"
// und „jeder Ausweis landet beim Buch" liegt genau ein Bindestrich.
func TestEchteKartennummerFindetDenSchueler(t *testing.T) {
	const karte = "B97601826457"
	svc := &omniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen: &weicheLeserAttrappe{leser: map[string]*repository.Leser{
			karte: {ID: "s1", Vorname: "Nina", Nachname: "Berger"},
		}},
	}
	resp, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: karte})
	if err != nil {
		t.Fatalf("die echte Kartennummer muss aufloesen: %v", err)
	}
	if resp.Art != "student" || resp.Leser == nil {
		t.Fatalf("Schueler erwartet, geliefert: Type=%q Student=%v", resp.Art, resp.Leser)
	}
}

// TestLehrerausweisOhnePraefix: Ein Lehrerausweis ohne Vorsilbe lädt die Lehrkraft. Littera
// kennt keinen Unterschied zwischen Schüler- und Lehrerausweis, auf dem Aufdruck steht nur ein
// anderes Wort; fragte die Auflösung ohne Vorsilbe nur Bücher und Schüler ab, liefe der Ausweis
// bis in die Volltextsuche und meldete „keine Treffer". Beide stehen in der Tabelle leser; die
// Antwort heißt darum für beide „student" und trägt die Art.
func TestLehrerausweisOhnePraefix(t *testing.T) {
	const karte = "B97601826458"
	svc := &omniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen: &weicheLeserAttrappe{leser: map[string]*repository.Leser{
			karte: {ID: "l1", Vorname: "Anna", Nachname: "Berg", Art: "lehrkraft"},
		}},
	}
	resp, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: karte})
	if err != nil {
		t.Fatalf("der Lehrerausweis muss ohne Praefix aufloesen: %v", err)
	}
	if resp.Art != "student" || resp.Leser == nil {
		t.Fatalf("Leser erwartet, geliefert: Type=%q Student=%v", resp.Art, resp.Leser)
	}
	if resp.Leser.Nachname != "Berg" || resp.Leser.Art != "lehrkraft" {
		t.Errorf("falsche Person geladen: %+v", resp.Leser)
	}
}

// TestUnbekannteKarteLandetInDerSuche: Die Ausweis-Stufe darf den Rückfall nicht schlucken. Ein
// getippter Titel ist genau dieser Fall — er ist weder Buchbarcode noch Ausweis und muss in der
// Volltextsuche landen.
func TestUnbekannteKarteLandetInDerSuche(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen:   &weicheLeserAttrappe{leser: map[string]*repository.Leser{}},
	}
	for _, eingabe := range []string{"Goethe", "Die Leiden des jungen Werther"} {
		resp, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: eingabe})
		if err != nil {
			t.Fatalf("%q muss zur Volltextsuche werden: %v", eingabe, err)
		}
		if resp.Art != "search_results" {
			t.Errorf("%q: Volltextsuche erwartet, geliefert Type=%q", eingabe, resp.Art)
		}
		if resp.Leser != nil {
			t.Errorf("%q: ohne Treffer darf keine Person gemeldet werden", eingabe)
		}
	}
}

// TestLitteraErsatznummerFindetDenSchueler: Die Littera-Übernahme gibt einem Schüler ohne
// eindeutige Ausweisnummer „L-<Littera-Nummer>" (internal/littera, ausweis). „L-" sagt nur
// „Ausweis", nicht „Lehrkraft": Der Schüler muss über den neu gedruckten Ausweis zu laden sein.
func TestLitteraErsatznummerFindetDenSchueler(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen: &weicheLeserAttrappe{leser: map[string]*repository.Leser{
			"L-4711": {ID: "s1", BarcodeID: "L-4711", Vorname: "Ersatz", Nachname: "Nummer"},
		}},
	}
	resp, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "L-4711"})
	if err != nil {
		t.Fatalf("der Schüler mit der Ersatznummer muss laden: %v", err)
	}
	if resp.Art != "student" || resp.Leser == nil || resp.Leser.ID != "s1" {
		t.Fatalf("Schüler erwartet, geliefert: Type=%q Student=%v", resp.Art, resp.Leser)
	}
}

// TestVorsilbeEntscheidetNichtDieTabelle: „S-" und „L-" sagen nur „Ausweis". Littera kennt
// die Vorsilben nicht, und auf dem Ausweis stehen sie auch nicht — eine Lehrkraft mit
// S-Nummer muss laden.
func TestVorsilbeEntscheidetNichtDieTabelle(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen: &weicheLeserAttrappe{leser: map[string]*repository.Leser{
			"S-0815": {ID: "l1", Vorname: "Anna", Nachname: "Berg", Art: "lehrkraft"},
		}},
	}
	resp, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "S-0815"})
	if err != nil {
		t.Fatalf("die Lehrkraft mit S-Nummer muss laden: %v", err)
	}
	if resp.Art != "student" || resp.Leser == nil || resp.Leser.ID != "l1" {
		t.Fatalf("Leser erwartet, geliefert: Type=%q Student=%v", resp.Art, resp.Leser)
	}
}

// TestUnbekannterAusweisMeldetSichLaut: Eine S-/L-Nummer ohne Person bleibt ein Fehler —
// sie verschwindet nicht leise in der Volltextsuche.
func TestUnbekannterAusweisMeldetSichLaut(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen:   &weicheLeserAttrappe{leser: map[string]*repository.Leser{}},
	}
	for _, code := range []string{"S-404", "L-404"} {
		resp, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: code})
		if !errors.Is(err, ErrNichtGefunden) {
			t.Errorf("%s: ErrNichtGefunden erwartet, geliefert %v (Type %q)", code, err, resp.Art)
		}
	}
}
