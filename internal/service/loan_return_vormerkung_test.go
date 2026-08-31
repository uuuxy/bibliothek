package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"

	"github.com/pashagolub/pgxmock/v4"
)

// Die Rückgabe darf kein Abholfach behaupten, das es nicht gibt.
//
// Bis zum 31.08.2026 wurde ein Fehlschlag des vormerkungen-UPDATEs nur geloggt, die
// Antwortfelder (HasVormerkung, VormerkungUser) wurden trotzdem gesetzt und der
// Aufrufer committete: Der Arbeitsplatz meldete „Reserviert für Max M. — ins Abholfach
// legen", in der Datenbank stand die Vormerkung weiter auf 'wartend'. Das Buch lag im
// Fach UND war frei ausleihbar; der Wartende sah in seinem Profil nichts, und der
// Verfall-Cron griff mangels bereitgestellt_bis nie. Auch ein Transportfehler des
// SELECTs galt still als „keine Vormerkung".
func TestProcessReturnVormerkung_FehlschlagLuegtNicht(t *testing.T) {
	ctx := context.Background()
	copy := &repository.BookCopy{ID: "11111111-1111-1111-1111-111111111111",
		TitelID: "22222222-2222-2222-2222-222222222222", Titel: "Tschick"}

	t.Run("UPDATE scheitert → Fehler, keine Abholfach-Behauptung", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT v.id, s.vorname`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id", "vorname", "nachname", "klasse"}).
				AddRow("33333333-3333-3333-3333-333333333333", "Max", "Mustermann", "10A"))
		mock.ExpectExec(`UPDATE vormerkungen SET status = 'abholbereit'`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errors.New("verbindung weg"))

		tx, err := mock.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin: %v", err)
		}
		resp := &LoanResult{}
		if err := (&defaultLoanService{}).processReturnVormerkungTx(ctx, tx, copy, resp, nil); err == nil {
			t.Error("UPDATE-Fehlschlag kam nicht als Fehler zurück — der Aufrufer committet und die Theke lügt")
		}
		if resp.HasVormerkung || resp.VormerkungUser != "" {
			t.Errorf("Antwort behauptet ein Abholfach trotz gescheitertem UPDATE: %+v", resp)
		}
	})

	t.Run("SELECT-Transportfehler ist nicht „keine Vormerkung“", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT v.id, s.vorname`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errors.New("conn busy"))

		tx, err := mock.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin: %v", err)
		}
		resp := &LoanResult{}
		if err := (&defaultLoanService{}).processReturnVormerkungTx(ctx, tx, copy, resp, nil); err == nil {
			t.Error("SELECT-Transportfehler galt still als „keine Vormerkung“")
		}
	})

	t.Run("keine wartende Vormerkung → kein Fehler, keine Felder", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT v.id, s.vorname`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(pgx.ErrNoRows)

		tx, err := mock.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin: %v", err)
		}
		resp := &LoanResult{}
		if err := (&defaultLoanService{}).processReturnVormerkungTx(ctx, tx, copy, resp, nil); err != nil {
			t.Errorf("ErrNoRows ist der Normalfall (niemand wartet) und kein Fehler: %v", err)
		}
		if resp.HasVormerkung {
			t.Error("keine Vormerkung, aber HasVormerkung gesetzt")
		}
	})
}
