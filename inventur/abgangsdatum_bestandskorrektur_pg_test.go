package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Bestandskorrektur der Buchmaske ist die fünfte und sechste Tür zum Zustand
// „ausgesondert" (Migration 128, docs/OFFEN.md 9.3 d): Wird der Bestand eines Titels nach
// unten gesetzt, sondert sie erst freie und dann notfalls verliehene Exemplare aus. Sie
// liegt in einem anderen Paket als die vier Türen aus api/abgangsdatum_pg_test.go — und
// genau so entstehen Lücken in einem Nachweis: Der Test steht da, wo man ihn erwartet,
// und die beiden Wege daneben stehen woanders.
func TestAbgangsdatum_BestandskorrekturStempelt(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const titelID = "00000000-0000-0000-0000-0000000000a8"

	for _, sql := range []string{
		`DELETE FROM buecher_exemplare WHERE titel_id = '` + titelID + `'`,
		`DELETE FROM buecher_titel WHERE id = '` + titelID + `'`,
		`INSERT INTO buecher_titel (id, titel) VALUES ('` + titelID + `', 'Abgangsbuch Korrektur')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("%.50s: %v", sql, err)
		}
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	repo := NewBookRepository(pool)
	if err := repo.syncBookStock(ctx, pool, titelID, 3); err != nil {
		t.Fatalf("Bestand aufbauen: %v", err)
	}
	// Runter auf 1 — zwei Exemplare gehen ab.
	if err := repo.syncBookStock(ctx, pool, titelID, 1); err != nil {
		t.Fatalf("Bestand korrigieren: %v", err)
	}

	var ausgesondert, mitDatum int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE ist_ausgesondert),
		       count(*) FILTER (WHERE ist_ausgesondert AND ausgesondert_am IS NOT NULL)
		FROM buecher_exemplare WHERE titel_id = $1`, titelID).Scan(&ausgesondert, &mitDatum); err != nil {
		t.Fatalf("zählen: %v", err)
	}
	if ausgesondert != 2 {
		t.Fatalf("%d Exemplare ausgesondert, erwartet 2 — der Test misst nicht, was er soll", ausgesondert)
	}
	if mitDatum != ausgesondert {
		t.Errorf("%d von %d ausgesonderten Exemplaren ohne Abgangsdatum — diese Zeilen fehlen im Abgangsbuch",
			ausgesondert-mitDatum, ausgesondert)
	}
}
