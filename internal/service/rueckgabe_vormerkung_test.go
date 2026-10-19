package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"

	"github.com/pashagolub/pgxmock/v5"
)

// Die Rückgabe darf kein Abholfach behaupten, das es nicht gibt.
//
// Scheitert das UPDATE der Vormerkung, setzt die Antwort weder HatVormerkung noch
// VorgemerktFuer, und der Aufrufer committet nicht: Sonst meldet der Arbeitsplatz „Reserviert
// für Max M. — ins Abholfach legen", während die Vormerkung in der Datenbank weiter auf
// 'wartend' steht. Das Buch läge im Fach und wäre frei ausleihbar, der Wartende sähe in seinem
// Profil nichts, und der Verfall griffe mangels bereitgestellt_bis nie. Auch ein
// Transportfehler des SELECTs ist ein Fehler, nicht „keine Vormerkung".
func TestBedieneVormerkung_FehlschlagLuegtNicht(t *testing.T) {
	ctx := context.Background()
	exemplar := &repository.Exemplar{ID: "11111111-1111-1111-1111-111111111111",
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
		mock.ExpectQuery(`SELECT schluessel, wert FROM system_einstellungen`).
			WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))
		mock.ExpectExec(`UPDATE vormerkungen SET status = 'abholbereit'`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errors.New("verbindung weg"))

		tx, err := mock.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin: %v", err)
		}
		resp := &AusleihErgebnis{}
		if err := (&AusleihDienst{}).bedieneVormerkung(ctx, tx, exemplar, resp, nil); err == nil {
			t.Error("UPDATE-Fehlschlag kam nicht als Fehler zurück — der Aufrufer committet und die Theke lügt")
		}
		if resp.HatVormerkung || resp.VorgemerktFuer != "" {
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
		resp := &AusleihErgebnis{}
		if err := (&AusleihDienst{}).bedieneVormerkung(ctx, tx, exemplar, resp, nil); err == nil {
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
		resp := &AusleihErgebnis{}
		if err := (&AusleihDienst{}).bedieneVormerkung(ctx, tx, exemplar, resp, nil); err != nil {
			t.Errorf("ErrNoRows ist der Normalfall (niemand wartet) und kein Fehler: %v", err)
		}
		if resp.HatVormerkung {
			t.Error("keine Vormerkung, aber HasVormerkung gesetzt")
		}
	})
}
