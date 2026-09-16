package repository

import (
	"context"
	"strings"
	"testing"
)

// Die Sicht `schueler` zeigt die Schüler aus `leser` (Migration 124). Zwei Eigenschaften
// muss sie haben, und keine davon ist selbstverständlich.

// TestLeserSichtZeigtJedeSpalte: `CREATE VIEW ... SELECT *` friert die Spaltenliste beim
// Anlegen ein. Eine Spalte, die später zu `leser` kommt, erscheint in der Sicht NICHT von
// selbst — und weil 51 Abfragen über die Sicht lesen, wäre die neue Spalte für fast das
// ganze Programm nicht vorhanden.
//
// Der Fehler meldet sich laut („column does not exist"), aber erst zur Laufzeit und
// vielleicht erst in dem einen Bericht, den selten jemand öffnet. Dieses Gate zieht ihn
// auf den Tisch: Wer eine Spalte ergänzt, muss die Sicht neu anlegen (CREATE OR REPLACE
// VIEW in derselben Migration).
func TestLeserSichtZeigtJedeSpalte(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	spalten := func(relation string) []string {
		rows, err := pool.Query(ctx, `
			SELECT attname FROM pg_attribute
			WHERE attrelid = $1::regclass AND attnum > 0 AND NOT attisdropped
			ORDER BY attname`, relation)
		if err != nil {
			t.Fatalf("Spalten von %s: %v", relation, err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatalf("Spaltenname: %v", err)
			}
			out = append(out, s)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Spalten von %s: %v", relation, err)
		}
		return out
	}

	tabelle := spalten("leser")
	sicht := spalten("schueler")
	if len(tabelle) == 0 {
		t.Fatal("die Tabelle leser hat keine Spalten — der Test misst nichts")
	}

	inSicht := map[string]bool{}
	for _, s := range sicht {
		inSicht[s] = true
	}
	for _, s := range tabelle {
		if !inSicht[s] {
			t.Errorf("leser.%s fehlt in der Sicht schueler — die Sicht wurde nach dem "+
				"Ergänzen der Spalte nicht neu angelegt; jede der Abfragen über die Sicht "+
				"sieht diese Spalte nicht", s)
		}
	}

	inTabelle := map[string]bool{}
	for _, s := range tabelle {
		inTabelle[s] = true
	}
	for _, s := range sicht {
		if !inTabelle[s] {
			t.Errorf("die Sicht schueler zeigt %s, die Tabelle leser hat die Spalte nicht", s)
		}
	}
}

// TestLeserSichtLaesstKeinenKollegenDurch: WITH CHECK OPTION ist der Kern der Sicht, nicht
// Zierde. Ohne sie könnte ein Schreibweg durch die Sicht eine Zeile anlegen, die die Sicht
// danach nicht mehr zeigt — angelegt und sofort unsichtbar. Und jeder Weg, der heute
// „einen Schüler anlegt", könnte unbemerkt einen Kollegen erzeugen.
func TestLeserSichtLaesstKeinenKollegenDurch(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
		VALUES ('SICHT-1', 'Durch', 'Die Sicht', '7a', 2030, 'lehrkraft')`)
	if err == nil {
		// Aufräumen, damit der Fehlschlag nicht auch noch Daten hinterlässt.
		if _, delErr := pool.Exec(ctx, `DELETE FROM leser WHERE barcode_id = 'SICHT-1'`); delErr != nil {
			t.Logf("Aufräumen: %v", delErr)
		}
		t.Fatal("durch die Sicht schueler ließ sich eine Lehrkraft anlegen — " +
			"WITH CHECK OPTION fehlt")
	}
	if !strings.Contains(err.Error(), "check option") {
		t.Errorf("abgewiesen, aber von anderer Stelle: %v", err)
	}

	// Gegenprobe: derselbe Schreibvorgang mit art='schueler' geht durch. Ohne sie könnte
	// das Gate auch grün sein, wenn die Sicht gar nichts annimmt.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
		VALUES ('SICHT-2', 'Echter', 'Schueler', '7a', 2030, 'schueler')`); err != nil {
		t.Fatalf("ein Schüler ließ sich durch die Sicht nicht anlegen (%v) — dann prüft "+
			"der Fall oben nicht die Art, sondern irgendetwas anderes", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE barcode_id = 'SICHT-2'`); err != nil {
		t.Logf("Aufräumen: %v", err)
	}
}
