package repository

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Klassensätze aus zwei Quellen: Die Liste von Hand bleibt, dazu kommt jedes Mal gerechnet, was
// mehr als die Hälfte einer Klasse (mindestens fünf Kinder) gerade ausgeliehen hat. Geprüft an
// Postgres, weil der Kern ein Schwellwert über den Normschlüsseln der Klassen ist.
func TestLadeKlassensatzGruppen_HandUndAusAusleihen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	for _, sql := range []string{
		`DELETE FROM ausleihen`, `DELETE FROM class_books`, `DELETE FROM buecher_exemplare`, `DELETE FROM buecher_titel`,
		`DELETE FROM schueler WHERE barcode_id LIKE 'KA-%'`,
		// 07G1: sechs Kinder; 05F2: vier Kinder (zu klein für einen Satz).
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 SELECT 'KA-G' || g, 'Kind' || g, 'Klassensatz', '07G1', 2031 FROM generate_series(1, 6) g`,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 SELECT 'KA-F' || g, 'Kind' || g, 'Foerder', '05F2', 2033 FROM generate_series(1, 4) g`,
		`INSERT INTO buecher_titel (id, titel, isbn) VALUES
			('00000000-0000-0000-0000-00000000b001', 'Hand-Lektüre', '978-h1'),
			('00000000-0000-0000-0000-00000000b002', 'Beides: Hand und Ausleihe', '978-h2'),
			('00000000-0000-0000-0000-00000000b003', 'Nur aus Ausleihen', '978-a3'),
			('00000000-0000-0000-0000-00000000b004', 'Nur ein Leser', '978-a4'),
			('00000000-0000-0000-0000-00000000b005', 'Kleine Klasse komplett', '978-a5')`,
		// Exemplare: je Titel so viele, wie ausgeliehen werden.
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		 SELECT '00000000-0000-0000-0000-00000000b002', 'KA-B2-' || g, true FROM generate_series(1, 5) g`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		 SELECT '00000000-0000-0000-0000-00000000b003', 'KA-B3-' || g, true FROM generate_series(1, 5) g`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ('00000000-0000-0000-0000-00000000b004', 'KA-B4-1', true)`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		 SELECT '00000000-0000-0000-0000-00000000b005', 'KA-B5-' || g, true FROM generate_series(1, 4) g`,
		// Handliste: b001 für 07G1, b002 in Schreibvariante „7g1" (Trigger kanonisiert).
		`INSERT INTO class_books (class_name, book_id) VALUES ('07G1', '00000000-0000-0000-0000-00000000b001'), ('7g1', '00000000-0000-0000-0000-00000000b002')`,
		// Ausleihen: b002 und b003 an fünf von sechs Kindern der 07G1, b004 an eines,
		// b005 an alle vier der 05F2.
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 SELECT e.id, s.id, now() + interval '30 days'
		 FROM (SELECT id, row_number() OVER (ORDER BY barcode_id) AS r FROM buecher_exemplare WHERE barcode_id LIKE 'KA-B2-%') e
		 JOIN (SELECT id, row_number() OVER (ORDER BY barcode_id) AS r FROM schueler WHERE barcode_id LIKE 'KA-G%') s ON s.r = e.r`,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 SELECT e.id, s.id, now() + interval '30 days'
		 FROM (SELECT id, row_number() OVER (ORDER BY barcode_id) AS r FROM buecher_exemplare WHERE barcode_id LIKE 'KA-B3-%') e
		 JOIN (SELECT id, row_number() OVER (ORDER BY barcode_id) AS r FROM schueler WHERE barcode_id LIKE 'KA-G%') s ON s.r = e.r`,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 SELECT e.id, s.id, now() + interval '30 days' FROM buecher_exemplare e, schueler s WHERE e.barcode_id = 'KA-B4-1' AND s.barcode_id = 'KA-G6'`,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 SELECT e.id, s.id, now() + interval '30 days'
		 FROM (SELECT id, row_number() OVER (ORDER BY barcode_id) AS r FROM buecher_exemplare WHERE barcode_id LIKE 'KA-B5-%') e
		 JOIN (SELECT id, row_number() OVER (ORDER BY barcode_id) AS r FROM schueler WHERE barcode_id LIKE 'KA-F%') s ON s.r = e.r`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("%.40s: %v", sql, err)
		}
	}
	t.Cleanup(func() {
		for _, sql := range []string{`DELETE FROM ausleihen`, `DELETE FROM class_books`, `DELETE FROM buecher_exemplare`,
			`DELETE FROM buecher_titel`, `DELETE FROM schueler WHERE barcode_id LIKE 'KA-%'`} {
			if _, err := pool.Exec(ctx, sql); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		}
	})

	gruppen, err := NewTitelAbfragen(pool).LadeKlassensatzGruppen(ctx, "", "")
	if err != nil {
		t.Fatalf("ladeKlassensatzGruppen: %v", err)
	}
	je := map[string]map[string]KlassensatzTitel{}
	for _, g := range gruppen {
		je[g.Klasse] = map[string]KlassensatzTitel{}
		for _, b := range g.Titel {
			if _, doppelt := je[g.Klasse][b.Titel]; doppelt {
				t.Errorf("%s: %q steht zweimal in der Gruppe", g.Klasse, b.Titel)
			}
			je[g.Klasse][b.Titel] = b
		}
	}

	g1, ok := je["07G1"]
	if !ok {
		t.Fatalf("Gruppe 07G1 fehlt (Gruppen: %v)", gruppenNamen(gruppen))
	}
	if b := g1["Hand-Lektüre"]; b.Quelle != "hand" {
		t.Errorf("Hand-Lektüre: Quelle %q, erwartet hand", b.Quelle)
	}
	if b := g1["Beides: Hand und Ausleihe"]; b.Quelle != "hand" {
		t.Errorf("von Hand zugeordnet UND ausgeliehen: erscheint einmal als hand, war %q", b.Quelle)
	}
	if b := g1["Nur aus Ausleihen"]; b.Quelle != "ausleihe" || b.Leser != 5 {
		t.Errorf("Nur aus Ausleihen: Quelle %q Leser %d, erwartet ausleihe/5", b.Quelle, b.Leser)
	}
	if _, da := g1["Nur ein Leser"]; da {
		t.Error("ein einzelner Leser ist kein Klassensatz")
	}
	if _, da := je["05F2"]; da {
		t.Error("vier Kinder sind unter der Mindestzahl — 05F2 darf keine abgeleitete Gruppe bilden")
	}
}

