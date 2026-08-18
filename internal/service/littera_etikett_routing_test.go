package service

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// etikettLoanSvc hält fest, mit welchem Exemplar die Ausleih-Logik angesteuert
// wurde — mehr Loan-Verhalten braucht der Routing-Beweis nicht.
type etikettLoanSvc struct {
	LoanService
	zuletzt *repository.BookCopy
}

func (l *etikettLoanSvc) HandleSimpleReturn(_ context.Context, copy *repository.BookCopy, _ string, _ string) (*LoanResult, error) {
	l.zuletzt = copy
	return &LoanResult{Type: "rueckgabe", Book: copy}, nil
}

// TestProcessQuery_LitteraEtikettWirdRueckgerechnet bildet den Thekenfall ab:
// Der Scanner liefert die EAN-13 vom Littera-Etikett (echter Messwert), im
// System steht das Exemplar unter seiner kurzen Mediennummer. Die Auflösung
// muss beim richtigen Buch landen — und eine unbekannte EAN weiterhin in der
// Volltextsuche, nicht in einem falschen Treffer.
func TestProcessQuery_LitteraEtikettWirdRueckgerechnet(t *testing.T) {
	exemplar := &repository.BookCopy{ID: "ex1", BarcodeID: "58968", Titel: "Testbuch", IstAusleihbar: true}
	loanSvc := &etikettLoanSvc{}
	svc := &defaultOmniboxService{
		bookRepo:    &routingBookRepo{copies: map[string]*repository.BookCopy{"58968": exemplar}},
		userRepo:    &routingUserRepo{lehrer: map[string]*repository.User{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{}},
		loanSvc:     loanSvc,
	}

	res, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "5896800039556", StaffID: "staff", StaffRole: "admin"})
	if err != nil {
		t.Fatalf("ProcessQuery: %v", err)
	}
	if res.Type != "rueckgabe" || loanSvc.zuletzt == nil || loanSvc.zuletzt.ID != "ex1" {
		t.Fatalf("Littera-EAN muss beim Exemplar 58968 landen, got Type=%q zuletzt=%v", res.Type, loanSvc.zuletzt)
	}
}

// TestProcessQuery_UnbekanntesEtikettFaelltAufSuche: Eine formal gültige
// Littera-EAN, deren Mediennummer NICHT im Bestand ist, darf keine Aktion
// auslösen — sie läuft wie bisher in die Volltextsuche.
func TestProcessQuery_UnbekanntesEtikettFaelltAufSuche(t *testing.T) {
	svc := &defaultOmniboxService{
		bookRepo:    &routingBookRepo{copies: map[string]*repository.BookCopy{}},
		userRepo:    &routingUserRepo{lehrer: map[string]*repository.User{}},
		studentRepo: &routingStudentRepo{students: map[string]*repository.Student{}},
	}

	res, err := svc.ProcessQuery(context.Background(), OmniboxQuery{Query: "1241170039561", StaffID: "staff", StaffRole: "admin"})
	if err != nil {
		t.Fatalf("ProcessQuery: %v", err)
	}
	if res.Type != "search_results" {
		t.Fatalf("unbekannte Littera-EAN muss in der Suche landen, got Type=%q", res.Type)
	}
}
