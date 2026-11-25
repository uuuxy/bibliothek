package repository

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"slices"
	"testing"

	"bibliothek/internal/pgtest"
)

// titelsucheFaelle sind die geteilten Fälle der Titelsuche (titelsuche.faelle.json).
type titelsucheFaelle struct {
	Kuerzel map[string]string `json:"kuerzel"`
	Titel   []struct {
		ID            string   `json:"id"`
		Titel         string   `json:"titel"`
		Autor         string   `json:"autor"`
		Subject       string   `json:"subject"`
		ISBN          string   `json:"isbn"`
		Signatur      string   `json:"signatur"`
		IstLernmittel bool     `json:"ist_lernmittel"`
		JahrgangVon   *int     `json:"jahrgang_von"`
		JahrgangBis   *int     `json:"jahrgang_bis"`
		Schlagworte   []string `json:"schlagworte"`
	} `json:"titel"`
	Faelle []struct {
		Fall    string   `json:"fall"`
		Suche   string   `json:"suche"`
		Treffer []string `json:"treffer"`
	} `json:"faelle"`
}

func leseTitelsucheFaelle(t *testing.T) titelsucheFaelle {
	t.Helper()
	roh, err := os.ReadFile("../../frontend/src/inventur/lib/titelsuche.faelle.json")
	if err != nil {
		t.Fatalf("Fälle lesen: %v", err)
	}
	var f titelsucheFaelle
	if err := json.Unmarshal(roh, &f); err != nil {
		t.Fatalf("Fälle: %v", err)
	}
	if len(f.Faelle) < 15 || len(f.Titel) < 5 {
		t.Fatalf("nur %d Fälle und %d Titel gelesen", len(f.Faelle), len(f.Titel))
	}
	return f
}

// Die Kürzel am Server sind die der Fall-Liste und damit die des Browsers.
func TestTitelsuche_KuerzelWieImBrowser(t *testing.T) {
	f := leseTitelsucheFaelle(t)
	if !maps.Equal(titelKuerzel, f.Kuerzel) {
		t.Errorf("titelKuerzel = %v, die Fall-Liste nennt %v", titelKuerzel, f.Kuerzel)
	}
}

// Die Bedingung am Server findet zu jeder Suche der Fall-Liste dieselben Titel wie
// buecherSuchen im Browser (startseiten_api.test.js liest dieselbe Datei).
func TestTitelsuche_GemeinsameFaelle(t *testing.T) {
	f := leseTitelsucheFaelle(t)
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	kennung := map[string]string{}
	ids := make([]string, 0, len(f.Titel))
	for _, titel := range f.Titel {
		// Das Fach hängt an der Systematik (fk_titel_subject_systematik).
		if titel.Subject != "" {
			if _, err := tx.Exec(ctx, `INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
				SELECT 'TS-' || $1, $1 WHERE NOT EXISTS (SELECT 1 FROM systematik_kategorien WHERE bezeichnung = $1)`,
				titel.Subject); err != nil {
				t.Fatalf("Fach %s anlegen: %v", titel.Subject, err)
			}
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel
			(titel, autor, subject, isbn, signatur, ist_lernmittel, jahrgang_von, jahrgang_bis)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8)
			RETURNING id::text`, titel.Titel, titel.Autor, titel.Subject, titel.ISBN, titel.Signatur,
			titel.IstLernmittel, titel.JahrgangVon, titel.JahrgangBis).Scan(&id); err != nil {
			t.Fatalf("Titel %s anlegen: %v", titel.ID, err)
		}
		for _, wort := range titel.Schlagworte {
			if _, err := tx.Exec(ctx, `WITH sw AS (INSERT INTO schlagworte (wort) VALUES ($1) RETURNING id)
				INSERT INTO titel_schlagworte (titel_id, schlagwort_id) SELECT $2, id FROM sw`, wort, id); err != nil {
				t.Fatalf("Schlagwort %s anlegen: %v", wort, err)
			}
		}
		kennung[id] = titel.ID
		ids = append(ids, id)
	}

	for _, fall := range f.Faelle {
		bedingung, args := sqlTitelTrifft("t", fall.Suche, 2)
		zeilen, err := tx.Query(ctx, `SELECT t.id::text FROM buecher_titel t WHERE t.id = ANY($1::uuid[]) AND `+bedingung,
			append([]any{ids}, args...)...)
		if err != nil {
			t.Fatalf("%s: %v", fall.Fall, err)
		}
		treffer := []string{}
		for zeilen.Next() {
			var id string
			if err := zeilen.Scan(&id); err != nil {
				t.Fatalf("%s: %v", fall.Fall, err)
			}
			treffer = append(treffer, kennung[id])
		}
		zeilen.Close()
		if err := zeilen.Err(); err != nil {
			t.Fatalf("%s: %v", fall.Fall, err)
		}
		slices.Sort(treffer)
		erwartet := slices.Sorted(slices.Values(fall.Treffer))
		if !slices.Equal(treffer, erwartet) {
			t.Errorf("%s: Suche %q findet %v, erwartet %v", fall.Fall, fall.Suche, treffer, erwartet)
		}
	}
}
