package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Testdaten für die Tests an der Datenbank mit nur so vielen Pflichtfeldern wie nötig, damit
// die geprüfte Bedingung der einzige Grund zum Scheitern ist.

// legeTitelAn legt einen Titel mit den Pflichtfeldern an und liefert seine Kennung.
func legeTitelAn(t *testing.T, tx pgx.Tx) string {
	t.Helper()

	var id string
	err := tx.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel) VALUES ('Testtitel') RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("Fixture buecher_titel fehlgeschlagen: %v", err)
	}
	return id
}

// legeExemplarAn legt ein Exemplar zum Titel an und liefert seine Kennung.
func legeExemplarAn(t *testing.T, tx pgx.Tx, titelID string) string {
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

// legeBestellungAn legt den Kopf einer Bestellung an und liefert seine Kennung.
func legeBestellungAn(t *testing.T, tx pgx.Tx) string {
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
