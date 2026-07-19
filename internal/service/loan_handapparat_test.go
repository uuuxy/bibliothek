package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

// TestHandleLehrerHandapparat_SchranktEin sichert #2 ab: Der Handapparat-Schnellpfad muss
// dieselben Schranken achten wie der reguläre Checkout. Ein nicht ausleihbares, ausgesondertes
// oder für einen Schüler reserviertes Exemplar darf nicht kommentarlos auf die Lehrkraft
// gebucht werden — es muss eine Fehlermeldung geben.
func TestHandleLehrerHandapparat_SchranktEin(t *testing.T) {
	svc := &defaultLoanService{}

	t.Run("nicht ausleihbar wird abgelehnt", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		tx := beginTx(t, mock)
		copy := &repository.BookCopy{ID: "c1", TitelID: "t1", IstAusleihbar: false}
		if _, err := svc.handleLehrerHandapparat(context.Background(), tx, copy, "staff1", &LoanResult{}); !errors.Is(err, ErrInvalidState) {
			t.Errorf("erwartet ErrInvalidState, war %v", err)
		}
	})

	t.Run("ausgesondert wird abgelehnt", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		tx := beginTx(t, mock)
		copy := &repository.BookCopy{ID: "c2", TitelID: "t1", IstAusleihbar: true, IstAusgesondert: true}
		if _, err := svc.handleLehrerHandapparat(context.Background(), tx, copy, "staff1", &LoanResult{}); !errors.Is(err, ErrInvalidState) {
			t.Errorf("erwartet ErrInvalidState, war %v", err)
		}
	})

	t.Run("für einen Schüler reserviert wird abgelehnt", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		tx := beginTx(t, mock)
		// pruefeVormerkungKonflikt findet die abholbereite Reservierung → Konflikt für die
		// Lehrkraft (kein Schüler-Ausleiher).
		mock.ExpectQuery(vormerkungQuery).WithArgs("c3").
			WillReturnRows(pgxmock.NewRows([]string{"schueler_id", "vorname", "nachname"}).
				AddRow("s9", "Rex", "Reserviert"))
		copy := &repository.BookCopy{ID: "c3", TitelID: "t1", IstAusleihbar: true}
		if _, err := svc.handleLehrerHandapparat(context.Background(), tx, copy, "staff1", &LoanResult{}); !errors.Is(err, ErrConflict) {
			t.Errorf("erwartet ErrConflict, war %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Mock-Erwartungen: %v", err)
		}
	})
}
