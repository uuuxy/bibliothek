package api

import (
	"testing"
)

// Parität Go ↔ SQL: klassenNormkey (Klassenvergleich im LUSD-Import) muss dasselbe
// liefern wie klassen_normkey (Migration 079, der Trigger-Schlüssel des Klassen-
// Vokabulars). Driften die beiden, meldet die Vorschau Klassenwechsel, die der Trigger
// anschließend wegkanonisiert — oder übersieht echte. Zwei Implementierungen derselben
// Regel brauchen ein Gate, das sie gegeneinander hält.
func TestKlassenNormkey_ParitaetZurSQLFunktion(t *testing.T) {
	pool := pgTestPool(t)
	ctx := t.Context()

	eingaben := []string{
		"05A", "5a", " 5a ", "5 a", "Q2", "q 2", "007b", "E1", "E 1", "ABG", "10a", "10 A",
		"", " ", "0", "00", "5a/1", "Förderstufe 5", "11T1", "Ab",
	}
	for _, in := range eingaben {
		var sql string
		if err := pool.QueryRow(ctx, `SELECT klassen_normkey($1)`, in).Scan(&sql); err != nil {
			t.Fatalf("klassen_normkey(%q): %v", in, err)
		}
		if got := klassenNormkey(in); got != sql {
			t.Errorf("klassenNormkey(%q) = %q, SQL liefert %q", in, got, sql)
		}
	}
}
