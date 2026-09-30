package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/internal/pgtest"
)

// Migration 154 setzt Titeltexte zusammen (NFC): der Trigger an jeder Tür und das Nachziehen
// des Bestands. Gemessen am Testserver am 30.09.2026: 19 Titel mit zerlegten Zeichen, alle
// aus der DNB. Geprüft wird die echte Migrationsdatei an Altzeilen, die am Trigger vorbei
// entstehen — so, wie sie vor der Migration in die Tabelle kamen. Alles in einer Transaktion,
// die am Ende zurückgerollt wird; das Abschalten des Triggers gilt nur darin.
func TestTiteltextNFC_Migration154(t *testing.T) {
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
	type zeile struct{ titel, untertitel, autor, verlag, beschreibung, signatur string }
	lies := func(isbn string) zeile {
		t.Helper()
		var z zeile
		if err := tx.QueryRow(ctx, `
			SELECT titel, coalesce(untertitel, ''), coalesce(autor, ''), coalesce(verlag, ''),
			       coalesce(beschreibung, ''), coalesce(signatur, '')
			FROM buecher_titel WHERE isbn = $1`, isbn).
			Scan(&z.titel, &z.untertitel, &z.autor, &z.verlag, &z.beschreibung, &z.signatur); err != nil {
			t.Fatalf("Titel %s lesen: %v", isbn, err)
		}
		return z
	}

	// Die Signatur trägt ein zerlegtes Zeichen mit Absicht: Sie bleibt, wie sie am Buch steht.
	const zerlegteSignatur = "Sk Mu\u0308"
	exec(`ALTER TABLE buecher_titel DISABLE TRIGGER trg_titel_text_nfc`)
	exec(`INSERT INTO buecher_titel (titel, untertitel, autor, verlag, beschreibung, signatur, isbn)
	      VALUES ($1, $2, $3, $4, $5, $6, '9783608126044')`,
		"Der Herr der Ringe - Anha\u0308nge und Register", "U\u0308bersetzung", "Ma\u0308rz, Tobias",
		"Klett-Cotta Stuttgart", "Zu Tolkiens Welt. Fu\u0308r Leser", zerlegteSignatur)
	exec(`ALTER TABLE buecher_titel ENABLE TRIGGER trg_titel_text_nfc`)

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "154_titeltext_nfc.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	// Zweimal: Die Migration muss beim erneuten Lauf still bleiben.
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	want := zeile{
		titel: "Der Herr der Ringe - Anhänge und Register", untertitel: "Übersetzung",
		autor: "März, Tobias", verlag: "Klett-Cotta Stuttgart",
		beschreibung: "Zu Tolkiens Welt. Für Leser", signatur: zerlegteSignatur,
	}
	if got := lies("9783608126044"); got != want {
		t.Errorf("nach der Migration:\n got %+q\nwant %+q", got, want)
	}

	// Der Trigger an jeder Tür: ein neuer Titel und eine Änderung, beide zerlegt geschrieben.
	exec(`INSERT INTO buecher_titel (titel, isbn) VALUES ($1, '9783551354068')`, "Harry Potter und der Halbblutprinz, Bu\u0308cher")
	if got := lies("9783551354068").titel; got != "Harry Potter und der Halbblutprinz, Bücher" {
		t.Errorf("neuer Titel: %+q — erwartet zusammengesetzt", got)
	}
	exec(`UPDATE buecher_titel SET autor = $1 WHERE isbn = '9783551354068'`, "Rowling, J. K. (U\u0308bers.)")
	if got := lies("9783551354068").autor; got != "Rowling, J. K. (Übers.)" {
		t.Errorf("geänderter Autor: %+q — erwartet zusammengesetzt", got)
	}
}
