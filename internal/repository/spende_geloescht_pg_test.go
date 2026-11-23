package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/mitteltopf"
	"bibliothek/internal/testdaten"
)

// Ein gelöschtes Exemplar behält seinen Topf in geloeschte_exemplare, damit Zugangs- und Abgangsbuch
// es weiter in seiner Liste führen. Für eine Spende verlangt das die Checks der Migration 182;
// ohne sie scheiterte das Löschen eines Titels mit einem gespendeten Exemplar.
func TestLoescheTitel_SpendeBehaeltIhrenTopf(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := testdaten.LegeTitelAn(t, pool, "Geschenk des Fördervereins")
	exemplar := testdaten.LegeExemplarAn(t, pool, titelID, "SP-GESCHENK", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET eigentum = $2, eigentum_quelle = 'hand' WHERE id = $1`,
		exemplar, mitteltopf.Spende); err != nil {
		t.Fatalf("Eigentum Spende setzen: %v", err)
	}

	if _, err := NewTitelAbfragen(pool).LoescheTitel(ctx, TitelLoeschauftrag{TitelIDs: []string{titelID}, Grund: "AUSSORTIERT", BearbeiterID: testdaten.LegeBearbeiterAn(t, pool)}); err != nil {
		t.Fatalf("Titel löschen: %v", err)
	}

	var topf, belegt string
	if err := pool.QueryRow(ctx, `SELECT topf, coalesce(topf_belegt, '') FROM geloeschte_exemplare WHERE exemplar_id = $1`,
		exemplar).Scan(&topf, &belegt); err != nil {
		t.Fatalf("Zeile des gelöschten Exemplars lesen: %v", err)
	}
	if topf != mitteltopf.Spende || belegt != mitteltopf.Spende {
		t.Errorf("Topf %q, belegt %q — erwartet beide %q", topf, belegt, mitteltopf.Spende)
	}
}
