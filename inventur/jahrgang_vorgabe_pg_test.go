package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"bibliothek/internal/pgtest"
)

// Ein neuer Titel ohne Jahrgangsangabe bleibt „unbekannt": NULL in beiden Spalten, im Lesepfad
// 0 und 0. Eine Vorgabe gäbe ihm eine Spanne, die niemand eingetragen hat (bis Migration 162
// 5 bis 10: Die Inventur nach Klasse traf damit den ganzen Bestand), und eine 0 in der Spalte
// träfe jede Regel, die mit dem Jahrgang rechnet. Der Scanner-Dialog und der Listenimport
// schicken keinen Jahrgang.
func TestNeuerTitel_OhneJahrgang_BleibtUnbekannt(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	const ohne, mit = "978-9-99-200000-1", "978-9-99-200000-2"
	// Die Spalte trägt die ISBN ohne Bindestriche (isbn_normalform); aufgeräumt wird über die
	// Kennung der angelegten Titel.
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn IN (isbn_normalform($1), isbn_normalform($2))`, ohne, mit); err != nil {
		t.Fatal(err)
	}
	var angelegt []string
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id::text = ANY($1)`, angelegt); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	idOhne, err := repo.CreateBook(ctx, Book{ISBN: ohne, Title: "Scanner-Neuanlage"})
	if err != nil {
		t.Fatal(err)
	}
	angelegt = append(angelegt, idOhne)
	idMit, err := repo.CreateBook(ctx, Book{ISBN: mit, Title: "Lesebuch 5 bis 10", JahrgangVon: 5, JahrgangBis: 10})
	if err != nil {
		t.Fatal(err)
	}
	angelegt = append(angelegt, idMit)

	var unbekannt bool
	if err := pool.QueryRow(ctx,
		`SELECT jahrgang_von IS NULL AND jahrgang_bis IS NULL FROM buecher_titel WHERE id = $1`, idOhne).Scan(&unbekannt); err != nil {
		t.Fatal(err)
	}
	if !unbekannt {
		t.Error("ein Titel ohne Jahrgangsangabe trägt eine Spanne in der Datenbank, erwartet NULL und NULL")
	}
	var von, bis int
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(jahrgang_von, 0), coalesce(jahrgang_bis, 0) FROM buecher_titel WHERE id = $1`, idMit).Scan(&von, &bis); err != nil {
		t.Fatal(err)
	}
	if von != 5 || bis != 10 {
		t.Errorf("die eingetragene Spanne 5 bis 10 steht als %d bis %d in der Datenbank", von, bis)
	}

	// Die zwei Lesepfade der Titel-Verwaltung, der Einzel-Read der Maske und die Titelliste
	// (hier die Sicht „Ohne Exemplare": Die zwei Titel tragen keins): unbekannt ist 0 und 0,
	// die eingetragene Spanne bleibt.
	einzeln, err := repo.ListBooksByIDs(ctx, []string{idOhne, idMit})
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	liste, err := repo.ListBooks(ctx, "", nil, "978-9-99-200000", true)
	if err != nil {
		t.Fatalf("Titelliste lesen: %v", err)
	}
	soll := map[string][2]int{idOhne: {0, 0}, idMit: {5, 10}}
	for weg, buecher := range map[string][]Book{"Einzel-Read": einzeln, "Titelliste": liste} {
		gelesen := 0
		for _, b := range buecher {
			spanne, ok := soll[b.ID]
			if !ok {
				continue
			}
			gelesen++
			if b.JahrgangVon != spanne[0] || b.JahrgangBis != spanne[1] {
				t.Errorf("%s, %s: gelesen %d bis %d, erwartet %d bis %d", weg, b.Title, b.JahrgangVon, b.JahrgangBis, spanne[0], spanne[1])
			}
		}
		if gelesen != len(soll) {
			t.Errorf("%s: %d von %d Titeln gelesen", weg, gelesen, len(soll))
		}
	}
	// Cover-Upload und Cover-Abgleich lesen den Titel über GetBookByID: An der leeren Spalte
	// darf das Lesen nicht scheitern.
	if buch, err := repo.GetBookByID(ctx, idOhne); err != nil {
		t.Errorf("GetBookByID an einem Titel ohne Jahrgang: %v", err)
	} else if buch.JahrgangVon != 0 || buch.JahrgangBis != 0 {
		t.Errorf("GetBookByID: gelesen %d bis %d, erwartet 0 bis 0", buch.JahrgangVon, buch.JahrgangBis)
	}

	// Die Datenbank hält die Regel auch für Schreiber, die nicht durch pruefeJahrgangsSpanne
	// gehen: beide Spalten oder keine, 1 bis 13, „von" nicht über „bis".
	for name, sql := range map[string]string{
		"nur von":          `UPDATE buecher_titel SET jahrgang_von = 7, jahrgang_bis = NULL WHERE id = $1`,
		"nur bis":          `UPDATE buecher_titel SET jahrgang_von = NULL, jahrgang_bis = 9 WHERE id = $1`,
		"von über bis":     `UPDATE buecher_titel SET jahrgang_von = 9, jahrgang_bis = 7 WHERE id = $1`,
		"die Null":         `UPDATE buecher_titel SET jahrgang_von = 0, jahrgang_bis = 0 WHERE id = $1`,
		"über Jahrgang 13": `UPDATE buecher_titel SET jahrgang_von = 12, jahrgang_bis = 14 WHERE id = $1`,
	} {
		_, err := pool.Exec(ctx, sql, idOhne)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.ConstraintName != "chk_jahrgang_spanne" {
			t.Errorf("%s: die Datenbank muss die Spanne abweisen (chk_jahrgang_spanne), bekam %v", name, err)
		}
	}
}

// Beim Ändern gilt die Regel am Stand nach der Änderung: Die Maske schickt nur die geänderten
// Felder. Beide Felder geleert heißt „unbekannt"; eine halbe Spanne weist die Tür mit einem
// Satz ab, bevor die Datenbank sie zu einem Abbruch macht.
func TestTitelAendern_SpanneGanzOderGarNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	const isbn = "978-9-99-200000-3"
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = isbn_normalform($1)`, isbn); err != nil {
		t.Fatal(err)
	}
	id, err := repo.CreateBook(ctx, Book{ISBN: isbn, Title: "Bio 7 bis 9", Author: "Verlag", JahrgangVon: 7, JahrgangBis: 9})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	spanne := func() (von, bis *int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT jahrgang_von, jahrgang_bis FROM buecher_titel WHERE id = $1`, id).Scan(&von, &bis); err != nil {
			t.Fatal(err)
		}
		return von, bis
	}

	// Nur „von" geleert: 0 bis 9 ist keine Spanne.
	if err := repo.UpdateBook(ctx, id, Book{JahrgangVon: 0}, []string{"jahrgangVon"}, nil); !errors.Is(err, errJahrgangsSpanne) {
		t.Errorf("nur „von“ geleert: erwartet errJahrgangsSpanne, bekam %v", err)
	}
	// „von" über das gespeicherte „bis" gehoben.
	if err := repo.UpdateBook(ctx, id, Book{JahrgangVon: 11}, []string{"jahrgangVon"}, nil); !errors.Is(err, errJahrgangsSpanne) {
		t.Errorf("„von“ über „bis“: erwartet errJahrgangsSpanne, bekam %v", err)
	}
	if von, bis := spanne(); von == nil || bis == nil || *von != 7 || *bis != 9 {
		t.Errorf("nach den abgewiesenen Änderungen steht nicht mehr 7 bis 9 am Titel")
	}

	// Beide geleert: unbekannt.
	if err := repo.UpdateBook(ctx, id, Book{}, []string{"jahrgangVon", "jahrgangBis"}, nil); err != nil {
		t.Fatalf("beide Felder leeren: %v", err)
	}
	if von, bis := spanne(); von != nil || bis != nil {
		t.Errorf("nach dem Leeren beider Felder steht eine Spanne in der Datenbank, erwartet NULL und NULL")
	}

	// An einem Titel ohne Spanne nur „von" gesetzt.
	if err := repo.UpdateBook(ctx, id, Book{JahrgangVon: 7}, []string{"jahrgangVon"}, nil); !errors.Is(err, errJahrgangsSpanne) {
		t.Errorf("nur „von“ gesetzt: erwartet errJahrgangsSpanne, bekam %v", err)
	}
	// Ein anderes Feld lässt sich weiter ändern, und die Spanne bleibt unbekannt.
	if err := repo.UpdateBook(ctx, id, Book{Untertitel: "Neu"}, []string{"untertitel"}, nil); err != nil {
		t.Fatalf("anderes Feld ändern: %v", err)
	}
	if von, bis := spanne(); von != nil || bis != nil {
		t.Errorf("das Ändern eines anderen Felds hat eine Spanne eingetragen")
	}
}

// Beide Türen der Titel-Verwaltung weisen eine halbe Spanne mit demselben Satz ab, bevor die
// Datenbank sie abweist: Ohne die Prüfung an der Tür meldete das Anlegen nur „buch konnte
// nicht erstellt werden". Ein geleertes Zahlenfeld der Maske geht als null hinaus; zwei null
// leeren die Spanne.
func TestTitelTueren_JahrgangsSpanne(t *testing.T) {
	p := neueGenanntProbe(t)
	ctx := context.Background()
	const satz = "gehören zusammen"

	titel := genanntVorsatz + "halbe Spanne"
	rumpf, err := json.Marshal(map[string]any{"title": titel, "jahrgangVon": 7})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	p.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(rumpf)))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), satz) {
		t.Errorf("Anlegen mit nur „von“: Status %d: %s — erwartet 400 mit %q", rec.Code, rec.Body.String(), satz)
	}
	var angelegt int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM buecher_titel WHERE titel = $1`, titel).Scan(&angelegt); err != nil {
		t.Fatal(err)
	}
	if angelegt != 0 {
		t.Errorf("das abgewiesene Anlegen hat %d Titel geschrieben", angelegt)
	}

	// Ändern: Zwei null leeren die Spanne 5 bis 10 des Titels.
	id := p.lege()
	if status, antwort := p.aendere(id, map[string]any{"jahrgangVon": nil, "jahrgangBis": nil}); status != http.StatusOK {
		t.Fatalf("beide Felder leeren: Status %d: %s", status, antwort)
	}
	leer := p.zeile(id)
	if leer["jahrgang_von"] != "null" || leer["jahrgang_bis"] != "null" {
		t.Errorf("nach dem Leeren steht %s bis %s in der Tabelle, erwartet null und null", leer["jahrgang_von"], leer["jahrgang_bis"])
	}

	// Nur „von" an einem Titel ohne Spanne.
	if status, antwort := p.aendere(id, map[string]any{"jahrgangVon": 7}); status != http.StatusBadRequest || !strings.Contains(antwort, satz) {
		t.Errorf("Ändern mit nur „von“: Status %d: %s — erwartet 400 mit %q", status, antwort, satz)
	}
	if spalten := geaenderteSpalten(leer, p.zeile(id)); len(spalten) != 0 {
		t.Errorf("die abgewiesene Änderung schrieb %v", spalten)
	}
}

