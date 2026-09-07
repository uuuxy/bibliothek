package api

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"
)

// Migration 107 räumt die drei erfundenen Lieferanten des alten Programmstarts ab; die
// Selbstprüfung meldet Reste. Beides am echten Postgres: Ein Eintrag mit echten Daten
// bleibt, die Bestellung an den gelöschten Eintrag bleibt als Beleg erhalten, und ein
// umbenannter Rest mit der erfundenen Adresse wird gefunden.
func TestSeedLieferantenAbgeschafft107(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE bestellungen_verlauf, lieferanten CASCADE`); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	var klettID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO lieferanten (name, email, kundennummer) VALUES
			('Cornelsen',  'service@cornelsen.de', 'C-88123'),
			('Westermann', 'order@westermann.de',  'W-77441'),
			('Buchhandlung Echt', 'bestellung@buchhandlung-echt.example', '4711'),
			('Klett Verlag', 'bestellung@klett.de', 'K-99281')
		RETURNING id`).Scan(&klettID); err != nil {
		t.Fatalf("Lieferanten anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO bestellungen_verlauf (lieferant_id, lieferant_name, lieferant_email, kundennummer)
		VALUES ($1, 'Klett Verlag', 'bestellung@klett.de', 'K-99281')`, klettID); err != nil {
		t.Fatalf("Bestellung anlegen: %v", err)
	}

	sql, err := os.ReadFile("../migrations/107_seed_lieferanten_abgeschafft.sql")
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("Migration ausführen: %v", err)
	}

	var uebrig []string
	rows, err := pool.Query(ctx, `SELECT name FROM lieferanten ORDER BY name`)
	if err != nil {
		t.Fatalf("Lieferanten lesen: %v", err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		uebrig = append(uebrig, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("Lieferanten lesen (rows): %v", err)
	}
	if len(uebrig) != 1 || uebrig[0] != "Buchhandlung Echt" {
		t.Fatalf("nach Migration 107 übrig: %v, want nur Buchhandlung Echt", uebrig)
	}

	// Der Beleg bleibt: Name und Adresse stehen in der Bestellung selbst.
	var belegName string
	var belegLieferant *string
	if err := pool.QueryRow(ctx,
		`SELECT lieferant_name, lieferant_id::text FROM bestellungen_verlauf`).Scan(&belegName, &belegLieferant); err != nil {
		t.Fatalf("Bestellung nach Migration: %v", err)
	}
	if belegName != "Klett Verlag" || belegLieferant != nil {
		t.Errorf("Beleg = %q / lieferant_id=%v, want Klett Verlag / NULL", belegName, belegLieferant)
	}

	// Die Selbstprüfung: nach der Migration sauber …
	repo := repository.NewBetriebszustandRepository(pool)
	namen, err := repo.ErfundeneLieferanten(ctx)
	if err != nil {
		t.Fatalf("ErfundeneLieferanten: %v", err)
	}
	if len(namen) != 0 {
		t.Errorf("nach Migration noch erfunden: %v", namen)
	}

	// … und ein umbenannter Rest mit der erfundenen Adresse wird trotzdem gefunden — den
	// lässt die Migration bewusst stehen (kein exaktes Tripel), die Prüfung nicht.
	if _, err := pool.Exec(ctx, `
		INSERT INTO lieferanten (name, email, kundennummer)
		VALUES ('Unser Schulbuchhändler', 'Bestellung@Klett.de', 'K-1')`); err != nil {
		t.Fatalf("Rest anlegen: %v", err)
	}
	namen, err = repo.ErfundeneLieferanten(ctx)
	if err != nil {
		t.Fatalf("ErfundeneLieferanten (Rest): %v", err)
	}
	if len(namen) != 1 || namen[0] != "Unser Schulbuchhändler" {
		t.Errorf("Rest nicht erkannt: %v", namen)
	}
}
