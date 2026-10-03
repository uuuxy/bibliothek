package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/internal/pgtest"
)

// fahreMigrationUeberAltzeilen legt Titel am Trigger vorbei an — so, wie sie vor der Regel in
// die Tabelle kamen —, fährt die echte Migrationsdatei zweimal darüber und liefert die ISBN
// je Titel. Zweimal, weil die Migration beim erneuten Lauf still bleiben muss. Alles in einer
// Transaktion, die am Ende zurückgerollt wird; das Abschalten des Triggers gilt nur darin.
func fahreMigrationUeberAltzeilen(t *testing.T, datei string, altzeilen map[string]string) map[string]*string {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	exec(`ALTER TABLE buecher_titel DISABLE TRIGGER trg_titel_isbn_normalform`)
	for titel, isbn := range altzeilen {
		exec(`INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2)`, titel, isbn)
	}
	exec(`ALTER TABLE buecher_titel ENABLE TRIGGER trg_titel_isbn_normalform`)

	migration, err := os.ReadFile(filepath.Join("..", "migrations", datei))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	danach := make(map[string]*string, len(altzeilen))
	for titel := range altzeilen {
		var isbn *string
		if err := tx.QueryRow(ctx, `SELECT isbn FROM buecher_titel WHERE titel = $1`, titel).Scan(&isbn); err != nil {
			t.Fatalf("ISBN von %q lesen: %v", titel, err)
		}
		danach[titel] = isbn
	}
	return danach
}

// pruefeISBNs vergleicht die ISBN je Titel; ein leerer Sollwert heißt NULL.
func pruefeISBNs(t *testing.T, ist map[string]*string, soll map[string]string) {
	t.Helper()
	for titel, want := range soll {
		got := ist[titel]
		switch {
		case want == "" && got != nil:
			t.Errorf("%s: ISBN %q, want NULL", titel, *got)
		case want != "" && (got == nil || *got != want):
			t.Errorf("%s: ISBN %v, want %q", titel, isbnAnzeige(got), want)
		}
	}
}

// Migration 140 bringt ISBNs, die vor Migration 133 geschrieben wurden, in die Normalform
// (docs/OFFEN.md 5.5). Gemessen am Testserver am 23.09.2026: 10.061 Titel mit ISBN, einer
// weicht ab, keine zwei werden nach dem Normalisieren gleich.
func TestIsbnAltbestand_Migration140(t *testing.T) {
	danach := fahreMigrationUeberAltzeilen(t, "140_isbn_altbestand_normalform.sql", map[string]string{
		"Alt mit Strichen": "978-3-12-622042-2", // der Fall vom Testserver
		"Alt kleines x":    "3-12-345678-x",
		"Alt nur Leer":     "   ",
		// Eine Dublette: Die Normalform der Altzeile trägt schon ein anderer Titel.
		"Dublette neu": "9783161484100",
		"Dublette alt": "978-3-16-148410-0",
	})
	pruefeISBNs(t, danach, map[string]string{
		"Alt mit Strichen": "9783126220422",
		"Alt kleines x":    "312345678X",
		"Alt nur Leer":     "",
		"Dublette neu":     "9783161484100",
		// Die Dublette bleibt, wie sie ist: Zusammenlegen entscheidet ein Mensch, und die
		// Migration darf daran nicht scheitern (UNIQUE auf isbn).
		"Dublette alt": "978-3-16-148410-0",
	})
}

// Migration 157 rechnet zehnstellige ISBNs mit richtigem Prüfzeichen in die dreizehnstellige
// um. Am Testserver trägt keine Zeile eine solche (die Messung steht im Kopf der Migration);
// die Fälle hier stehen für eine Anlage, an der es sie gibt.
func TestIsbnAltbestand_Migration157(t *testing.T) {
	danach := fahreMigrationUeberAltzeilen(t, "157_isbn_eine_laenge.sql", map[string]string{
		"Zehn gültig":        "316148410X",
		"Zehn mit Strichen":  "0-306-40615-2",
		"Zehn falsch":        "3499500252", // die Nummer vom Testserver
		"Dreizehn daneben":   "9783499500251",
		"Dreizehn unberührt": "9783126220422",
		// Ein Paar: Die dreizehnstellige Form trägt schon ein anderer Titel.
		"Paar dreizehn": "9780804429573",
		"Paar zehn":     "080442957X",
		// Zweimal dasselbe Buch zehnstellig, verschieden geschrieben.
		"Doppelt glatt":     "3551551677",
		"Doppelt Striche":   "3-551-55167-7",
		"Keine ISBN":        "ISBN-0000000001",
		"Mit Beiwerk":       "3-12-345678-9 kart.",
		"Zehn kleines x":    "0-9752298-0-x",
		"Zehn ohne Treffer": "3-8252-0890-1",
	})
	pruefeISBNs(t, danach, map[string]string{
		"Zehn gültig":        "9783161484100",
		"Zehn mit Strichen":  "9780306406157",
		"Zehn falsch":        "3499500252",
		"Dreizehn daneben":   "9783499500251",
		"Dreizehn unberührt": "9783126220422",
		"Paar dreizehn":      "9780804429573",
		// Das Paar bleibt stehen: Zusammenlegen entscheidet ein Mensch, und die Migration
		// darf am UNIQUE-Index nicht scheitern.
		"Paar zehn":         "080442957X",
		"Doppelt glatt":     "3551551677",
		"Doppelt Striche":   "3-551-55167-7",
		"Keine ISBN":        "ISBN-0000000001",
		"Mit Beiwerk":       "3-12-345678-9 kart.",
		"Zehn kleines x":    "9780975229804",
		"Zehn ohne Treffer": "3825208901",
	})
}

func isbnAnzeige(s *string) string {
	if s == nil {
		return "NULL"
	}
	return `"` + *s + `"`
}
