package lusd_test

import (
	"testing"

	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
)

// Parität Go ↔ SQL: lusd.KlassenSchluessel (Klassenvergleich im LUSD-Import) muss dasselbe
// liefern wie klassen_normkey, der Schlüssel des Triggers für die Klassen. Laufen die beiden
// auseinander, meldet die Vorschau Klassenwechsel, die der Trigger danach wegkanonisiert, oder
// übersieht echte.
func TestKlassenSchluessel_ParitaetZurSQLFunktion(t *testing.T) {
	pool := pgtest.Pool(t)
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
		if got := lusd.KlassenSchluessel(in); got != sql {
			t.Errorf("lusd.KlassenSchluessel(%q) = %q, SQL liefert %q", in, got, sql)
		}
	}
}
