package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Der Bestand in der Trefferliste (Protokoll des Medienzentrums vom 16.09.2026, Punkt 4:
// „Bücher, zu denen es keine Exemplare gibt, tauchen in der Trefferliste auf").
//
// Ein Titel ohne Exemplare ist ein legitimer Zustand — angelegt ohne Bestandsangabe, oder
// Altbestand aus Littera. Verstecken wäre also falsch. Die Trefferliste muss es SAGEN,
// und dafür braucht sie die Zahlen.
//
// Vier Fälle, weil „kein Exemplar da" vier verschiedene Ursachen hat und die Theke sie
// unterscheiden muss: gar keins, alle ausgesondert, alle verliehen, alles im Zulauf.
func TestTitelSucheNenntDenBestand(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	marke := "BSTND" + suffix

	titel := func(t *testing.T, name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_titel (titel, autor, medientyp)
			VALUES ($1, $2, 'Buch') RETURNING id`, name, marke).Scan(&id); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		return id
	}
	exemplar := func(t *testing.T, titelID, barcode string, ausleihbar, ausgesondert bool, bestellstatus *string) string {
		t.Helper()
		var id string
		grund := (*string)(nil)
		if ausgesondert {
			g := "AUSSORTIERT"
			grund = &g
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund, bestellstatus)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			titelID, barcode, ausleihbar, ausgesondert, grund, bestellstatus).Scan(&id); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		return id
	}

	// Der Titel ohne jedes Exemplar — die ID braucht niemand, es hängt nichts daran.
	titel(t, "Ohne Exemplar "+marke)

	ausgesondert := titel(t, "Nur ausgesondert "+marke)
	exemplar(t, ausgesondert, "B-BST-A"+suffix, false, true, nil)

	// Im Zulauf: Der CHECK chk_exemplar_bestellstatus_nur_im_zulauf verlangt dafür
	// ist_ausleihbar = false — ein bestelltes Buch steht noch nicht im Regal.
	imZulauf := titel(t, "Nur im Zulauf "+marke)
	bestellt := "bestellt"
	exemplar(t, imZulauf, "B-BST-Z"+suffix, false, false, &bestellt)

	verliehen := titel(t, "Alles verliehen "+marke)
	exID := exemplar(t, verliehen, "B-BST-V"+suffix, true, false, nil)

	frei := titel(t, "Zwei frei eines weg "+marke)
	exemplar(t, frei, "B-BST-F1"+suffix, true, false, nil)
	exemplar(t, frei, "B-BST-F2"+suffix, true, false, nil)
	exemplar(t, frei, "B-BST-F3"+suffix, false, true, nil)

	var schuelerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Bestand', 'Leser', '07A', 2031) RETURNING id`, "S-BST-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, now(), now() + interval '14 days')`, exID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE schueler_id = $1`,
			`DELETE FROM schueler WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, schuelerID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE barcode_id LIKE $1`, "B-BST-%"+suffix); err != nil {
			t.Errorf("Aufräumen Exemplare: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE autor = $1`, marke); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
	})

	// Seit dem 22.09.2026 (Antwort der Schule, docs/OFFEN.md 9.4) zeigt keine der beiden
	// Türen einen Titel ohne ein nicht ausgesondertes Exemplar. Der Zulauf zählt als
	// vorhanden: Die Bücher kommen. Rot gesehen am Rückbau des Prädikats.
	erwartet := map[string][2]int{
		"Nur im Zulauf":       {0, 0}, // bestellt heißt: noch nicht im Regal — der Titel bleibt sichtbar
		"Alles verliehen":     {1, 0},
		"Zwei frei eines weg": {2, 2},
	}
	versteckt := []string{"Ohne Exemplar", "Nur ausgesondert"}

	// BEIDE Türen prüfen: Die Theke sucht über SearchTitlesFuzzy, der
	// Katalog/Aktionspfad über SearchTitles. Eine der beiden zu vergessen wäre genau
	// die Art Lücke, um die es hier geht.
	pruefe := func(t *testing.T, tuer string, zeilen []BookTitle) {
		t.Helper()
		gefunden := 0
		for _, z := range zeilen {
			for _, name := range versteckt {
				if strings.HasPrefix(z.Titel, name) {
					t.Errorf("%s: %q steht im Katalog, hat aber kein Exemplar", tuer, z.Titel)
				}
			}
			for name, soll := range erwartet {
				if !strings.HasPrefix(z.Titel, name) {
					continue
				}
				gefunden++
				if z.Bestand == nil || z.Verfuegbar == nil {
					t.Errorf("%s/%s: keine Bestandszahlen — die Trefferliste kann dann "+
						"nichts über den Bestand sagen", tuer, name)
					continue
				}
				if *z.Bestand != soll[0] || *z.Verfuegbar != soll[1] {
					t.Errorf("%s/%s: Bestand %d, verfügbar %d — erwartet %d und %d",
						tuer, name, *z.Bestand, *z.Verfuegbar, soll[0], soll[1])
				}
			}
		}
		if gefunden != len(erwartet) {
			t.Errorf("%s: %d von %d Titeln gefunden", tuer, gefunden, len(erwartet))
		}
	}

	repo := NewBookRepository(pool)
	zeilen, err := repo.SearchTitles(ctx, marke)
	if err != nil {
		t.Fatalf("SearchTitles: %v", err)
	}
	pruefe(t, "SearchTitles", zeilen)

	fuzzy, _, err := repo.SearchTitlesFuzzy(ctx, marke, 50)
	if err != nil {
		t.Fatalf("SearchTitlesFuzzy: %v", err)
	}
	pruefe(t, "SearchTitlesFuzzy", fuzzy)
}
