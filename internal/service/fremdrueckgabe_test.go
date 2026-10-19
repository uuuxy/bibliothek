package service

import (
	"context"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

type rueckgabeFehlerAttrappe struct {
	ausleihPflege   // eingebettet, damit die nicht nachgebildeten Methoden da sind
	rueckgabeFehler error
}

func (m *rueckgabeFehlerAttrappe) BucheRueckgabe(ctx context.Context, tx pgx.Tx, loanID string, bearbeiterID string, isFremdrueckgabe bool) error {
	return m.rueckgabeFehler
}

func TestVerarbeiteFremdrueckgabe_Schueler(t *testing.T) {
	svc, _, mockPool := newVorgangDienst(t, &repository.Leser{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	mockLoan := &rueckgabeFehlerAttrappe{}
	svc.ausleihAbfragen = mockLoan

	tx := beginneTx(t, mockPool)

	activeLoan := &repository.Ausleihe{
		ID:         "loan1",
		SchuelerID: strPtr("s1"),
	}
	exemplar := &repository.Exemplar{
		ID:        "copy1",
		BarcodeID: "B-C1",
		Titel:     "Testbuch",
	}

	resp := &AusleihErgebnis{}

	// bedieneVormerkung findet keine Vormerkung.
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit()

	res, err := svc.verarbeiteFremdrueckgabe(context.Background(), tx, exemplar, activeLoan, "staff1", resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Art != "rueckgabe" {
		t.Errorf("verarbeiteFremdrueckgabe(…): Art = %s, erwartet rueckgabe", res.Art)
	}
	if res.Vorbesitzer == nil || res.Vorbesitzer.ID != "s1" {
		t.Errorf("verarbeiteFremdrueckgabe(…).Vorbesitzer = %v, erwartet s1", res.Vorbesitzer)
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Auch das Buch eines Kollegen muss bei der Fremdrückgabe seinen Vorbesitzer nennen. Es ist
// derselbe Weg wie beim Schüler — Finde, nicht die Sicht `schueler`, sonst bliebe der
// Vorbesitzer namenlos.
func TestVerarbeiteFremdrueckgabe_Kollege(t *testing.T) {
	svc, _, mockPool := newVorgangDienst(t, &repository.Leser{
		ID: "t1", Vorname: "Hans", Nachname: "Lehrer", Art: "lehrkraft",
	})
	defer mockPool.Close()

	mockLoan := &rueckgabeFehlerAttrappe{}
	svc.ausleihAbfragen = mockLoan

	tx := beginneTx(t, mockPool)

	activeLoan := &repository.Ausleihe{
		ID:         "loan2",
		SchuelerID: strPtr("t1"),
	}
	exemplar := &repository.Exemplar{
		ID:        "copy2",
		BarcodeID: "B-C2",
		Titel:     "Lehrerbuch",
	}

	resp := &AusleihErgebnis{}

	// bedieneVormerkung findet keine Vormerkung.
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit()

	res, err := svc.verarbeiteFremdrueckgabe(context.Background(), tx, exemplar, activeLoan, "staff1", resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Art != "rueckgabe" {
		t.Errorf("verarbeiteFremdrueckgabe(…): Art = %s, erwartet rueckgabe", res.Art)
	}
	if res.Vorbesitzer == nil || res.Vorbesitzer.ID != "t1" {
		t.Errorf("der Kollege muss als Vorbesitzer in der Antwort stehen, bekam: %+v", res.Vorbesitzer)
	}
	if res.Vorbesitzer.Art != "lehrkraft" {
		t.Errorf("die Art des Vorbesitzers fehlt: %+v", res.Vorbesitzer)
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVerarbeiteFremdrueckgabe_RueckgabeFehler(t *testing.T) {
	svc, _, mockPool := newVorgangDienst(t, &repository.Leser{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	// Return error
	mockLoan := &rueckgabeFehlerAttrappe{rueckgabeFehler: pgx.ErrTxClosed}
	svc.ausleihAbfragen = mockLoan

	tx := beginneTx(t, mockPool)

	activeLoan := &repository.Ausleihe{
		ID:         "loan3",
		SchuelerID: strPtr("s1"),
	}
	exemplar := &repository.Exemplar{
		ID: "copy3",
	}

	resp := &AusleihErgebnis{}

	res, err := svc.verarbeiteFremdrueckgabe(context.Background(), tx, exemplar, activeLoan, "staff1", resp)

	if err == nil {
		t.Fatalf("verarbeiteFremdrueckgabe(…) = nil, erwartet einen Fehler")
	}
	if res != nil {
		t.Errorf("verarbeiteFremdrueckgabe(…) = %v, erwartet nil", res)
	}
}

func TestVerarbeiteFremdrueckgabe_CommitFehler(t *testing.T) {
	svc, _, mockPool := newVorgangDienst(t, &repository.Leser{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	mockLoan := &rueckgabeFehlerAttrappe{}
	svc.ausleihAbfragen = mockLoan

	tx := beginneTx(t, mockPool)

	activeLoan := &repository.Ausleihe{
		ID:         "loan4",
		SchuelerID: strPtr("s1"),
	}
	exemplar := &repository.Exemplar{
		ID:        "copy4",
		BarcodeID: "B-C4",
		Titel:     "Testbuch",
	}

	resp := &AusleihErgebnis{}

	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit().WillReturnError(pgx.ErrTxClosed)

	res, err := svc.verarbeiteFremdrueckgabe(context.Background(), tx, exemplar, activeLoan, "staff1", resp)

	if err == nil {
		t.Fatalf("verarbeiteFremdrueckgabe(…) = nil, erwartet den Fehler des Commits")
	}
	if res != nil {
		t.Errorf("verarbeiteFremdrueckgabe(…) = %v, erwartet nil", res)
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
