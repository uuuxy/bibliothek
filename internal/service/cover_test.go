package service

import (
	"context"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestNewCoverDienst(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	service := NewCoverDienst(mock)
	if service == nil {
		t.Fatal("Erwartete ein initialisiertes CoverDienst, erhielt nil")
	}

	if service.pool != mock {
		t.Errorf("Erwartete, dass die db-Instanz übereinstimmt")
	}
}

// panikDatenbank paniert bei Query — nur diese Methode wird vor dem Panik erreicht;
// die übrigen PgxPoolIface-Methoden bleiben ungenutzt (eingebettetes nil-Interface).
type panikDatenbank struct{ db.PgxPoolIface }

func (panikDatenbank) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("boom aus dem cover-sync (simuliert)")
}

// TestGleicheFehlendeCoverAbUeberlebtPanik: Eine Panik in der äußeren Sync-Goroutine darf
// den Prozess nicht mitreißen (safego.FangePanic) — und muss coverAbgleichLaeuft zurücksetzen,
// sonst überspringt jeder künftige Lauf für immer.
func TestGleicheFehlendeCoverAbUeberlebtPanik(t *testing.T) {
	coverAbgleichLaeuft.Store(false)
	t.Cleanup(func() { coverAbgleichLaeuft.Store(false) })

	svc := &CoverDienst{pool: panikDatenbank{}}
	svc.GleicheFehlendeCoverAb() // darf nicht paniken — sonst stirbt der Test mit

	if coverAbgleichLaeuft.Load() {
		t.Error("coverAbgleichLaeuft blieb true nach dem Panik — künftige Cover-Läufe überspringen still für immer")
	}
}
