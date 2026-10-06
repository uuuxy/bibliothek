package service

import (
	"context"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestNewCoverService(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	service := NewCoverService(mock)
	if service == nil {
		t.Fatal("Erwartete ein initialisiertes CoverService, erhielt nil")
	}

	if service.db != mock {
		t.Errorf("Erwartete, dass die db-Instanz übereinstimmt")
	}
}

// panicQueryDB paniert bei Query — nur diese Methode wird vor dem Panik erreicht;
// die übrigen PgxPoolIface-Methoden bleiben ungenutzt (eingebettetes nil-Interface).
type panicQueryDB struct{ db.PgxPoolIface }

func (panicQueryDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("boom aus dem cover-sync (simuliert)")
}

// TestSyncMissingCoversUeberlebtPanik: Ein Panik in der ÄUSSEREN Sync-Goroutine darf
// den Prozess NICHT mitreissen (safego.Guard) — und muss coverSyncRunning zurücksetzen,
// sonst überspringt jeder künftige Lauf für immer.
func TestSyncMissingCoversUeberlebtPanik(t *testing.T) {
	coverSyncRunning.Store(false)
	t.Cleanup(func() { coverSyncRunning.Store(false) })

	svc := &CoverService{db: panicQueryDB{}}
	svc.SyncMissingCoversAsync() // darf nicht paniken — sonst stirbt der Test mit

	if coverSyncRunning.Load() {
		t.Error("coverSyncRunning blieb true nach dem Panik — künftige Cover-Läufe überspringen still für immer")
	}
}

func TestSyncMissingCoversAsync_NoMissingCovers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	coverSyncRunning.Store(false)
	t.Cleanup(func() { coverSyncRunning.Store(false) })

	svc := NewCoverService(mock)

	mock.ExpectQuery(`SELECT id, isbn FROM buecher_titel`).
		WillReturnRows(mock.NewRows([]string{"id", "isbn"}))

	svc.SyncMissingCoversAsync()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestSyncMissingCoversAsync_AlreadyRunning tests that if a sync is already in progress,
// another call will just return early without hitting the DB.
func TestSyncMissingCoversAsync_AlreadyRunning(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// Set it to true manually
	coverSyncRunning.Store(true)
	t.Cleanup(func() { coverSyncRunning.Store(false) })

	svc := NewCoverService(mock)
	// No expectations set, so if it hits the DB, the test will fail
	svc.SyncMissingCoversAsync()
}

// errorQueryDB simuliert einen Datenbankfehler beim initialen SELECT,
// um zu prüfen, ob der Fehler geloggt und sicher zurückgekehrt wird.
type errorQueryDB struct{ db.PgxPoolIface }

func (errorQueryDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, context.DeadlineExceeded
}

func TestSyncMissingCoversAsync_DBQueryError(t *testing.T) {
	coverSyncRunning.Store(false)
	t.Cleanup(func() { coverSyncRunning.Store(false) })

	svc := &CoverService{db: errorQueryDB{}}
	svc.SyncMissingCoversAsync() // should log and return

	if coverSyncRunning.Load() {
		t.Error("coverSyncRunning left as true after DB error")
	}
}

type scanErrorRows struct {
	pgx.Rows
	called bool
}

func (m *scanErrorRows) Next() bool {
	if m.called {
		return false
	}
	m.called = true
	return true
}

func (m *scanErrorRows) Scan(dest ...any) error {
	return context.DeadlineExceeded
}

func (m *scanErrorRows) Err() error {
	return context.DeadlineExceeded
}

func (m *scanErrorRows) Close() {}

type scanErrorQueryDB struct{ db.PgxPoolIface }

func (scanErrorQueryDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &scanErrorRows{}, nil
}

func TestSyncMissingCoversAsync_DBScanError(t *testing.T) {
	coverSyncRunning.Store(false)
	t.Cleanup(func() { coverSyncRunning.Store(false) })

	svc := &CoverService{db: scanErrorQueryDB{}}
	svc.SyncMissingCoversAsync() // should log error from Err() and return

	if coverSyncRunning.Load() {
		t.Error("coverSyncRunning left as true after DB scan error")
	}
}
