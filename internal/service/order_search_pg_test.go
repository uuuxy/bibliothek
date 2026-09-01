package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// TestSearchLocalOrders_LiefertSignatur belegt, dass die lokale Bestellsuche die
// Regalsignatur eines bereits katalogisierten Titels mitliefert — Voraussetzung
// dafür, dass der Bestellkorb sie anzeigen und "die vorhandene Systematik
// übernehmen" statt sie zu verschweigen. Gated auf TEST_DATABASE_URL, siehe
// [[pg-integration-test-workflow]].
//
// Seit 01.09.2026 über internal/pgtest: Der Test hielt sich vorher einen EIGENEN
// Advisory-Lock auf demselben Schlüssel (0x42DB0001) samt eigenem Schema-Reset —
// sobald ein zweiter Test im selben Binary den Lock über pgtest hielt (der ihn
// bis Prozessende hält), wartete dieser Test für immer auf sein eigenes Binary.
// Genau so hat er ab dem ersten weiteren PG-Test in diesem Paket die komplette
// Suite in den 10-Minuten-Timeout gezogen; der eigene DROP SCHEMA mitten im
// Binary hätte zudem jedem nachfolgenden Test die Tabellen weggezogen.
func TestSearchLocalOrders_LiefertSignatur(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, autor, isbn, signatur) VALUES ($1, $2, $3, $4) RETURNING id`,
		"Effi Briest", "Fontane, Theodor", "9783150001", "Pg").Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})

	results := searchLocalOrders(ctx, pool, "Effi Briest")
	if len(results) != 1 {
		t.Fatalf("Treffer = %d, want 1", len(results))
	}
	if results[0].Signatur != "Pg" {
		t.Errorf("Signatur = %q, want %q", results[0].Signatur, "Pg")
	}
	if results[0].Source != "local" {
		t.Errorf("Source = %q, want %q", results[0].Source, "local")
	}
}