// Die Oberstufe bildet Klassensätze aus Ausleihen wie die Mittelstufe, auch die Einführungsphase,
// deren Klasse nicht mit einer Zahl beginnt („ET1"). Sie steht als Jahrgang 11 vor 12T, und die
// Auswahl „Nur Oberstufe" zeigt nur sie.
func TestLadeKlassensatzGruppen_Oberstufe(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	sqls := []string{`DELETE FROM ausleihen`, `DELETE FROM class_books`, `DELETE FROM buecher_exemplare`,
		`DELETE FROM buecher_titel`, `DELETE FROM schueler WHERE barcode_id LIKE 'OS-%'`}
	for i, klasse := range []string{"12T1", "ET1", "07G1"} {
		// Sechs Kinder, fünf mit demselben Buch: mehr als die Hälfte und die Mindestzahl.
		sqls = append(sqls,
			fmt.Sprintf(`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			 SELECT 'OS-%d-' || g, 'Kind' || g, 'Oberstufe', '%s', 2029 FROM generate_series(1, 6) g`, i, klasse),
			fmt.Sprintf(`INSERT INTO buecher_titel (id, titel, isbn)
			 VALUES ('00000000-0000-0000-0000-00000000c00%d', 'Buch der %s', '978-os%d')`, i, klasse, i),
			fmt.Sprintf(`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			 SELECT '00000000-0000-0000-0000-00000000c00%d', 'OS-B%d-' || g, true FROM generate_series(1, 5) g`, i, i),
			fmt.Sprintf(`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			 SELECT e.id, s.id, now() + interval '30 days'
			 FROM (SELECT id, row_number() OVER (ORDER BY barcode_id) AS r FROM buecher_exemplare WHERE barcode_id LIKE 'OS-B%d-%%') e
			 JOIN (SELECT id, row_number() OVER (ORDER BY barcode_id) AS r FROM schueler WHERE barcode_id LIKE 'OS-%d-%%') s ON s.r = e.r`, i, i))
	}
	for _, sql := range sqls {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("%.40s: %v", sql, err)
		}
	}
	t.Cleanup(func() {
		for _, sql := range sqls[:5] {
			if _, err := pool.Exec(ctx, sql); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		}
	})

	for _, fall := range []struct {
		zweig string
		want  []string
	}{
		// Ohne Auswahl stehen die Zweige vorn (G vor allem ohne Zweigbuchstaben), dann die Oberstufe.
		{"", []string{"07G1", "ET1", "12T1"}},
		{zweigOberstufe, []string{"ET1", "12T1"}},
	} {
		gruppen, err := NewTitelAbfragen(pool).LadeKlassensatzGruppen(ctx, fall.zweig, "asc")
		if err != nil {
			t.Fatalf("ladeKlassensatzGruppen(%q): %v", fall.zweig, err)
		}
		if got := gruppenNamen(gruppen); !slices.Equal(got, fall.want) {
			t.Fatalf("Auswahl %q: Gruppen %v, erwartet %v", fall.zweig, got, fall.want)
		}
		for _, g := range gruppen {
			if len(g.Titel) != 1 || g.Titel[0].Quelle != "ausleihe" || g.Titel[0].Leser != 5 {
				t.Errorf("%s: %+v, erwartet ein Buch aus Ausleihen mit fünf Lesern", g.Klasse, g.Titel)
			}
		}
	}
}

// Die Auswahl der Oberfläche schickt für „Nur Oberstufe" denselben Wert, den der Server kennt.
// Die beiden Seiten verbindet sonst nur die Schreibweise.
func TestZweigOberstufe_StehtInDerAuswahl(t *testing.T) {
	inhalt, err := os.ReadFile("../frontend/src/inventur/lib/components/admin/KlassenUebersicht.svelte")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(inhalt), "{ value: '"+zweigOberstufe+"', label: 'Nur Oberstufe' }") {
		t.Errorf("KlassenUebersicht.svelte bietet „Nur Oberstufe“ nicht mit dem Wert %q an", zweigOberstufe)
	}
}

func gruppenNamen(gruppen []KlassensatzGruppe) []string {
	namen := make([]string, 0, len(gruppen))
	for _, g := range gruppen {
		namen = append(namen, g.Klasse)
	}
	return namen
}
