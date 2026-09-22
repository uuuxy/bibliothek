package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Antwort der Schule vom 22.09.2026 auf Punkt 4 des Protokolls (docs/OFFEN.md 9.4): Ein
// Titel ohne (verliehene oder verfügbare) Exemplare erscheint nicht im Katalog. GET
// /api/books ist der Katalog des Portals UND die Liste der Verwaltung; die Verwaltung
// erreicht die versteckten Titel über die Aufräumsicht (bestand=ohne) derselben Tür.
// Beide Sichten sind EIN Prädikat (repository.SQLTitelHatExemplar) mit und ohne NOT —
// der Zulauf zählt als vorhanden, ein ausgesondertes Exemplar nicht.
// Rot gesehen am Rückbau des Prädikats.
func TestListBooks_TitelOhneExemplarNurInDerAufraeumsicht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	marke := "Sicht-" + t.Name()

	titel := func(name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel, autor) VALUES ($1, $2) RETURNING id`, name, marke).Scan(&id); err != nil {
			t.Fatalf("Titel %q anlegen: %v", name, err)
		}
		return id
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE autor = $1`, marke); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})

	ohne := titel("Ohne Exemplar")
	ausgesondert := titel("Nur ausgesondert")
	zulauf := titel("Nur im Zulauf")
	mit := titel("Mit Exemplar")
	for _, e := range []struct {
		titelID, barcode, sql string
	}{
		{mit, "B-SICHT-M", `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`},
		{ausgesondert, "B-SICHT-A", `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
			VALUES ($1, $2, false, true, 'AUSSORTIERT')`},
		{zulauf, "B-SICHT-Z", `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
			VALUES ($1, $2, false, 'bestellt')`},
	} {
		if _, err := pool.Exec(ctx, e.sql, e.titelID, e.barcode+marke); err != nil {
			t.Fatalf("Exemplar %s: %v", e.barcode, err)
		}
	}

	ids := func(liste []Book) map[string]bool {
		m := map[string]bool{}
		for _, b := range liste {
			m[b.ID] = true
		}
		return m
	}
	pruefe := func(t *testing.T, sicht string, liste map[string]bool, drin, draussen map[string]string) {
		t.Helper()
		for name, id := range drin {
			if !liste[id] {
				t.Errorf("%s: %q fehlt", sicht, name)
			}
		}
		for name, id := range draussen {
			if liste[id] {
				t.Errorf("%s: %q steht in der Liste, gehört aber nicht hinein", sicht, name)
			}
		}
	}

	katalog, err := repo.ListBooks(ctx, "", nil, marke, false)
	if err != nil {
		t.Fatalf("Katalog: %v", err)
	}
	pruefe(t, "Katalog", ids(katalog),
		map[string]string{"Mit Exemplar": mit, "Nur im Zulauf": zulauf},
		map[string]string{"Ohne Exemplar": ohne, "Nur ausgesondert": ausgesondert})

	aufraeumen, err := repo.ListBooks(ctx, "", nil, marke, true)
	if err != nil {
		t.Fatalf("Aufräumsicht: %v", err)
	}
	pruefe(t, "Aufräumsicht", ids(aufraeumen),
		map[string]string{"Ohne Exemplar": ohne, "Nur ausgesondert": ausgesondert},
		map[string]string{"Mit Exemplar": mit, "Nur im Zulauf": zulauf})
}
