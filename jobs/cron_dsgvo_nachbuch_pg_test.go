package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Was die Frist der quittierten Nachbuch-Meldungen löscht, steht im Protokoll (Rechenschaftspflicht);
// ein Lauf ohne Fund schreibt nichts, sonst käme jede Nacht eine Zeile dazu.
func TestNachbuchMeldungenBefristung_ProtokolliertNurMitFund(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "nachbuchfrist")
	befuelleQuelle(t, dsn)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	z := NewZeitplan(pool, repository.NewProtokollAbfragen(pool))
	eintraege := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM audit_log WHERE tabelle = 'nachbuch_meldungen' AND aktion = 'DELETE'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	z.fuehreNachbuchMeldungenBefristungAus(ctx)
	if n := eintraege(); n != 0 {
		t.Errorf("Lauf ohne Fund: %d Einträge im Protokoll, erwartet 0", n)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO nachbuch_meldungen (idempotency_key, barcode, ergebnis, grund, gescannt_am, quittiert_am)
		VALUES (gen_random_uuid(), 'B-ALT', $1, 'alt', now() - interval '2 years', now() - interval '2 years')`,
		repository.NachbuchUmgebucht); err != nil {
		t.Fatalf("Meldung anlegen: %v", err)
	}
	z.fuehreNachbuchMeldungenBefristungAus(ctx)
	var uebrig int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM nachbuch_meldungen WHERE barcode = 'B-ALT'`).Scan(&uebrig); err != nil {
		t.Fatal(err)
	}
	if uebrig != 0 {
		t.Error("die abgelaufene Meldung steht noch da")
	}
	if n := eintraege(); n != 1 {
		t.Errorf("Lauf mit Fund: %d Einträge im Protokoll, erwartet 1", n)
	}
}
