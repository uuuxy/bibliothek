package repository

import (
	"context"
	"testing"
)

// Die Suche der Theke und die Kataloge (ohne Anmeldung und „Mein Portal") vergleichen den
// Suchtext neben dem Volltext als Teilstring. Die Datenbank speichert Titeltexte mit einem
// Leerzeichen zwischen den Wörtern (Migration 160); der Suchtext geht in dieselbe Form
// (TiteltextNormalform). Die Suchtexte enden mitten im Wort, damit der Volltext nicht
// aushilft: Er trifft nur ganze Wörter.
func TestSuche_LeerraumInFolgeTrenntSuchtextUndTitelNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	const gespeichert = "La Peste"
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor) VALUES ($1, $2) RETURNING id::text`,
		"La  Peste", "Camus,"+geschuetzt+"Albert").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-LEERRAUM-1')`, id); err != nil {
		t.Fatal(err)
	}

	for _, suchtext := range []string{
		"La Pes",                    // getippt, wie man tippt
		"La  Pes",                   // wie der Titel in Littera steht
		" la" + geschuetzt + "pes ", // aus einer Liste kopiert
		"Camus, Alb",
		"Camus," + tabulator + "Alb",
	} {
		theke, err := repo.SearchTitles(ctx, suchtext)
		if err != nil {
			t.Fatalf("SearchTitles(%+q): %v", suchtext, err)
		}
		if len(theke) != 1 || theke[0].Titel != gespeichert {
			t.Errorf("SearchTitles(%+q): %d Treffer, erwartet den Titel %q", suchtext, len(theke), gespeichert)
		}
		katalog, gesamt, err := SucheImKatalog(ctx, pool, KatalogSuche{
			Sichtbar: KollegiumSichtbar("bt"), Suchtext: suchtext, Grenze: 10,
		})
		if err != nil {
			t.Fatalf("SucheImKatalog(%+q): %v", suchtext, err)
		}
		if len(katalog) != 1 || gesamt != 1 || katalog[0].Titel != gespeichert {
			t.Errorf("SucheImKatalog(%+q): %d Treffer (gesamt %d), erwartet den Titel %q", suchtext, len(katalog), gesamt, gespeichert)
		}
	}

	// Gegenprobe: Der Leerraum fällt nicht weg, zwei Wörter bleiben zwei.
	if treffer, err := repo.SearchTitles(ctx, "LaPes"); err != nil || len(treffer) != 0 {
		t.Errorf("SearchTitles ohne Leerzeichen: %d Treffer, %v — erwartet keinen", len(treffer), err)
	}
	// Ein Suchtext nur aus Leerraum ist kein Suchtext: Der Katalog antwortet leer.
	if treffer, gesamt, err := SucheImKatalog(ctx, pool, KatalogSuche{
		Sichtbar: KollegiumSichtbar("bt"), Suchtext: " " + geschuetzt + " ", Grenze: 10,
	}); err != nil || len(treffer) != 0 || gesamt != 0 {
		t.Errorf("SucheImKatalog nur mit Leerraum: %d Treffer (gesamt %d), %v — erwartet keinen", len(treffer), gesamt, err)
	}
}
