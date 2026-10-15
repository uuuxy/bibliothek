package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Der Nachtrag ordnet alten Bestellungen ihren Topf nur zu, wo er aus den Positionen eindeutig
// ist. Geprüft wird die echte Datei der Migration an der Testdatenbank; deshalb ist sie so
// geschrieben, dass sie mehrfach laufen kann (ADD COLUMN IF NOT EXISTS): Die Spalten sind aus
// schema.sql schon da, hier zählt der UPDATE.
//
// Vier Fälle decken alle Antworten des CASE ab: nur Lernmittel → land, kein Lernmittel →
// schultraeger, gemischt → NULL, gelöschter Titel → NULL (Raten wäre schlimmer als „ohne
// Zuordnung").
func TestNachtragMittelOrdnetNurEindeutigeAltBestellungenZu(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE bestellungen_verlauf CASCADE`); err != nil {
		t.Fatal(err)
	}

	lernmittel := testdaten.LegeTitelAn(t, pool, "LMF-Mathe 7", 0)
	lernmittel2 := testdaten.LegeTitelAn(t, pool, "LMF-Deutsch 7", 0)
	buecherei := testdaten.LegeTitelAn(t, pool, "Tribute von Panem", 0)

	nurLernmittel := testdaten.LegeBestellungOhneTopfAn(t, pool, "nur-lernmittel", []string{lernmittel, lernmittel2})
	nurBuecherei := testdaten.LegeBestellungOhneTopfAn(t, pool, "nur-buecherei", []string{buecherei})
	gemischt := testdaten.LegeBestellungOhneTopfAn(t, pool, "gemischt", []string{lernmittel, buecherei})
	titelWeg := testdaten.LegeBestellungOhneTopfAn(t, pool, "titel-geloescht", []string{lernmittel, ""})
	ohnePositionen := testdaten.LegeBestellungOhneTopfAn(t, pool, "leer", nil)

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "109_bestellung_mittel.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("Migration 109 ausführen: %v", err)
	}

	erwartet := map[string]*string{
		nurLernmittel:  testdaten.ZeigerAuf("land"),
		nurBuecherei:   testdaten.ZeigerAuf("schultraeger"),
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
			ist := "NULL"
			if got != nil {
				ist = *got
			}
			t.Errorf("Bestellung %s: Topf %s, want %q", id, ist, *want)
		}
	}

	// Ein zweiter Lauf ändert nichts: Auf einer Anlage, die schon zugeordnet ist, schreibt die
	// Migration nichts um (WHERE mittel IS NULL).
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
