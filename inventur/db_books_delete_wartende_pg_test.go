package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Frage 12 „Gegenrichtung Schema" (06.09.2026): An `buecher_titel` hängen vier Kinder
// mit ON DELETE CASCADE. Der Löschpfad kannte nur die Exemplare — Vormerkungen,
// Klassensatz-Reservierungen und Klassensatz-Zuordnungen fielen lautlos.
//
// Dahinter stehen Menschen, die gewartet haben: ein Schüler auf Platz 1 der
// Warteschlange, eine Lehrkraft mit einer angemeldeten Klassensatz-Anforderung. Ohne
// Protokollzeile ist an der Theke später nicht einmal nachvollziehbar, dass sie je
// gewartet haben. Dieselbe Antwort wie bei offenen Ausleihen und Forderungen: Was
// verschwindet, steht vorher im Protokoll.
func TestDeleteBooks_WartendeHinterlassenEineSpur(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	eins := func(was, sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}

	var titelID, schuelerID string
	t.Cleanup(func() {
		ctx := context.Background()
		if titelID != "" {
			if _, err := pool.Exec(ctx, `DELETE FROM audit_log WHERE datensatz_id = $1`, titelID); err != nil {
				t.Errorf("Aufräumen audit_log: %v", err)
			}
			if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
				t.Errorf("Aufräumen Titel: %v", err)
			}
		}
		if schuelerID != "" {
			if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE id = $1`, schuelerID); err != nil {
				t.Errorf("Aufräumen Schüler: %v", err)
			}
		}
	})

	klasse := eins("Klasse sichern",
		`INSERT INTO klassen (name) VALUES ('05A') ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING name`)
	titelID = eins("Titel anlegen",
		`INSERT INTO buecher_titel (titel, autor) VALUES ('Warteschlangen-Titel', 'Test') RETURNING id`)
	schuelerID = eins("Schüler anlegen",
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('W-1', 'Wanda', 'Wartend', $1, 2030) RETURNING id`, klasse)

	if _, err := pool.Exec(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')`,
		titelID, schuelerID); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl) VALUES ($1, $2, 30)`,
		titelID, klasse); err != nil {
		t.Fatalf("Klassensatz-Reservierung anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO class_books (class_name, book_id) VALUES ($1, $2)`, klasse, titelID); err != nil {
		t.Fatalf("Klassensatz-Zuordnung anlegen: %v", err)
	}

	if err := repo.DeleteBooks(ctx, []string{titelID}); err != nil {
		t.Fatalf("DeleteBooks: %v", err)
	}

	// Die drei Zeilen sind weg (das tut der CASCADE) — aber jede hat eine Spur.
	for _, tabelle := range []string{"vormerkungen", "klassensatz_reservierungen", "class_books"} {
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM audit_log
			WHERE datensatz_id = $1 AND tabelle = $2
			  AND details->>'action' = 'titel_geloescht_mit_offenem_bezug'`, titelID, tabelle).Scan(&n); err != nil {
			t.Fatalf("Protokoll lesen (%s): %v", tabelle, err)
		}
		if n != 1 {
			t.Errorf("%s: erwartet 1 Protokollzeile, waren %d — der Bezug fällt per CASCADE, "+
				"und ohne Spur weiß später niemand, dass jemand gewartet hat", tabelle, n)
		}
	}

	// Und die Spur nennt, WEN es betrifft — eine Zeile ohne Namen hilft an der Theke nicht.
	var betrifft string
	if err := pool.QueryRow(ctx, `
		SELECT details->>'betrifft' FROM audit_log
		WHERE datensatz_id = $1 AND tabelle = 'vormerkungen'`, titelID).Scan(&betrifft); err != nil {
		t.Fatalf("Betroffenen lesen: %v", err)
	}
	if betrifft != "Wanda Wartend" {
		t.Errorf("Vormerkungs-Spur: erwartet den Schülernamen, war %q", betrifft)
	}
}
