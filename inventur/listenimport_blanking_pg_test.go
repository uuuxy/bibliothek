package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Listenimport ist für „Sammelkäufe und Spenden" gedacht (ListenImportWidget) —
// genau der Fall einer ISBN, die schon im Katalog steht. Dort soll er Exemplare
// dazulegen, nicht die Stammdaten ersetzen: Bestand gewinnt, der Import füllt Lücken.
//
// Bis zum 10.09.2026 setzte ON CONFLICT jede Spalte auf EXCLUDED.* (nur signatur und
// ist_lernmittel waren geschützt). Eine Datei `isbn;bestand` für 30 weitere Exemplare
// leerte Titel, Autor, Verlag, Beschreibung, setzte Jahr und Jahrgang auf 0, Medientyp auf
// „Buch", die Stufe auf die Vorgabe und warf ein von Hand hochgeladenes Cover weg — und
// die Antwort meldete „1 Titel importiert" (Bestands-Durchgang, Bugklasse
// Upsert-Blanking; der Littera-Zwilling ist seit 96c2f8c geschützt).
func TestListenimport_UeberschreibtVorhandeneStammdatenNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	wege := map[string]func(isbn string) error{
		"UpsertBooksBatch": func(isbn string) error {
			_, err := repo.UpsertBooksBatch(ctx, []Book{{ISBN: isbn, Stock: 2}})
			return err
		},
		"UpsertBook (Einzel-Rückfall)": func(isbn string) error {
			_, err := repo.UpsertBook(ctx, Book{ISBN: isbn, Stock: 2})
			return err
		},
	}
	isbns := map[string]string{"UpsertBooksBatch": "978-9-99-300000-1", "UpsertBook (Einzel-Rückfall)": "978-9-99-300000-2"}

	for weg, importiere := range wege {
		isbn := isbns[weg]
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = $1)`, isbn); err != nil {
			t.Fatal(err)
		}
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_titel (isbn, titel, autor, verlag, untertitel, beschreibung, erscheinungsjahr,
				cover_url, cover_status, grade_level, track, medientyp, jahrgang_von, jahrgang_bis, erweiterte_eigenschaften)
			VALUES ($1, 'Mathematik Neue Wege 7', 'Lütticken', 'Westermann', 'Arbeitsheft', 'Kuratiert', 2019,
				'/uploads/covers/hand.webp', 'FOUND', 7, 'G', 'DVD', 7, 8, '{"auflage":"3"}')
			ON CONFLICT (isbn) DO UPDATE SET titel = EXCLUDED.titel, autor = EXCLUDED.autor, verlag = EXCLUDED.verlag,
				untertitel = EXCLUDED.untertitel, beschreibung = EXCLUDED.beschreibung,
				erscheinungsjahr = EXCLUDED.erscheinungsjahr, cover_url = EXCLUDED.cover_url,
				grade_level = EXCLUDED.grade_level, track = EXCLUDED.track, medientyp = EXCLUDED.medientyp,
				jahrgang_von = EXCLUDED.jahrgang_von, jahrgang_bis = EXCLUDED.jahrgang_bis,
				erweiterte_eigenschaften = EXCLUDED.erweiterte_eigenschaften
			RETURNING id`, isbn).Scan(&id); err != nil {
			t.Fatalf("%s: Titel anlegen: %v", weg, err)
		}
		t.Cleanup(func() {
			for _, sql := range []string{`DELETE FROM buecher_exemplare WHERE titel_id = $1`, `DELETE FROM buecher_titel WHERE id = $1`} {
				if _, err := pool.Exec(context.Background(), sql, id); err != nil {
					t.Logf("Aufräumen: %v", err)
				}
			}
		})

		if err := importiere(isbn); err != nil {
			t.Fatalf("%s: %v", weg, err)
		}

		var titel, autor, verlag, untertitel, beschreibung, cover, track, medientyp, auflage string
		var jahr, stufe, von, bis, exemplare int
		if err := pool.QueryRow(ctx, `
			SELECT titel, coalesce(autor,''), coalesce(verlag,''), coalesce(untertitel,''), coalesce(beschreibung,''),
			       coalesce(cover_url,''), coalesce(track,''), coalesce(medientyp,''), coalesce(erweiterte_eigenschaften->>'auflage',''),
			       coalesce(erscheinungsjahr,0), coalesce(grade_level,0), jahrgang_von, jahrgang_bis,
			       (SELECT count(*) FROM buecher_exemplare e WHERE e.titel_id = t.id)::int
			FROM buecher_titel t WHERE id = $1`, id).Scan(&titel, &autor, &verlag, &untertitel, &beschreibung,
			&cover, &track, &medientyp, &auflage, &jahr, &stufe, &von, &bis, &exemplare); err != nil {
			t.Fatal(err)
		}
		soll := map[string][2]any{
			"titel": {titel, "Mathematik Neue Wege 7"}, "autor": {autor, "Lütticken"}, "verlag": {verlag, "Westermann"},
			"untertitel": {untertitel, "Arbeitsheft"}, "beschreibung": {beschreibung, "Kuratiert"},
			"cover_url": {cover, "/uploads/covers/hand.webp"}, "track": {track, "G"}, "medientyp": {medientyp, "DVD"},
			"erweiterte_eigenschaften.auflage": {auflage, "3"}, "erscheinungsjahr": {jahr, 2019},
			"grade_level": {stufe, 7}, "jahrgang_von": {von, 7}, "jahrgang_bis": {bis, 8},
		}
		for spalte, paar := range soll {
			if paar[0] != paar[1] {
				t.Errorf("%s: %s = %v, want %v (Bestand)", weg, spalte, paar[0], paar[1])
			}
		}
		if exemplare != 2 {
			t.Errorf("%s: %d Exemplare, want 2 (der Import legt dazu)", weg, exemplare)
		}
	}
}
