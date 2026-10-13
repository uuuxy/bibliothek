package testdaten

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// AusleiheUeberDenDienst leiht ein Exemplar an einen Schüler aus — über denselben Dienst,
// den auch der Scanner benutzt, damit die Frist wirklich aus den Einstellungen kommt und
// nicht aus dem Test.
func AusleiheUeberDenDienst(t *testing.T, pool *pgxpool.Pool, barcode, schuelerID, bearbeiterID string) time.Time {
	t.Helper()
	ctx := context.Background()

	bookRepo := repository.NewKatalogAbfragen(pool)
	loanSvc := service.NewLoanService(pool, repository.NewLeserAbfragen(pool), bookRepo,
		repository.NewAusleihAbfragen(pool), repository.NewProtokollAbfragen(pool))

	exemplar, err := bookRepo.FindeExemplar(ctx, barcode)
	if err != nil {
		t.Fatalf("Exemplar %s laden: %v", barcode, err)
	}
	if _, err := loanSvc.HandleUnifiedCheckout(ctx, exemplar, &schuelerID, bearbeiterID, false); err != nil {
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
