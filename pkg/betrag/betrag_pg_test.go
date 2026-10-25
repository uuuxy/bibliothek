package betrag_test

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/betrag"
)

// Der Treiber reicht eine Spalte NUMERIC als Text an Scan und übernimmt den Text aus Value genau
// in NUMERIC; NULL wird in einem *betrag.Cent zu nil. Ohne das läsen die Abfragen über Cent
// nichts oder rundeten über float64.
func TestCent_RundreiseDurchPostgres(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var gelesen betrag.Cent
	var leer *betrag.Cent
	var summe betrag.Cent
	if err := pool.QueryRow(ctx, `SELECT 12.50::numeric(10,2), NULL::numeric(10,2), (0.1::numeric + 0.2::numeric)`).
		Scan(&gelesen, &leer, &summe); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if gelesen != 1250 || leer != nil || summe != 30 {
		t.Errorf("gelesen %d, leer %v, Summe %d — erwartet 1250, nil, 30", gelesen, leer, summe)
	}

	var zurueck string
	if err := pool.QueryRow(ctx, `SELECT ($1::numeric(10,2))::text`, betrag.Cent(-1999)).Scan(&zurueck); err != nil {
		t.Fatalf("Value: %v", err)
	}
	if zurueck != "-19.99" {
		t.Errorf("Postgres las %q, erwartet „-19.99“", zurueck)
	}

	var gesetzt *betrag.Cent
	if err := pool.QueryRow(ctx, `SELECT 7::numeric(10,2)`).Scan(&gesetzt); err != nil || gesetzt == nil || *gesetzt != 700 {
		t.Errorf("Scan in *Cent: %v, %v — erwartet 700", gesetzt, err)
	}
}
