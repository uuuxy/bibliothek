package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/internal/pgtest"
)

// Migration 140 bringt ISBNs, die vor Migration 133 geschrieben wurden, in die Normalform
// (docs/OFFEN.md 5.5). Gemessen am Testserver am 23.09.2026: 10.061 Titel mit ISBN, einer
// weicht ab, keine zwei werden nach dem Normalisieren gleich.
//
// Geprüft wird die echte Migrationsdatei an Altzeilen, die am Trigger vorbei entstehen —
// so, wie sie vor Migration 133 in die Tabelle kamen. Alles in einer Transaktion, die am
// Ende zurückgerollt wird; das Abschalten des Triggers gilt nur darin.
func TestIsbnAltbestand_Migration140(t *testing.T) {
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
	isbnVon := func(titel string) *string {
		t.Helper()
		var isbn *string
		if err := tx.QueryRow(ctx, `SELECT isbn FROM buecher_titel WHERE titel = $1`, titel).Scan(&isbn); err != nil {
			t.Fatalf("ISBN von %q lesen: %v", titel, err)
		}
		return isbn
	}

	exec(`ALTER TABLE buecher_titel DISABLE TRIGGER trg_titel_isbn_normalform`)
	for titel, isbn := range map[string]string{
		"Alt mit Strichen": "978-3-12-622042-2", // der Fall vom Testserver
		"Alt kleines x":    "3-12-345678-x",
		"Alt nur Leer":     "   ",
		// Eine Dublette: Die Normalform der Altzeile trägt schon ein anderer Titel.
		"Dublette neu": "9783161484100",
		"Dublette alt": "978-3-16-148410-0",
	} {
		exec(`INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2)`, titel, isbn)
	}
	exec(`ALTER TABLE buecher_titel ENABLE TRIGGER trg_titel_isbn_normalform`)

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "140_isbn_altbestand_normalform.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	// Zweimal: Die Migration muss beim erneuten Lauf still bleiben.
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	pruefe := func(titel string, want *string) {
		t.Helper()
		got := isbnVon(titel)
		switch {
		case want == nil && got != nil:
			t.Errorf("%s: ISBN %q, want NULL", titel, *got)
		case want != nil && (got == nil || *got != *want):
			t.Errorf("%s: ISBN %v, want %q", titel, isbnAnzeige(got), *want)
		}
	}
	text := func(s string) *string { return &s }
	pruefe("Alt mit Strichen", text("9783126220422"))
	pruefe("Alt kleines x", text("312345678X"))
	pruefe("Alt nur Leer", nil)
	pruefe("Dublette neu", text("9783161484100"))
	// Die Dublette bleibt, wie sie ist: Zusammenlegen entscheidet ein Mensch, und die
	// Migration darf daran nicht scheitern (UNIQUE auf isbn).
	pruefe("Dublette alt", text("978-3-16-148410-0"))
}

func isbnAnzeige(s *string) string {
	if s == nil {
		return "NULL"
	}
	return `"` + *s + `"`
}
