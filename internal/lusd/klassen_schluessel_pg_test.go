package lusd_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
)

// Parität Go ↔ SQL: lusd.KlassenSchluessel (Klassenvergleich im LUSD-Import) muss dasselbe
// liefern wie klassen_normkey, der Schlüssel des Triggers für die Klassen. Laufen die beiden
// auseinander, meldet die Vorschau Klassenwechsel, die der Trigger danach wegkanonisiert, oder
// übersieht echte. Ohne -fuzz laufen nur die Startwerte; wie gesucht wird, steht in
// docs/ARCHITEKTUR.md 8.14.
func FuzzKlassenSchluessel_ParitaetZurSQLFunktion(f *testing.F) {
	for _, in := range []string{
		"05A", "5a", " 5a ", "5 a", "Q2", "q 2", "007b", "E1", "E 1", "ABG", "10a", "10 A",
		"", " ", "0", "00", "5a/1", "Förderstufe 5", "11T1", "Ab",
		// Tab und Zeilenumbruch am Rand: btrim schneidet nur Leerzeichen ab.
		"\t5a", "5a\n",
	} {
		f.Add(in)
	}
	f.Fuzz(func(t *testing.T, in string) {
		// Postgres nimmt in Text kein ungültiges UTF-8 und kein Nullbyte; solche Werte erreichen
		// die Spalte nie.
		if !utf8.ValidString(in) || strings.ContainsRune(in, 0) {
			t.Skip()
		}
		pool := pgtest.Pool(t)
		var sql string
		if err := pool.QueryRow(t.Context(), `SELECT klassen_normkey($1)`, in).Scan(&sql); err != nil {
			t.Fatalf("klassen_normkey(%q): %v", in, err)
		}
		if got := lusd.KlassenSchluessel(in); got != sql {
			t.Errorf("lusd.KlassenSchluessel(%q) = %q, SQL liefert %q", in, got, sql)
		}
	})
}
