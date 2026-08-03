package api

import (
	"context"
	"testing"
)

// TestNurEinStandardLieferant sichert die Invariante ab, auf der die Vorauswahl beruht:
// Es darf immer nur EINEN Standardlieferanten geben.
//
// Zwei wären ein stiller Fehler — das Bestellformular zeigte einen davon, und welchen,
// entschiede die Sortierung. Der Betreiber sähe einen Haken an zwei Zeilen und könnte sich
// nicht erklären, warum trotzdem der falsche Händler vorausgewählt ist.
//
// Geprüft wird gegen echtes Postgres, weil der Schutz ein Teil-Index ist
// (idx_lieferanten_ein_standard, Migration 058). Mit pgxmock liesse sich das nicht
// nachbilden: Ein Unique-Index auf einer WHERE-Bedingung ist Datenbankverhalten, keine
// Programmlogik.
func TestNurEinStandardLieferant(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	anlegen := func(name string, standard bool) string {
		t.Helper()
		var id string
		// $1 ausdruecklich als text: Sonst kann Postgres den Typ nicht ableiten, weil derselbe
		// Parameter einmal als Spaltenwert und einmal in einer Verkettung steht (SQLSTATE 42P08).
		err := pool.QueryRow(ctx, `
			INSERT INTO lieferanten (name, email, kundennummer, ist_standard)
			VALUES ($1::text, $1::text || '@test.invalid', $1::text, $2)
			RETURNING id
		`, name, standard).Scan(&id)
		if err != nil {
			t.Fatalf("Lieferant %s anlegen: %v", name, err)
		}
		return id
	}

	ersterID := anlegen("Erster", true)
	zweiterID := anlegen("Zweiter", false)

	// 1. Ein ZWEITER Standard darf gar nicht erst entstehen.
	if _, err := pool.Exec(ctx, `UPDATE lieferanten SET ist_standard = true WHERE id = $1`, zweiterID); err == nil {
		t.Fatal("zwei Standardlieferanten muessen an idx_lieferanten_ein_standard scheitern")
	}

	// 2. Der vorgesehene Weg raeumt erst und setzt dann — in dieser Reihenfolge.
	if err := setzeStandardLieferant(ctx, pool, zweiterID); err != nil {
		t.Fatalf("Wechsel des Standardlieferanten: %v", err)
	}

	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM lieferanten WHERE ist_standard`).Scan(&anzahl); err != nil {
		t.Fatalf("zaehlen: %v", err)
	}
	if anzahl != 1 {
		t.Fatalf("%d Standardlieferanten nach dem Wechsel, erwartet genau 1", anzahl)
	}

	var standardID string
	if err := pool.QueryRow(ctx, `SELECT id FROM lieferanten WHERE ist_standard`).Scan(&standardID); err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if standardID != zweiterID {
		t.Fatalf("Standard ist %s, erwartet der zuletzt gesetzte (%s)", standardID, zweiterID)
	}

	// 3. Und der Wechsel ist wiederholbar — genau hier faellt eine Umsetzung auf, die
	//    erst setzt und dann raeumt: Sie kommt beim ZWEITEN Wechsel an den Index.
	if err := setzeStandardLieferant(ctx, pool, ersterID); err != nil {
		t.Fatalf("zweiter Wechsel: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM lieferanten WHERE ist_standard`).Scan(&anzahl); err != nil {
		t.Fatalf("zaehlen: %v", err)
	}
	if anzahl != 1 {
		t.Fatalf("%d Standardlieferanten nach dem zweiten Wechsel, erwartet genau 1", anzahl)
	}
}
