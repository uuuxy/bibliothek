package inventur

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/repository"
)

// Die drei Wege, auf denen ein Titel entsteht (Anlegen an der Maske, Listenimport im Stapel und
// im Einzel-Rückfall), führen ihre Werte als nummerierte Parameter. Jedes Feld trägt hier einen
// eigenen Wert und muss in seiner Spalte ankommen: Ein verrutschter Parameter schriebe den
// Verlag in den Untertitel, ohne dass eine Anweisung scheitert.
func TestTitelSchreiber_JedesFeldInSeinerSpalte(t *testing.T) {
	p := newGenanntProbe(t)
	ctx := context.Background()
	abfragen := repository.NewTitelAbfragen(p.pool)

	fach := genanntVorsatz + "Fach Schreiber"
	preis := 19.9
	gezaehlt := "2026-09-30"
	buch := func(nr int) repository.KatalogTitel {
		return repository.KatalogTitel{
			ISBN:                    fmt.Sprintf("97899966%05d", nr),
			Titel:                   fmt.Sprintf("%sSchreiber %d", genanntVorsatz, nr),
			Autor:                   "Eine Autorin",
			CoverURL:                "/covers/schreiber.webp",
			Fach:                    fach,
			Zweig:                   "R",
			ZuletztGezaehlt:         &gezaehlt,
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
		schreibe       func(repository.KatalogTitel) error
	}{
		{"LegeTitelAn", "true", func(b repository.KatalogTitel) error {
			_, err := abfragen.LegeTitelAn(ctx, b)
			return err
		}},
		{"schreibeTitel (Einzel-Rückfall)", "false", func(b repository.KatalogTitel) error {
			_, err := abfragen.SchreibeTitelAusListe(ctx, b)
			return err
		}},
		{"SchreibeTitelStapel", "false", func(b repository.KatalogTitel) error {
			_, err := abfragen.SchreibeTitelStapel(ctx, []repository.KatalogTitel{b})
			return err
		}},
	}
	for nr, weg := range wege {
		b := buch(nr + 1)
		if err := weg.schreibe(b); err != nil {
			t.Fatalf("%s: %v", weg.name, err)
		}
		var id string
		if err := p.pool.QueryRow(ctx, `SELECT id::text FROM buecher_titel WHERE titel = $1`, b.Titel).Scan(&id); err != nil {
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
