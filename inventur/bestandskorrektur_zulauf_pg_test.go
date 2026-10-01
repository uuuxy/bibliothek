package inventur

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Die Buchmaske zeigt im Feld „Aktueller Bestand" die Exemplare im Bestand, ohne die
// bestellten (gesamt, repository.SQLExemplarImBestand). Die Bestandskorrektur dahinter muss
// mit derselben Grenze rechnen: Zählt sie die bestellten mit, liest sie die Zahl der Maske
// als Verringerung und sondert aus, ohne dass jemand das Feld angefasst hat.
func TestBestandskorrektur_RechnetOhneDenZulauf(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	marke := fmt.Sprintf("ZULKORR%d", time.Now().UnixNano())

	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ($1, $2, 'Buch') RETURNING id`,
		"Korrektur mit Zulauf "+marke, marke).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id = $1`,
			`DELETE FROM buecher_titel WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, titelID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})
	bestellt, unterwegs := "bestellt", "im_zulauf"
	for i, status := range []*string{nil, &bestellt, &unterwegs} {
		// chk_exemplar_bestellstatus_nur_im_zulauf: im Zulauf heißt nicht ausleihbar.
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
			VALUES ($1, $2, $3, $4)`, titelID, fmt.Sprintf("B-ZK-%d-%s", i, marke), status == nil, status); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
	}

	// {im Bestand, im Zulauf, ausgesondert}
	zaehle := func() [3]int {
		t.Helper()
		var z [3]int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FILTER (WHERE NOT ist_ausgesondert AND bestellstatus IS NULL),
			       count(*) FILTER (WHERE NOT ist_ausgesondert AND bestellstatus IS NOT NULL),
			       count(*) FILTER (WHERE ist_ausgesondert)
			FROM buecher_exemplare WHERE titel_id = $1`, titelID).Scan(&z[0], &z[1], &z[2]); err != nil {
			t.Fatalf("zählen: %v", err)
		}
		return z
	}
	if start := zaehle(); start != [3]int{1, 2, 0} {
		t.Fatalf("Ausgangslage %v, erwartet [1 2 0] — der Test misst nicht, was er soll", start)
	}

	buecher, err := repo.ListBooksByIDs(ctx, []string{titelID})
	if err != nil || len(buecher) != 1 {
		t.Fatalf("Einzel-Read: %v, %d Titel", err, len(buecher))
	}
	gezeigt := buecher[0].Stock
	if gezeigt != 1 {
		t.Fatalf("die Maske zeigt Bestand %d, erwartet 1", gezeigt)
	}

	schritte := []struct {
		name    string
		soll    int
		erwarte [3]int
	}{
		{"die gezeigte Zahl zurückgeschrieben", gezeigt, [3]int{1, 2, 0}},
		{"von 1 auf 3 erhöht", 3, [3]int{3, 2, 0}},
		{"von 3 auf 1 verringert", 1, [3]int{1, 2, 2}},
		{"auf 0 gesetzt", 0, [3]int{0, 2, 3}},
	}
	for _, s := range schritte {
		if err := repo.syncBookStock(ctx, pool, titelID, s.soll); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if ist := zaehle(); ist != s.erwarte {
			t.Errorf("%s: im Bestand %d, im Zulauf %d, ausgesondert %d — erwartet %d, %d und %d",
				s.name, ist[0], ist[1], ist[2], s.erwarte[0], s.erwarte[1], s.erwarte[2])
		}
	}
}
