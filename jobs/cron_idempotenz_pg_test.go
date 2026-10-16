package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die stündliche Bereinigung an Postgres: Ein Schlüssel jenseits der 24 Stunden fällt weg,
// einer knapp darunter bleibt. Der Lauf meldet einen Fehler nur im Log; passte sein SQL nicht
// mehr zur Tabelle, wüchse sie, ohne dass es jemand sieht.
func TestIdempotenzBereinigung_LoeschtNurAbgelaufeneSchluessel(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}

	_, dsn := legeProbeDatenbankAn(t, adminDSN, "idempotenz")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	const (
		abgelaufen = "00000000-0000-4000-8000-000000000025"
		frisch     = "00000000-0000-4000-8000-000000000023"
	)
	legeAn := func(schluessel string, alterStunden int) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO idempotency_keys (idempotency_key, response_data, status_code, created_at)
			VALUES ($1, '{}'::jsonb, 200, now() - make_interval(hours => $2))`,
			schluessel, alterStunden); err != nil {
			t.Fatalf("Seed (%d Stunden): %v", alterStunden, err)
		}
	}
	legeAn(abgelaufen, 25)
	legeAn(frisch, 23)

	NewZeitplan(pool, repository.NewProtokollAbfragen(pool)).fuehreIdempotenzBereinigungAus()

	zeilen, err := pool.Query(ctx, `SELECT idempotency_key::text FROM idempotency_keys ORDER BY 1`)
	if err != nil {
		t.Fatalf("Schlüssel lesen: %v", err)
	}
	defer zeilen.Close()
	var uebrig []string
	for zeilen.Next() {
		var schluessel string
		if err := zeilen.Scan(&schluessel); err != nil {
			t.Fatalf("Schlüssel lesen: %v", err)
		}
		uebrig = append(uebrig, schluessel)
	}
	if err := zeilen.Err(); err != nil {
		t.Fatalf("Schlüssel lesen: %v", err)
	}
	if len(uebrig) != 1 || uebrig[0] != frisch {
		t.Errorf("nach der Bereinigung stehen %v, erwartet nur der 23 Stunden alte Schlüssel %s", uebrig, frisch)
	}
}
