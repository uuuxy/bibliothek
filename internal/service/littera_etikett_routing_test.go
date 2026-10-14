package service

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// etikettAusleihAttrappe hält fest, mit welchem Exemplar die Ausleih-Logik angesteuert
// wurde — mehr Loan-Verhalten braucht der Routing-Beweis nicht.
type etikettAusleihAttrappe struct {
	AusleihDienst
	zuletzt *repository.Exemplar
}

func (l *etikettAusleihAttrappe) NimmZurueck(_ context.Context, copy *repository.Exemplar, _ string) (*AusleihErgebnis, error) {
	l.zuletzt = copy
	return &AusleihErgebnis{Art: "rueckgabe", Exemplar: copy}, nil
}

// TestVerarbeiteEingabe_LitteraEtikettWirdRueckgerechnet bildet den Thekenfall ab:
// Der Scanner liefert die EAN-13 vom Littera-Etikett (echter Messwert), im
// System steht das Exemplar unter seiner kurzen Mediennummer. Die Auflösung
// muss beim richtigen Buch landen — und eine unbekannte EAN weiterhin in der
// Volltextsuche, nicht in einem falschen Treffer.
func TestVerarbeiteEingabe_LitteraEtikettWirdRueckgerechnet(t *testing.T) {
	exemplar := &repository.Exemplar{ID: "ex1", BarcodeID: "58968", Titel: "Testbuch", IstAusleihbar: true}
	loanSvc := &etikettAusleihAttrappe{}
	svc := &omniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{"58968": exemplar}},
		leserAbfragen:   &weicheLeserAttrappe{leser: map[string]*repository.Leser{}},
		ausleihe:        loanSvc,
	}

	res, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "5896800039556", BearbeiterID: "staff"})
	if err != nil {
		t.Fatalf("VerarbeiteEingabe: %v", err)
	}
	if res.Art != "rueckgabe" || loanSvc.zuletzt == nil || loanSvc.zuletzt.ID != "ex1" {
		t.Fatalf("Littera-EAN muss beim Exemplar 58968 landen, got Type=%q zuletzt=%v", res.Art, loanSvc.zuletzt)
	}
}

// TestVerarbeiteEingabe_UnbekanntesEtikettFaelltAufSuche: Eine formal gültige
// Littera-EAN, deren Mediennummer NICHT im Bestand ist, darf keine Aktion
// auslösen — sie läuft wie bisher in die Volltextsuche.
func TestVerarbeiteEingabe_UnbekanntesEtikettFaelltAufSuche(t *testing.T) {
	svc := &omniboxDienst{
		katalogAbfragen: &weicheKatalogAttrappe{exemplare: map[string]*repository.Exemplar{}},
		leserAbfragen:   &weicheLeserAttrappe{leser: map[string]*repository.Leser{}},
	}

	res, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: "1241170039561", BearbeiterID: "staff"})
	if err != nil {
		t.Fatalf("VerarbeiteEingabe: %v", err)
	}
	if res.Art != "search_results" {
		t.Fatalf("unbekannte Littera-EAN muss in der Suche landen, got Type=%q", res.Art)
	}
}
