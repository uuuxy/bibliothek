package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Backfill aus Migration 109 ordnet Alt-Bestellungen ihren Topf NUR zu, wo er aus den
// Positionen eindeutig ist. Geprüft wird die echte Migrationsdatei gegen die Test-DB —
// deshalb ist sie idempotent geschrieben (ADD COLUMN IF NOT EXISTS): Die Spalten sind
// aus schema.sql schon da, der UPDATE-Teil ist das, was hier zählt.
//
// Vier Fälle, die alle Antworten des CASE abdecken: nur Lernmittel → land, kein Lernmittel
// → schultraeger, gemischt → NULL, gelöschter Titel → NULL (raten wäre schlimmer als
// „ohne Zuordnung").

func altBestellung(t *testing.T, pool *pgxpool.Pool, name string, titelIDs []string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email) VALUES ($1, 'alt@example.invalid')
		RETURNING id`, name).Scan(&id); err != nil {
		t.Fatalf("Alt-Bestellung anlegen: %v", err)
	}
	for _, titelID := range titelIDs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO bestellungen_positionen (bestellung_id, titel_id, titel_name, menge)
			VALUES ($1, NULLIF($2, '')::uuid, 'Titel', 1)`, id, titelID); err != nil {
			t.Fatalf("Position anlegen: %v", err)
		}
	}
	return id
}

func TestBackfillMittelOrdnetNurEindeutigeAltBestellungenZu(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE bestellungen_verlauf CASCADE`); err != nil {
		t.Fatal(err)
	}

	lernmittel := titelMitMeldebestand(t, pool, "LMF-Mathe 7", 0)
	lernmittel2 := titelMitMeldebestand(t, pool, "LMF-Deutsch 7", 0)
	buecherei := titelMitMeldebestand(t, pool, "Tribute von Panem", 0)

	nurLernmittel := altBestellung(t, pool, "nur-lernmittel", []string{lernmittel, lernmittel2})
	nurBuecherei := altBestellung(t, pool, "nur-buecherei", []string{buecherei})
	gemischt := altBestellung(t, pool, "gemischt", []string{lernmittel, buecherei})
	titelWeg := altBestellung(t, pool, "titel-geloescht", []string{lernmittel, ""})
	ohnePositionen := altBestellung(t, pool, "leer", nil)

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "109_bestellung_mittel.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("Migration 109 ausführen: %v", err)
	}

	erwartet := map[string]*string{
		nurLernmittel:  ptr("land"),
		nurBuecherei:   ptr("schultraeger"),
		gemischt:       nil,
		titelWeg:       nil,
		ohnePositionen: nil,
	}
	for id, want := range erwartet {
		var got *string
		if err := pool.QueryRow(ctx, `SELECT mittel FROM bestellungen_verlauf WHERE id = $1`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		switch {
		case want == nil && got != nil:
			t.Errorf("Bestellung %s: Topf %q geraten, want NULL", id, *got)
		case want != nil && (got == nil || *got != *want):
			t.Errorf("Bestellung %s: Topf %v, want %q", id, got, *want)
		}
	}

	// Ein zweiter Lauf ändert nichts — die Migration darf auf einer Anlage, die schon
	// zugeordnet ist, nichts umschreiben (WHERE mittel IS NULL).
	if _, err := pool.Exec(ctx, `UPDATE bestellungen_verlauf SET mittel = 'schultraeger' WHERE id = $1`, nurLernmittel); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("zweiter Lauf: %v", err)
	}
	var danach string
	if err := pool.QueryRow(ctx, `SELECT mittel FROM bestellungen_verlauf WHERE id = $1`, nurLernmittel).Scan(&danach); err != nil {
		t.Fatal(err)
	}
	if danach != "schultraeger" {
		t.Errorf("zweiter Lauf hat eine bereits zugeordnete Bestellung umgeschrieben: %q", danach)
	}
}

func ptr(s string) *string { return &s }
