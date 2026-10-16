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
// die übrigen Methoden von db.Pool bleiben ungenutzt (eingebettetes nil-Interface).
type panikDatenbank struct{ db.Pool }

func (panikDatenbank) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("boom aus dem cover-sync (simuliert)")
}

// TestGleicheFehlendeCoverAbUeberlebtPanik: Eine Panik in der äußeren Sync-Goroutine darf
// den Prozess nicht mitreißen (safego.FangePanic) — und muss laeuft zurücksetzen, sonst
// überspringt jeder künftige Lauf für immer.
func TestGleicheFehlendeCoverAbUeberlebtPanik(t *testing.T) {
	svc := &CoverDienst{pool: panikDatenbank{}}
	svc.GleicheFehlendeCoverAb(t.Context()) // darf nicht paniken — sonst stirbt der Test mit

	if svc.laeuft.Load() {
		t.Error("laeuft blieb true nach dem Panik — künftige Cover-Läufe überspringen still für immer")
	}
}
