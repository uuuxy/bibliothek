package repository_test

import (
	"context"
	"regexp"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Spalten-Gate der Art.-15-Auskunft: Der Struct-Kommentar
// versprach „sämtliche gespeicherten Stammdaten", aber seit Migration 084 fehlten
// lusd_bestaetigt_am und anonymized_at, seit 094 schul_eintritt_am und abgaenger_seit —
// das bestehende Gate (dsgvo_paar_vollstaendigkeit_test.go) prüft nur Tabellen, nicht
// Spalten. Hier: jede Spalte von leser steht in repository.DSGVOStammdatenSQL, oder sie steht
// mit Begründung in der Ausnahmeliste.
//
// Geprüft wird die Tabelle leser, nicht die Sicht schueler: Die Auskunft
// liest die Tabelle, weil es sie für jeden Leser gibt.
func TestDSGVOAuskunft_KenntJedeLeserSpalte(t *testing.T) {
	pool := pgtest.Pool(t)
	// Ausnahmen: Spalte → Begründung. Heute leer — jede Spalte von leser ist eine
	// gespeicherte Angabe über die Person.
	ausnahmen := map[string]string{}
	rows, err := pool.Query(context.Background(),
		`SELECT column_name FROM information_schema.columns WHERE table_name = 'leser' ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var spalten []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		spalten = append(spalten, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(spalten) < 20 {
		t.Fatalf("Liveness: nur %d Spalten gelesen", len(spalten))
	}
	for _, c := range spalten {
		if _, ok := ausnahmen[c]; ok {
			continue
		}
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(c) + `\b`).MatchString(repository.DSGVOStammdatenSQL) {
			t.Errorf("Spalte leser.%s fehlt in der Art.-15-Auskunft (repository.DSGVOStammdatenSQL) — aufnehmen oder begründet ausnehmen", c)
		}
	}
	for c := range ausnahmen {
		gefunden := false
		for _, s := range spalten {
			gefunden = gefunden || s == c
		}
		if !gefunden {
			t.Errorf("Ausnahme %q ist keine Spalte mehr — aus der Liste nehmen", c)
		}
	}
}
