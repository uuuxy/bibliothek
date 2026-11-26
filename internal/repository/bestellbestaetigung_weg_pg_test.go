package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Jeder der zwei Wege allein genügt: ein Bestätigungs-Link, auch wenn der Lieferant nicht mehr
// Hauptlieferant ist, oder der heutige Hauptlieferant ohne Link. Ohne beides, auch mit gelöschtem
// Lieferanten, gehört die Bestellung nicht dazu.
func TestIstBestellungImBestaetigungsweg_JederWegAllein(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	marke := time.Now().UnixNano()
	haupt := testdaten.LegeLieferantAn(t, pool, fmt.Sprintf("BW-Haupt-%d", marke), true)
	neben := testdaten.LegeLieferantAn(t, pool, fmt.Sprintf("BW-Neben-%d", marke), false)
	bestellung := func(lieferant string, mitLink bool) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, lieferant_id, bestaetigungs_token_hash)
			VALUES ('Weg', 'weg@example.invalid', NULLIF($1, '')::uuid,
			        CASE WHEN $2 THEN md5(gen_random_uuid()::text) END)
			RETURNING id`, lieferant, mitLink).Scan(&id); err != nil {
			t.Fatalf("Bestellung anlegen: %v", err)
		}
		return id
	}

	faelle := []struct {
		name string
		id   string
		soll bool
	}{
		{"Link, Lieferant nicht mehr Hauptlieferant", bestellung(neben, true), true},
		{"Hauptlieferant ohne Link", bestellung(haupt, false), true},
		{"weder Link noch Hauptlieferant", bestellung(neben, false), false},
		{"Lieferant gelöscht, ohne Link", bestellung("", false), false},
	}
	for _, f := range faelle {
		got, err := repository.IstBestellungImBestaetigungsweg(ctx, pool, f.id)
		if err != nil || got != f.soll {
			t.Errorf("%s: IstBestellungImBestaetigungsweg = %v, %v; erwartet %v", f.name, got, err, f.soll)
		}
	}
}
