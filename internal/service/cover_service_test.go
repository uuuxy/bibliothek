package service

import (
	"context"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
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
