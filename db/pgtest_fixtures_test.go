package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Minimal-Fixtures für die DB-Integrationstests: nur so viele Pflichtfelder wie
// nötig, damit der jeweils geprüfte Constraint der einzige Grund zum Scheitern ist.

// neuerTitel legt einen Minimal-Titel an und liefert dessen ID.
func neuerTitel(t *testing.T, tx pgx.Tx) string {
	t.Helper()

	var id string
	err := tx.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel) VALUES ('Testtitel') RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("Fixture buecher_titel fehlgeschlagen: %v", err)
	}
	return id
}

// neuesExemplar legt ein Minimal-Exemplar zum Titel an und liefert dessen ID.
func neuesExemplar(t *testing.T, tx pgx.Tx, titelID string) string {
	t.Helper()

	var id string
	err := tx.QueryRow(context.Background(),
		`INSERT INTO buecher_exemplare (titel_id, barcode_id)
		 VALUES ($1, 'TEST-' || gen_random_uuid()::text) RETURNING id`, titelID).Scan(&id)
	if err != nil {
		t.Fatalf("Fixture buecher_exemplare fehlgeschlagen: %v", err)
	}
	return id
}

// neueBestellung legt einen Bestellkopf an und liefert dessen ID.
func neueBestellung(t *testing.T, tx pgx.Tx) string {
	t.Helper()

	var id string
	err := tx.QueryRow(context.Background(),
		`INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email)
		 VALUES ('Testverlag', 'test@example.org') RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("Fixture bestellungen_verlauf fehlgeschlagen: %v", err)
	}
	return id
}
