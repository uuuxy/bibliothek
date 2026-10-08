package inventur

import (
	"context"
	"fmt"
	"testing"
)

// Die drei Wege, auf denen ein Titel entsteht (Anlegen an der Maske, Listenimport im Stapel und
// im Einzel-Rückfall), führen ihre Werte als nummerierte Parameter. Jedes Feld trägt hier einen
// eigenen Wert und muss in seiner Spalte ankommen: Ein verrutschter Parameter schriebe den
// Verlag in den Untertitel, ohne dass eine Anweisung scheitert.
func TestTitelSchreiber_JedesFeldInSeinerSpalte(t *testing.T) {
	p := neueGenanntProbe(t)
	ctx := context.Background()
	repo := NewBookRepository(p.pool)

	fach := genanntVorsatz + "Fach Schreiber"
	preis := 19.9
	gezaehlt := "2026-09-30"
	buch := func(nr int) Book {
		return Book{
			ISBN:                    fmt.Sprintf("97899966%05d", nr),
			Title:                   fmt.Sprintf("%sSchreiber %d", genanntVorsatz, nr),
			Author:                  "Eine Autorin",
			CoverURL:                "/covers/schreiber.webp",
			Subject:                 fach,
			Track:                   "R",
			LastCounted:             &gezaehlt,
			Medientyp:               "DVD",
			JahrgangVon:             7,
			JahrgangBis:             9,
			Mehrjahresband:          true,
			Untertitel:              "Ein Untertitel",
			Verlag:                  "Ein Verlag",
			Erscheinungsjahr:        2021,
			Signatur:                "Bio 7",
			IstLernmittel:           true,
			Auflage:                 "2. Aufl.",
			Listenpreis:             &preis,
			ErweiterteEigenschaften: map[string]any{"probe": "schreiber"},
		}
	}
	soll := map[string]string{
		"autor":                    `"Eine Autorin"`,
		"cover_url":                `"/covers/schreiber.webp"`,
		"subject":                  `"` + fach + `"`,
		"track":                    `"R"`,
		"last_counted":             `"2026-09-30"`,
		"medientyp":                `"DVD"`,
		"jahrgang_von":             "7",
		"jahrgang_bis":             "9",
		"untertitel":               `"Ein Untertitel"`,
		"verlag":                   `"Ein Verlag"`,
		"erscheinungsjahr":         "2021",
		"signatur":                 `"Bio 7"`,
		"ist_lernmittel":           "true",
		"auflage":                  `"2. Aufl."`,
		"listenpreis":              "19.90",
		"erweiterte_eigenschaften": `{"probe": "schreiber"}`,
	}

	wege := []struct {
		name string
		// Nur das Anlegen an der Maske schreibt das Mehrjahresband; die Importe kennen es nicht.
		mehrjahresband string
		schreibe       func(Book) error
	}{
		{"CreateBook", "true", func(b Book) error {
			_, err := repo.CreateBook(ctx, b)
			return err
		}},
		{"UpsertBook (Einzel-Rückfall)", "false", func(b Book) error {
			_, err := repo.UpsertBook(ctx, b)
			return err
		}},
		{"UpsertBooksBatch", "false", func(b Book) error {
			_, err := repo.UpsertBooksBatch(ctx, []Book{b})
			return err
		}},
	}
	for nr, weg := range wege {
		b := buch(nr + 1)
		if err := weg.schreibe(b); err != nil {
			t.Fatalf("%s: %v", weg.name, err)
		}
		var id string
		if err := p.pool.QueryRow(ctx, `SELECT id::text FROM buecher_titel WHERE titel = $1`, b.Title).Scan(&id); err != nil {
			t.Fatalf("%s: Titel nicht gefunden: %v", weg.name, err)
		}
		zeile := p.zeile(id)
		erwartet := map[string]string{"isbn": `"` + b.ISBN + `"`, "mehrjahresband": weg.mehrjahresband}
		for spalte, wert := range soll {
			erwartet[spalte] = wert
		}
		for spalte, wert := range erwartet {
			if zeile[spalte] != wert {
				t.Errorf("%s: Spalte %s trägt %s, erwartet %s", weg.name, spalte, zeile[spalte], wert)
			}
		}
	}
}
