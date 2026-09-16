package db

import (
	"context"
	"testing"
)

// Die Sicht `schueler` zeigt ALLE Spalten der Tabelle `leser`.
//
// `CREATE VIEW schueler AS SELECT * FROM leser` friert die Spaltenliste im Moment des
// Anlegens ein: Ein späteres `ALTER TABLE leser ADD COLUMN` taucht in der Sicht NICHT auf.
// Das ist die unangenehme Sorte Abweichung — nichts bricht, es fehlt nur etwas: Die 51
// Abfragen, die „Schüler" meinen, läsen die neue Spalte nie, ein INSERT durch die Sicht
// setzte sie nie, und beim Lesen merkt man es nicht, weil die Abfrage ja durchläuft.
//
// Reparatur bei Rot: In derselben Migration, die die Spalte anlegt, die Sicht neu
// erzeugen (DROP VIEW schueler; CREATE VIEW … WITH CHECK OPTION) — und schema.sql
// nachziehen, damit der frische Weg gleich bleibt.
func TestSichtSchuelerZeigtAlleSpaltenDerLesertabelle(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	spalten := func(relation string) []string {
		t.Helper()
		rows, err := pool.Query(ctx, `
			SELECT column_name FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1
			ORDER BY column_name`, relation)
		if err != nil {
			t.Fatalf("Spalten von %s lesen: %v", relation, err)
		}
		defer rows.Close()
		var namen []string
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Fatalf("Spalte von %s lesen: %v", relation, err)
			}
			namen = append(namen, n)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Spalten von %s lesen: %v", relation, err)
		}
		return namen
	}

	tabelle := spalten("leser")
	sicht := spalten("schueler")
	if len(tabelle) == 0 || len(sicht) == 0 {
		t.Fatal("Tabelle oder Sicht ohne Spalten — das Gate wäre still grün")
	}

	inSicht := make(map[string]bool, len(sicht))
	for _, n := range sicht {
		inSicht[n] = true
	}
	for _, n := range tabelle {
		if !inSicht[n] {
			t.Errorf("leser.%s fehlt in der Sicht schueler — die Sicht wurde nach dem "+
				"Anlegen der Spalte nicht neu erzeugt", n)
		}
	}
	// Die Gegenrichtung ist genauso ein Fehler: Eine Spalte, die es in der Tabelle nicht
	// mehr gibt, kann die Sicht gar nicht behalten — sie wäre dann von Hand gebaut.
	inTabelle := make(map[string]bool, len(tabelle))
	for _, n := range tabelle {
		inTabelle[n] = true
	}
	for _, n := range sicht {
		if !inTabelle[n] {
			t.Errorf("die Sicht schueler zeigt %q, was die Tabelle leser nicht hat", n)
		}
	}
}
