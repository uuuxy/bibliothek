package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Migration 159: Der Standort am Titel (erweiterte_eigenschaften.standort) geht an die
// Exemplare des Titels ohne eigenen Standort, danach fällt der Schlüssel weg. Geprüft wird die
// echte Migrationsdatei an Zeilen jeder Form, in einer Transaktion, die zurückgerollt wird.
func TestStandortAmTitelEntfaellt_Migration159(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	titel := func(name, eigenschaften string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, erweiterte_eigenschaften)
			VALUES ($1, $2::jsonb) RETURNING id`, name, eigenschaften).Scan(&id); err != nil {
			t.Fatalf("Titel %q: %v", name, err)
		}
		return id
	}
	exemplar := func(titelID, barcode string, standort *string) {
		t.Helper()
		if _, err := tx.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, standort)
			VALUES ($1, $2, $3)`, titelID, barcode, standort); err != nil {
			t.Fatalf("Exemplar %q: %v", barcode, err)
		}
	}
	standort := func(barcode string) string {
		t.Helper()
		var wert string
		if err := tx.QueryRow(ctx, `SELECT coalesce(standort, '(NULL)') FROM buecher_exemplare
			WHERE barcode_id = $1`, barcode).Scan(&wert); err != nil {
			t.Fatalf("Standort von %q: %v", barcode, err)
		}
		return wert
	}

	eigen := "Lehrerschrank"
	krimi := titel("M159 Krimi", `{"standort": "  Krimi-Ecke ", "regal": "R1"}`)
	exemplar(krimi, "M159-OHNE", nil)
	exemplar(krimi, "M159-EIGEN", &eigen)
	leer := titel("M159 Leer", `{"standort": "   "}`)
	exemplar(leer, "M159-LEER", nil)
	lang := titel("M159 Lang", `{"standort": "`+strings.Repeat("ä", 254)+` Regal"}`)
	exemplar(lang, "M159-LANG", nil)
	titel("M159 Ohne Exemplar", `{"standort": "Keller", "regal": "R2"}`)
	ohneSchluessel := titel("M159 Ohne Schlüssel", `{"regal": "R3"}`)
	exemplar(ohneSchluessel, "M159-UNBERUEHRT", nil)

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "159_standort_am_titel_entfaellt.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	// Zweimal: Die Migration muss beim erneuten Lauf still bleiben.
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	soll := map[string]string{
		"M159-OHNE":       "Krimi-Ecke",             // bekommt den Wert des Titels, ohne Leerraum am Rand
		"M159-EIGEN":      "Lehrerschrank",          // der eigene Standort bleibt
		"M159-LEER":       "(NULL)",                 // ein leerer Wert am Titel ist kein Standort
		"M159-LANG":       strings.Repeat("ä", 254), // gekürzt auf 255 Zeichen, das Leerzeichen am Ende fällt
		"M159-UNBERUEHRT": "(NULL)",
	}
	for barcode, erwartet := range soll {
		if got := standort(barcode); got != erwartet {
			t.Errorf("%s: Standort %q, erwartet %q", barcode, got, erwartet)
		}
	}

	var mitSchluessel, regale int
	if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE erweiterte_eigenschaften ? 'standort'),
			count(*) FILTER (WHERE erweiterte_eigenschaften ? 'regal')
		FROM buecher_titel WHERE titel LIKE 'M159 %'`).Scan(&mitSchluessel, &regale); err != nil {
		t.Fatalf("Titel zählen: %v", err)
	}
	if mitSchluessel != 0 {
		t.Errorf("%d Titel tragen den Schlüssel standort noch", mitSchluessel)
	}
	if regale != 3 {
		t.Errorf("%d Titel tragen den Schlüssel regal, erwartet 3 — die Migration hat andere Eigenschaften angefasst", regale)
	}
}
