package testdaten

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// BucheAusleiheUeberDienst leiht ein Exemplar an einen Schüler aus, über denselben Dienst wie
// der Scan an der Theke, und liefert die Frist. So kommt die Frist aus den Einstellungen und
// nicht aus dem Test.
func BucheAusleiheUeberDienst(t *testing.T, pool *pgxpool.Pool, barcode, schuelerID, bearbeiterID string) time.Time {
	t.Helper()
	ctx := context.Background()

	katalog := repository.NewKatalogAbfragen(pool)
	dienst := service.NewAusleihDienst(pool, repository.NewLeserAbfragen(pool), katalog,
		repository.NewAusleihAbfragen(pool), repository.NewProtokollAbfragen(pool))

	exemplar, err := katalog.FindeExemplar(ctx, barcode)
	if err != nil {
		t.Fatalf("Exemplar %s laden: %v", barcode, err)
	}
	if _, err := dienst.BucheAmLeser(ctx, exemplar, &schuelerID, bearbeiterID, false); err != nil {
		t.Fatalf("Ausleihe von %s: %v", barcode, err)
	}

	var frist time.Time
	if err := pool.QueryRow(ctx,
		`SELECT rueckgabe_frist FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`,
		exemplar.ID).Scan(&frist); err != nil {
		t.Fatalf("Frist lesen: %v", err)
	}
	return frist
}
