package inventur

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Katalogliste, Einzel-Read und Klassenkachel nennen neben Bestand und verfügbar die
// bestellten, noch nicht eingetroffenen Exemplare — dieselbe Grenze wie die Theken-Suche
// (repository.SQLBestandImZulauf, dort TestTitelSucheNenntDenBestand). Ohne die dritte Zahl
// stand über einem Titel, dessen Exemplare alle unterwegs sind, „Keine Exemplare"
// (docs/OFFEN.md 5.5).
func TestBestandNenntDenZulauf(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	marke := fmt.Sprintf("ZULAUF%d", time.Now().UnixNano())
	klasse := "Z" + marke[len(marke)-8:]

	titel := func(name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ($1, $2, 'Buch') RETURNING id`,
			name+" "+marke, marke).Scan(&id); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO class_books (class_name, book_id) VALUES ($1, $2)`, klasse, id); err != nil {
			t.Fatalf("Klassenzuordnung: %v", err)
		}
		return id
	}
	nr := 0
	exemplar := func(titelID string, bestellstatus *string) {
		t.Helper()
		nr++
		// chk_exemplar_bestellstatus_nur_im_zulauf: im Zulauf heißt nicht ausleihbar.
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
			VALUES ($1, $2, $3, $4)`, titelID, fmt.Sprintf("B-ZUL-%d-%s", nr, marke), bestellstatus == nil, bestellstatus); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE autor = $1)`,
			`DELETE FROM buecher_titel WHERE autor = $1`,
		} {
			if _, err := pool.Exec(auf, sql, marke); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})

	bestellt, unterwegs := "bestellt", "im_zulauf"
	nurBestellt := titel("Nur bestellt")
	exemplar(nurBestellt, &bestellt)
	teils := titel("Eins da zwei bestellt")
	exemplar(teils, nil)
	exemplar(teils, &bestellt)
	exemplar(teils, &unterwegs)

	// {gesamt, verfügbar, im Zulauf}
	erwartet := map[string][3]int{nurBestellt: {0, 0, 1}, teils: {1, 1, 2}}
	pruefe := func(tuer, id string, gesamt, verfuegbar, imZulauf int) {
		t.Helper()
		soll := erwartet[id]
		if gesamt != soll[0] || verfuegbar != soll[1] || imZulauf != soll[2] {
			t.Errorf("%s: gesamt %d, verfügbar %d, im Zulauf %d — erwartet %d, %d und %d",
				tuer, gesamt, verfuegbar, imZulauf, soll[0], soll[1], soll[2])
		}
	}

	liste, err := repo.ListBooks(ctx, "", nil, marke, false)
	if err != nil {
		t.Fatalf("ListBooks: %v", err)
	}
	einzeln, err := repo.ListBooksByIDs(ctx, []string{nurBestellt, teils})
	if err != nil {
		t.Fatalf("ListBooksByIDs: %v", err)
	}
	for tuer, buecher := range map[string][]Book{"ListBooks": liste, "ListBooksByIDs": einzeln} {
		if len(buecher) != 2 {
			t.Fatalf("%s: %d Titel, erwartet 2", tuer, len(buecher))
		}
		for _, b := range buecher {
			pruefe(tuer+"/"+b.Title, b.ID, b.Gesamt, b.Verfuegbar, b.ImZulauf)
		}
	}

	gruppen, err := repo.GetClassGroups(ctx, klasse, "")
	if err != nil {
		t.Fatalf("GetClassGroups: %v", err)
	}
	gesehen := 0
	for _, g := range gruppen {
		for _, b := range g.Books {
			if _, ok := erwartet[b.ID]; ok {
				gesehen++
				pruefe("GetClassGroups/"+b.Title, b.ID, b.Gesamt, b.Verfuegbar, b.ImZulauf)
			}
		}
	}
	if gesehen != 2 {
		t.Errorf("GetClassGroups: %d Titel gesehen, erwartet 2", gesehen)
	}
}
