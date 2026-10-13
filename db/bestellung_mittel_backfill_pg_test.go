package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Der Backfill aus Migration 109 ordnet Alt-Bestellungen ihren Topf NUR zu, wo er aus den
// Positionen eindeutig ist. Geprüft wird die echte Migrationsdatei gegen die Test-DB —
// deshalb ist sie idempotent geschrieben (ADD COLUMN IF NOT EXISTS): Die Spalten sind
// aus schema.sql schon da, der UPDATE-Teil ist das, was hier zählt.
//
// Vier Fälle, die alle Antworten des CASE abdecken: nur Lernmittel → land, kein Lernmittel
// → schultraeger, gemischt → NULL, gelöschter Titel → NULL (raten wäre schlimmer als
// „ohne Zuordnung").

func TestBackfillMittelOrdnetNurEindeutigeAltBestellungenZu(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE bestellungen_verlauf CASCADE`); err != nil {
		t.Fatal(err)
	}

	lernmittel := testdaten.TitelMitMeldebestand(t, pool, "LMF-Mathe 7", 0)
	lernmittel2 := testdaten.TitelMitMeldebestand(t, pool, "LMF-Deutsch 7", 0)
	buecherei := testdaten.TitelMitMeldebestand(t, pool, "Tribute von Panem", 0)

	nurLernmittel := testdaten.AltBestellung(t, pool, "nur-lernmittel", []string{lernmittel, lernmittel2})
	nurBuecherei := testdaten.AltBestellung(t, pool, "nur-buecherei", []string{buecherei})
	gemischt := testdaten.AltBestellung(t, pool, "gemischt", []string{lernmittel, buecherei})
	titelWeg := testdaten.AltBestellung(t, pool, "titel-geloescht", []string{lernmittel, ""})
	ohnePositionen := testdaten.AltBestellung(t, pool, "leer", nil)

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "109_bestellung_mittel.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("Migration 109 ausführen: %v", err)
	}

	erwartet := map[string]*string{
		nurLernmittel:  testdaten.Ptr("land"),
		nurBuecherei:   testdaten.Ptr("schultraeger"),
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