// Der Listenimport nennt keinen Jahrgang: Ein neuer Titel bleibt unbekannt, auf beiden Wegen
// (Stapel und Einzel-Rückfall). Trifft eine Zeile einen vorhandenen Titel, kommt eine fehlende
// Spanne als Ganzes aus der Zeile, und eine eingetragene bleibt stehen.
func TestListenimport_JahrgangBleibtUnbekanntUndWirdNachgetragen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	wege := map[string]func(Book) error{
		"UpsertBooksBatch": func(b Book) error {
			_, err := repo.UpsertBooksBatch(ctx, []Book{b})
			return err
		},
		"UpsertBook (Einzel-Rückfall)": func(b Book) error {
			_, err := repo.UpsertBook(ctx, b)
			return err
		},
	}
	isbns := map[string]string{"UpsertBooksBatch": "978-9-99-200000-4", "UpsertBook (Einzel-Rückfall)": "978-9-99-200000-5"}

	for weg, importiere := range wege {
		isbn := isbns[weg]
		loesche := func() {
			for _, sql := range []string{
				`DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = isbn_normalform($1))`,
				`DELETE FROM buecher_titel WHERE isbn = isbn_normalform($1)`,
			} {
				if _, err := pool.Exec(context.Background(), sql, isbn); err != nil {
					t.Errorf("%s: Aufräumen: %v", weg, err)
				}
			}
		}
		loesche()
		t.Cleanup(loesche)
		spanne := func() string {
			t.Helper()
			var s string
			if err := pool.QueryRow(ctx, `
				SELECT coalesce(jahrgang_von::text, 'leer') || ' bis ' || coalesce(jahrgang_bis::text, 'leer')
				FROM buecher_titel WHERE isbn = isbn_normalform($1)`, isbn).Scan(&s); err != nil {
				t.Fatalf("%s: %v", weg, err)
			}
			return s
		}

		zeile, err := verarbeiteImportZeile(ImportConfig{
			Ctx:       ctx,
			Row:       []string{isbn, "Die Welle", "Rhue"},
			ColIdx:    map[string]int{"isbn": 0, "titel": 1, "autor": 2, "fach": -1, "klasse": -1, "bestand": -1},
			Metadaten: offlineMetadatenClient(),
		})
		if err != nil || zeile == nil {
			t.Fatalf("%s: Zeile: %v", weg, err)
		}
		for _, schritt := range []struct {
			was      string
			von, bis int
			soll     string
		}{
			{"eine Liste ohne Jahrgang", 0, 0, "leer bis leer"},
			{"dieselbe Zeile mit 8 bis 9 füllt die Lücke", 8, 9, "8 bis 9"},
			{"dieselbe Zeile mit 5 bis 6 lässt die Spanne stehen", 5, 6, "8 bis 9"},
		} {
			buch := *zeile
			buch.JahrgangVon, buch.JahrgangBis = schritt.von, schritt.bis
			if err := importiere(buch); err != nil {
				t.Fatalf("%s, %s: %v", weg, schritt.was, err)
			}
			if ist := spanne(); ist != schritt.soll {
				t.Errorf("%s, %s: am Titel steht %s, erwartet %s", weg, schritt.was, ist, schritt.soll)
			}
		}
	}
}
