package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Die Rolle eines Kontos hat eine Quelle, benutzer.rolle. Käme die Tabelle benutzer_rollen zurück,
// etwa durch ein CREATE TABLE IF NOT EXISTS beim Start, stünde die Rolle wieder an zwei Orten.
func TestRollenVokabular(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	t.Run("Legacy-Tabelle benutzer_rollen existiert nicht mehr", func(t *testing.T) {
		var anzahl int
		err := pool.QueryRow(ctx,
			`SELECT count(*) FROM information_schema.tables WHERE table_name = 'benutzer_rollen'`).Scan(&anzahl)
		if err != nil {
			t.Fatalf("Abfrage fehlgeschlagen: %v", err)
		}
		if anzahl != 0 {
			t.Error("benutzer_rollen ist zurück — die Rolle hat wieder zwei Quellen (siehe Migration 044)")
		}
	})

	t.Run("Rolle helfer ist im ENUM vergebbar", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			// Ohne den Wert im ENUM bekäme niemand die Rolle, und die Theke für Helfer wäre
			// unerreichbar.
			erwarteErfolg(t, tx, "Benutzer mit Rolle helfer",
				`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
				 VALUES ('H', 'Helfer', 'h@example.org', 'helfer', true)`)
		})
	})
}
