package uebernahme

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestImSavepoint(t *testing.T) {
	t.Run("Erfolg", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		tx, err := mock.Begin(context.Background())
		if err != nil {
			t.Fatalf("mock.Begin: %v", err)
		}

		mock.ExpectBegin()
		mock.ExpectCommit()

		fn := func(sp pgx.Tx) error {
			return nil
		}

		erg, err := ImSavepoint(context.Background(), tx, "Titel 1", fn)
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if !erg.Uebernommen {
			t.Error("erwartet: Uebernommen = true")
		}
		if erg.Zurueckgerollt != nil {
			t.Errorf("erwartet: Zurueckgerollt = nil, erhalten: %v", erg.Zurueckgerollt)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unerfüllte Erwartungen: %s", err)
		}
	})

	t.Run("Zeilenfehler führt zum Rollback", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		tx, err := mock.Begin(context.Background())
		if err != nil {
			t.Fatalf("mock.Begin: %v", err)
		}

		mock.ExpectBegin()
		mock.ExpectRollback()

		zeilenFehler := &pgconn.PgError{Code: "23505"} // doppelte ISBN
		fn := func(sp pgx.Tx) error {
			return zeilenFehler
		}

		erg, err := ImSavepoint(context.Background(), tx, "Titel 2", fn)
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if erg.Uebernommen {
			t.Error("erwartet: Uebernommen = false")
		}
		if erg.Zurueckgerollt != zeilenFehler {
			t.Errorf("erwartet: Zurueckgerollt = %v, erhalten: %v", zeilenFehler, erg.Zurueckgerollt)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unerfüllte Erwartungen: %s", err)
		}
	})

	t.Run("Fataler Fehler ohne Rollback", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		tx, err := mock.Begin(context.Background())
		if err != nil {
			t.Fatalf("mock.Begin: %v", err)
		}

		mock.ExpectBegin()
		// kein Commit, kein Rollback auf dem Savepoint!

		fatalFehler := errors.New("datenbank weg")
		fn := func(sp pgx.Tx) error {
			return fatalFehler
		}

		erg, err := ImSavepoint(context.Background(), tx, "Titel 3", fn)
		if err == nil {
			t.Fatal("erwartet: Fehler, erhalten: nil")
		}
		if !strings.Contains(err.Error(), "abgebrochen bei Titel 3") {
			t.Errorf("Fehlermeldung falsch: %v", err)
		}
		if erg.Uebernommen {
			t.Error("erwartet: Uebernommen = false")
		}
		if erg.Zurueckgerollt != nil {
			t.Error("erwartet: Zurueckgerollt = nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unerfüllte Erwartungen: %s", err)
		}
	})

	t.Run("Fehler bei Begin", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		tx, err := mock.Begin(context.Background())
		if err != nil {
			t.Fatalf("mock.Begin: %v", err)
		}

		beginErr := errors.New("begin fehler")
		mock.ExpectBegin().WillReturnError(beginErr)

		fn := func(sp pgx.Tx) error {
			return nil
		}

		_, err = ImSavepoint(context.Background(), tx, "Titel 4", fn)
		if err == nil {
			t.Fatal("erwartet: Fehler, erhalten: nil")
		}
		if !strings.Contains(err.Error(), "konnte den Savepoint") {
			t.Errorf("Fehlermeldung falsch: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unerfüllte Erwartungen: %s", err)
		}
	})

	t.Run("Fehler bei Commit", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		tx, err := mock.Begin(context.Background())
		if err != nil {
			t.Fatalf("mock.Begin: %v", err)
		}

		mock.ExpectBegin()
		commitErr := errors.New("commit fehler")
		mock.ExpectCommit().WillReturnError(commitErr)

		fn := func(sp pgx.Tx) error {
			return nil
		}

		_, err = ImSavepoint(context.Background(), tx, "Titel 5", fn)
		if err == nil {
			t.Fatal("erwartet: Fehler, erhalten: nil")
		}
		if !strings.Contains(err.Error(), "nicht freigeben") {
			t.Errorf("Fehlermeldung falsch: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unerfüllte Erwartungen: %s", err)
		}
	})

	t.Run("Fehler bei Rollback", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		tx, err := mock.Begin(context.Background())
		if err != nil {
			t.Fatalf("mock.Begin: %v", err)
		}

		mock.ExpectBegin()
		rollbackErr := errors.New("rollback fehler")
		mock.ExpectRollback().WillReturnError(rollbackErr)

		zeilenFehler := &pgconn.PgError{Code: "23505"} // doppelte ISBN
		fn := func(sp pgx.Tx) error {
			return zeilenFehler
		}

		_, err = ImSavepoint(context.Background(), tx, "Titel 6", fn)
		if err == nil {
			t.Fatal("erwartet: Fehler, erhalten: nil")
		}
		if !strings.Contains(err.Error(), "nicht zurückrollen") {
			t.Errorf("Fehlermeldung falsch: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unerfüllte Erwartungen: %s", err)
		}
	})
}
