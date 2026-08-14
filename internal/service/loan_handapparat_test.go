package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
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

// TestHandleLehrerHandapparat_Erfolg_Und_Konflikt prüft den erfolgreichen Checkout sowie
// den Fehlerfall, falls ein zweiter Scan auf dasselbe Exemplar gleichzeitig erfolgt.
func TestHandleLehrerHandapparat_Erfolg_Und_Konflikt(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	loanRepo := repository.NewLoanRepository(mock)
	auditRepo := repository.NewAuditRepository(mock)

	svc := &defaultLoanService{
		loanRepo:  loanRepo,
		auditRepo: auditRepo,
	}

	t.Run("erfolgreiche Handapparat-Ausleihe", func(t *testing.T) {
		tx := beginTx(t, mock)
		copy := &repository.BookCopy{ID: "c1", TitelID: "t1", IstAusleihbar: true}
		resp := &LoanResult{}

		mock.ExpectQuery(vormerkungQuery).WithArgs("c1").WillReturnError(pgx.ErrNoRows)

		c1 := "c1"
		staff1 := "staff1"
		var ptrStaff1 *string = &staff1
		var ptrNil *string

		mock.ExpectQuery(`INSERT INTO ausleihen`).WithArgs(
			c1, staff1, pgxmock.AnyArg(), staff1, true,
		).WillReturnRows(pgxmock.NewRows([]string{
			"id", "exemplar_id", "schueler_id", "ausleiher_benutzer_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat",
		}).AddRow(
			"l1", &c1, nil, &staff1, time.Now(), time.Now().AddDate(1, 0, 0), nil, &staff1, nil, false, true,
		))

		mock.ExpectCommit()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO audit_log`).WithArgs(
			"ausleihen", "CHECKOUT", "c1", ptrStaff1, "USER", ptrNil, pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		res, err := svc.handleLehrerHandapparat(context.Background(), tx, copy, "staff1", resp)

		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if res == nil {
			t.Fatal("erwartet LoanResult, war nil")
		}
		if res.Type != "ausleihe" {
			t.Errorf("erwartet Type 'ausleihe', war %s", res.Type)
		}
		if res.DueDate == nil {
			t.Error("erwartet DueDate, war nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Mock-Erwartungen: %v", err)
		}
	})

	t.Run("Ausleihe-Konflikt", func(t *testing.T) {
		tx := beginTx(t, mock)
		copy := &repository.BookCopy{ID: "c2", TitelID: "t1", IstAusleihbar: true}
		resp := &LoanResult{}

		mock.ExpectQuery(vormerkungQuery).WithArgs("c2").WillReturnError(pgx.ErrNoRows)

		c2 := "c2"
		staff2 := "staff2"

		mock.ExpectQuery(`INSERT INTO ausleihen`).WithArgs(
			c2, staff2, pgxmock.AnyArg(), staff2, true,
		).WillReturnError(repository.ErrAusleiheKonflikt)

		res, err := svc.handleLehrerHandapparat(context.Background(), tx, copy, "staff2", resp)

		if !errors.Is(err, ErrConflict) {
			t.Errorf("erwartet ErrConflict, war %v", err)
		}
		if res != nil {
			t.Errorf("erwartet nil, war %v", res)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Mock-Erwartungen: %v", err)
		}
	})
}
