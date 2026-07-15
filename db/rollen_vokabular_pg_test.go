package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestRollenVokabular sichert das Ergebnis von G5 ab: genau EINE Quelle für die Rolle
// eines Benutzers. Ohne diesen Test könnte die Legacy-Tabelle unbemerkt zurückkehren
// (z. B. durch ein wiederbelebtes CREATE TABLE IF NOT EXISTS im Seed).
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
			// Migration 042: ohne den ENUM-Wert war der fertig gebaute Kiosk-Modus
			// unerreichbar, weil keine Lehrkraft die Rolle zugewiesen bekommen konnte.
			erwarteErfolg(t, tx, "Benutzer mit Rolle helfer",
				`INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
				 VALUES ('h-1', 'H', 'Helfer', 'h@example.org', 'helfer', true)`)
		})
	})
}
