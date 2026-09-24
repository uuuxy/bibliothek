package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

// Spalten-Gate des Zugangskontos in der Auskunft (24.09.2026) — das Gegenstück zu
// TestDsgvoAuskunft_KenntJedeLeserSpalte für die Tabelle benutzer. Bekommt das Konto eine
// neue Spalte, steht sie in der Auskunft oder hier mit Begründung.
//
// Geprüft wird die Spaltenliste vor FROM, nicht die ganze Abfrage: leser_id steht im WHERE
// und wäre sonst „gelesen", ohne in der Auskunft zu stehen.
func TestDsgvoKonto_KenntJedeBenutzerSpalte(t *testing.T) {
	pool := pgTestPool(t)
	ausnahmen := map[string]string{
		"leser_id": "die Verknüpfung zum Leser; sie steht als hat_zugangskonto in den Stammdaten der Auskunft",
	}

	auswahl, _, gefunden := strings.Cut(DsgvoKontoSQL, "FROM benutzer")
	if !gefunden {
		t.Fatal("Liveness: DsgvoKontoSQL liest nicht mehr FROM benutzer — Gate zeigt ins Leere")
	}
	rows, err := pool.Query(context.Background(),
		`SELECT column_name FROM information_schema.columns WHERE table_name = 'benutzer' ORDER BY ordinal_position`)
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
	if len(spalten) < 8 {
		t.Fatalf("Liveness: nur %d Spalten von benutzer gelesen", len(spalten))
	}
	for _, c := range spalten {
		if _, ok := ausnahmen[c]; ok {
			continue
		}
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(c) + `\b`).MatchString(auswahl) {
			t.Errorf("Spalte benutzer.%s fehlt in der Auskunft (DsgvoKontoSQL) — aufnehmen oder begründet ausnehmen", c)
		}
	}
	for c := range ausnahmen {
		gibtEs := false
		for _, s := range spalten {
			gibtEs = gibtEs || s == c
		}
		if !gibtEs {
			t.Errorf("Ausnahme %q ist keine Spalte von benutzer mehr — aus der Liste nehmen", c)
		}
	}
}
