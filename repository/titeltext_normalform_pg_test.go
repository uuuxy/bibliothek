package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Zeichen, um die es geht, stehen hier mit ihrer Nummer: Im Quelltext wären sie
// unsichtbar, und ein Editor könnte sie beim Speichern still umsetzen.
var (
	trema      = string(rune(0x0308)) // „a" + trema ist ein zerlegtes „ä", so liefert es die DNB
	geschuetzt = string(rune(0x00A0)) // geschütztes Leerzeichen, in Littera hinter „/" und „:"
	tabulator  = string(rune(0x0009))
	umbruch    = string(rune(0x000A))
	nullbreite = string(rune(0x200B)) // kein Leerraum nach Unicode: bleibt stehen
)

// Titeltexte stehen in einer Form in der Tabelle: kein Leerraum am Rand, Leerraum in Folge
// ist ein Leerzeichen, Umlaute zusammengesetzt (NFC). Ein Trigger setzt sie an jeder Tür
// (Migration 160). Geprüft wird die echte Migrationsdatei an einer Altzeile, die am Trigger
// vorbei entsteht. Alles in einer Transaktion, die am Ende zurückgerollt wird; das Abschalten
// des Triggers gilt nur darin.
func TestTiteltextNormalform_NachMigration160(t *testing.T) {
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
	type zeile struct{ titel, untertitel, autor, verlag, signatur string }
	lies := func(isbn string) zeile {
		t.Helper()
		var z zeile
		if err := tx.QueryRow(ctx, `
			SELECT titel, coalesce(untertitel, ''), coalesce(autor, ''), coalesce(verlag, ''),
			       coalesce(signatur, '')
			FROM buecher_titel WHERE isbn = $1`, isbn).
			Scan(&z.titel, &z.untertitel, &z.autor, &z.verlag, &z.signatur); err != nil {
			t.Fatalf("Titel %s lesen: %v", isbn, err)
		}
		return z
	}

	// Die Signatur trägt zwei Leerzeichen und ein zerlegtes Zeichen mit Absicht: Sie bleibt,
	// wie sie am Buch steht.
	alt := zeile{
		titel:      " Der Herr  der Ringe -" + geschuetzt + "Anha" + trema + "nge und Register ",
		untertitel: "U" + trema + "bersetzung   von 1969",
		autor:      "Ma" + trema + "rz," + tabulator + "Tobias",
		verlag:     "Klett-Cotta" + umbruch + "Stuttgart",
		signatur:   "Sk  Mu" + trema,
	}
	exec(`ALTER TABLE buecher_titel DISABLE TRIGGER trg_titel_text_normalform`)
	exec(`INSERT INTO buecher_titel (titel, untertitel, autor, verlag, signatur, isbn)
	      VALUES ($1, $2, $3, $4, $5, '9783608126044')`,
		alt.titel, alt.untertitel, alt.autor, alt.verlag, alt.signatur)
	exec(`ALTER TABLE buecher_titel ENABLE TRIGGER trg_titel_text_normalform`)
	// Die Altzeile steht, wie sie geschrieben wurde: Sonst prüfte der Rest nur den Trigger.
	if got := lies("9783608126044"); got != alt {
		t.Fatalf("Altzeile am Trigger vorbei:\n got %+q\nwant %+q", got, alt)
	}

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "160_titeltext_normalform.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	// Zweimal: Die Migration muss beim erneuten Lauf still bleiben.
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	// Die Migration zieht den Bestand nach.
	want := zeile{
		titel: "Der Herr der Ringe - Anhänge und Register", untertitel: "Übersetzung von 1969",
		autor: "März, Tobias", verlag: "Klett-Cotta Stuttgart",
		signatur: alt.signatur,
	}
	if got := lies("9783608126044"); got != want {
		t.Errorf("nach der Migration:\n got %+q\nwant %+q", got, want)
	}
	var abweichend int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM buecher_titel
		WHERE titel IS DISTINCT FROM titeltext_normalform(titel)
		   OR untertitel IS DISTINCT FROM titeltext_normalform(untertitel)
		   OR autor IS DISTINCT FROM titeltext_normalform(autor)
		   OR verlag IS DISTINCT FROM titeltext_normalform(verlag)`).Scan(&abweichend); err != nil {
		t.Fatalf("Bestand zählen: %v", err)
	}
	if abweichend != 0 {
		t.Errorf("%d Titel stehen nach der Migration nicht in der Form", abweichend)
	}

	// Der Trigger an jeder Tür: ein neuer Titel und eine Änderung. Ein fehlender Untertitel
	// bleibt NULL.
	exec(`INSERT INTO buecher_titel (titel, isbn) VALUES ($1, '9783551354068')`,
		"La  Peste /"+geschuetzt+"Bu"+trema+"cher ")
	if got := lies("9783551354068").titel; got != "La Peste / Bücher" {
		t.Errorf("neuer Titel: %+q", got)
	}
	exec(`UPDATE buecher_titel SET autor = $1 WHERE isbn = '9783551354068'`,
		"Camus,  Albert (U"+trema+"bers.)")
	if got := lies("9783551354068").autor; got != "Camus, Albert (Übers.)" {
		t.Errorf("geänderter Autor: %+q", got)
	}
	var ohneUntertitel bool
	if err := tx.QueryRow(ctx, `SELECT untertitel IS NULL FROM buecher_titel WHERE isbn = '9783551354068'`).
		Scan(&ohneUntertitel); err != nil {
		t.Fatal(err)
	}
	if !ohneUntertitel {
		t.Error("ein fehlender Untertitel muss NULL bleiben")
	}
}

// Go und die Datenbank bilden dieselbe Form: Der Suchtext und der Titelschlüssel der Importe
// gehen durch TiteltextNormalform, der gespeicherte Titel durch titeltext_normalform. Lägen
// sie auseinander, träfe ein Suchtext den Titel nicht, und ein Import legte ihn ein zweites
// Mal an.
func TestTiteltextNormalform_GoUndSQLSindZwillinge(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	faelle := []struct{ roh, form string }{
		// Formen aus der Littera-Sicherung von 2010.
		{"La  Peste", "La Peste"},
		{"A    Schulatlas/grün", "A Schulatlas/grün"},
		{"Mumienherz. Die Rückkehr des Seth /" + geschuetzt + "1", "Mumienherz. Die Rückkehr des Seth / 1"},
		{"Verlegt nach Hadamar :" + geschuetzt + "die Geschichte", "Verlegt nach Hadamar : die Geschichte"},
		// Was eine Maske oder eine Datei sonst bringen kann.
		{" Rand  und" + tabulator + "Tabulator ", "Rand und Tabulator"},
		{"Zeile" + umbruch + umbruch + "zwei", "Zeile zwei"},
		{"Anha" + trema + "nge  und Register", "Anhänge und Register"},
		{"ohne" + nullbreite + "Breite", "ohne" + nullbreite + "Breite"},
		{"Die Welle", "Die Welle"},
		{"   ", ""},
		{"", ""},
	}
	for _, f := range faelle {
		var sql string
		if err := pool.QueryRow(ctx, `SELECT titeltext_normalform($1)`, f.roh).Scan(&sql); err != nil {
			t.Fatalf("titeltext_normalform(%+q): %v", f.roh, err)
		}
		if sql != f.form {
			t.Errorf("titeltext_normalform(%+q) = %+q, erwartet %+q", f.roh, sql, f.form)
		}
		if got := TiteltextNormalform(f.roh); got != f.form {
			t.Errorf("TiteltextNormalform(%+q) = %+q, erwartet %+q", f.roh, got, f.form)
		}
	}

	// Die Fälle oben sind eine Stichprobe. Die Vollprobe: jedes Zeichen der ersten Unicode-Ebene
	// am Rand, einzeln und doppelt zwischen zwei Buchstaben, durch beide Seiten. U+0000 nimmt
	// Postgres nicht an, U+D800 bis U+DFFF sind keine Zeichen.
	var zeichen []rune
	var eingaben []string
	for r := rune(1); r <= 0xFFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		zeichen = append(zeichen, r)
		eingaben = append(eingaben, string(r)+"a"+string(r)+"b"+string(r)+string(r)+"c"+string(r))
	}
	rows, err := pool.Query(ctx, `
		SELECT titeltext_normalform(e) FROM unnest($1::text[]) WITH ORDINALITY AS t(e, n) ORDER BY n`, eingaben)
	if err != nil {
		t.Fatalf("Vollprobe: %v", err)
	}
	defer rows.Close()
	var abweichungen []string
	i, leerraum := 0, 0
	for rows.Next() {
		var sql string
		if err := rows.Scan(&sql); err != nil {
			t.Fatalf("Vollprobe lesen: %v", err)
		}
		if got := TiteltextNormalform(eingaben[i]); got != sql {
			abweichungen = append(abweichungen, fmt.Sprintf("U+%04X: Go %+q, SQL %+q", zeichen[i], got, sql))
		}
		if sql == "a b c" {
			leerraum++
		}
		i++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Vollprobe lesen: %v", err)
	}
	if i != len(eingaben) {
		t.Fatalf("Vollprobe: %d von %d Zeichen geprüft", i, len(eingaben))
	}
	if len(abweichungen) > 0 {
		t.Errorf("%d Zeichen, bei denen Go und SQL auseinanderliegen:\n%s",
			len(abweichungen), strings.Join(abweichungen, "\n"))
	}
	// Unicode nennt 25 Zeichen Leerraum (White_Space). Sähe die Probe keins, bewiese sie nichts.
	if leerraum != 25 {
		t.Errorf("%d Zeichen gelten der Datenbank als Leerraum, erwartet 25", leerraum)
	}
}
