package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Klassensätze zählen am Buch, nicht an der Auflage (docs/OFFEN.md 4.18, Stufe 5, 25.09.2026).
// Ein Buch mit zwei Auflagen — die 3. von 2019 und die 4. von 2023 — in vier Klassen:
//
//   - 07B, zehn Kinder: je vier haben die 3. und die 4. Auflage, eines davon beide. Je Auflage
//     gezählt ist das kein Satz (vier sind unter der Mindestzahl), am Buch sind es sieben.
//     Gleichstand: Die Kachel zeigt die neueste Auflage.
//   - 08C, acht Kinder: sechs mit der 3., eines mit der 4. Die Kachel zeigt die 3. mit
//     sieben Lesern; die eine 4. Auflage steht in der Aufschlüsselung.
//   - 09D, sechs Kinder, alle mit der 4.: keine Aufschlüsselung.
//   - 10E: von Hand die 3. Auflage zugeordnet, alle sechs Kinder haben die 4. Die Kachel ist
//     die von Hand (einmal, nicht dazu noch eine aus den Ausleihen), die Aufschlüsselung
//     sagt, was die Klasse wirklich hat.
func TestGetClassGroups_ZaehltAmBuch(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const (
		werk = "00000000-0000-0000-0000-0000000a5000"
		alt  = "00000000-0000-0000-0000-0000000a5001"
		neu  = "00000000-0000-0000-0000-0000000a5002"
	)
	aufraeumen := []string{`DELETE FROM ausleihen`, `DELETE FROM class_books`, `DELETE FROM buecher_exemplare`,
		`DELETE FROM buecher_titel`, `DELETE FROM werke`, `DELETE FROM schueler WHERE barcode_id LIKE 'KB-%'`}
	vorbereitung := append(append([]string{}, aufraeumen...),
		`INSERT INTO werke (id) VALUES ('`+werk+`')`,
		`INSERT INTO buecher_titel (id, titel, auflage, erscheinungsjahr, ist_lernmittel, werk_id) VALUES
			('`+alt+`', 'Mathe 7', '3. Aufl.', 2019, true, '`+werk+`'),
			('`+neu+`', 'Mathe 7', '4. Aufl.', 2023, true, '`+werk+`')`,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 SELECT 'KB-B' || g, 'Kind' || g, 'Auflage', '07B', 2031 FROM generate_series(1, 10) g
		 UNION ALL SELECT 'KB-C' || g, 'Kind' || g, 'Auflage', '08C', 2030 FROM generate_series(1, 8) g
		 UNION ALL SELECT 'KB-D' || g, 'Kind' || g, 'Auflage', '09D', 2029 FROM generate_series(1, 6) g
		 UNION ALL SELECT 'KB-E' || g, 'Kind' || g, 'Auflage', '10E', 2028 FROM generate_series(1, 6) g`,
		`INSERT INTO class_books (class_name, book_id) VALUES ('10E', '`+alt+`')`,
	)
	for _, sql := range vorbereitung {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("%.40s: %v", sql, err)
		}
	}
	t.Cleanup(func() {
		for _, sql := range aufraeumen {
			if _, err := pool.Exec(ctx, sql); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		}
	})

	nr := 0
	leihe := func(titelID string, kinder ...string) {
		t.Helper()
		for _, kind := range kinder {
			nr++
			if _, err := pool.Exec(ctx, `
				WITH e AS (INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				           VALUES ($1, 'KB-X' || $2::int, true) RETURNING id)
				INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
				SELECT e.id, (SELECT id FROM schueler WHERE barcode_id = $3), now() + interval '30 days' FROM e`,
				titelID, nr, kind); err != nil {
				t.Fatalf("Ausleihe %s an %s: %v", titelID, kind, err)
			}
		}
	}
	leihe(alt, "KB-B1", "KB-B2", "KB-B3", "KB-B7")
	leihe(neu, "KB-B4", "KB-B5", "KB-B6", "KB-B7")
	leihe(alt, "KB-C1", "KB-C2", "KB-C3", "KB-C4", "KB-C5", "KB-C6")
	leihe(neu, "KB-C7")
	leihe(neu, "KB-D1", "KB-D2", "KB-D3", "KB-D4", "KB-D5", "KB-D6")
	leihe(neu, "KB-E1", "KB-E2", "KB-E3", "KB-E4", "KB-E5", "KB-E6")

	groups, err := NewBookRepository(pool).GetClassGroups(ctx, "", "")
	if err != nil {
		t.Fatalf("GetClassGroups: %v", err)
	}
	je := map[string][]ClassBook{}
	for _, g := range groups {
		je[g.ClassName] = g.Books
	}
	einzige := func(klasse string) ClassBook {
		t.Helper()
		if len(je[klasse]) != 1 {
			t.Fatalf("%s: %d Kacheln %+v — erwartet genau eine für das Buch", klasse, len(je[klasse]), je[klasse])
		}
		return je[klasse][0]
	}
	auflagen := func(klasse string, b ClassBook, soll ...repository.AuflageInKlasse) {
		t.Helper()
		if len(b.Auflagen) != len(soll) {
			t.Errorf("%s: Aufschlüsselung %+v — erwartet %+v", klasse, b.Auflagen, soll)
			return
		}
		for i := range soll {
			if b.Auflagen[i] != soll[i] {
				t.Errorf("%s: Aufschlüsselung %+v — erwartet %+v", klasse, b.Auflagen, soll)
				return
			}
		}
	}
	drei := func(kinder int) repository.AuflageInKlasse {
		return repository.AuflageInKlasse{ID: alt, Auflage: "3. Aufl.", Erscheinungsjahr: 2019, Kinder: kinder}
	}
	vier := func(kinder int) repository.AuflageInKlasse {
		return repository.AuflageInKlasse{ID: neu, Auflage: "4. Aufl.", Erscheinungsjahr: 2023, Kinder: kinder}
	}

	if b := einzige("07B"); b.ID != neu || b.Quelle != "ausleihe" || b.Leser != 7 {
		t.Errorf("07B: %s/%s/%d — erwartet die 4. Auflage aus Ausleihen mit 7 Lesern (ein Kind mit beiden zählt einmal)", b.ID, b.Quelle, b.Leser)
	} else {
		auflagen("07B", b, vier(4), drei(4))
	}
	if b := einzige("08C"); b.ID != alt || b.Quelle != "ausleihe" || b.Leser != 7 {
		t.Errorf("08C: %s/%s/%d — erwartet die 3. Auflage (die meisten Kinder) mit 7 Lesern", b.ID, b.Quelle, b.Leser)
	} else {
		auflagen("08C", b, drei(6), vier(1))
	}
	if b := einzige("09D"); b.ID != neu || b.Leser != 6 || b.Auflagen != nil {
		t.Errorf("09D: %s/%d/%+v — erwartet die 4. Auflage ohne Aufschlüsselung", b.ID, b.Leser, b.Auflagen)
	}
	if b := einzige("10E"); b.ID != alt || b.Quelle != "hand" {
		t.Errorf("10E: %s/%s — erwartet die Zuordnung von Hand (3. Auflage), keine zweite Kachel", b.ID, b.Quelle)
	} else {
		auflagen("10E", b, vier(6))
	}
}
