package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Gate für den Jahrgangsfilter der Leserdatei (OFFEN.md 9.5, Protokoll 5 und 6).
//
// Der teure Fall steht ganz unten: die OBERSTUFE. An dieser Schule heißt der elfte
// Jahrgang „ET", und ein Filter, der den Jahrgang aus führenden Ziffern liest, findet
// dort nichts — lautlos, denn eine leere Liste sieht aus wie „niemand in diesem
// Jahrgang". Genau so rechnet KlassenMitSchuelern für die Sortierung des LMF-Plans
// (`substring(klasse from '^\d+')`, sonst 99); als Filter wäre dieselbe Zeile falsch.
//
// Am echten Postgres, weil geprüft wird, was aus der Kombination aus Klassenbestand und
// Ableitung herausfällt — ein Mock würde genau die Frage wegdefinieren.
func TestJahrgangFilterFindetAuchDieOberstufe(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	lmfRepo := repository.NewLmfTerminRepository(pool)

	// Ein Kind je Klasse — KlassenMitSchuelern nennt nur Klassen, in denen jemand ist.
	klassen := map[string]string{"05F1": "Fuenf", "05G2": "Fuenfb", "10H1": "Zehn", "ET": "Elf", "13T": "Dreizehn"}
	angelegt := []string{}
	for klasse, name := range klassen {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, $2, 'Jahrgangstest', $3, 2035) RETURNING id
		`, fmt.Sprintf("SJG-%s-%s", suffix, klasse), name, klasse).Scan(&id); err != nil {
			t.Fatalf("Schüler in %s anlegen: %v", klasse, err)
		}
		angelegt = append(angelegt, id)
	}
	t.Cleanup(func() {
		for _, id := range angelegt {
			if _, err := pool.Exec(context.Background(), `DELETE FROM schueler WHERE id = $1`, id); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})

	// enthaelt sagt, ob die Klasse im Ergebnis steht.
	enthaelt := func(liste []string, klasse string) bool {
		for _, k := range liste {
			if k == klasse {
				return true
			}
		}
		return false
	}

	t.Run("Jahrgang 5 nennt beide fünften Klassen", func(t *testing.T) {
		got, leer, err := klassenFilter(ctx, lmfRepo, "", "5")
		if err != nil {
			t.Fatalf("klassenFilter: %v", err)
		}
		if leer {
			t.Fatal("leer = true, obwohl zwei fünfte Klassen besetzt sind")
		}
		for _, k := range []string{"05F1", "05G2"} {
			if !enthaelt(got, k) {
				t.Errorf("%s fehlt im Jahrgang 5 (bekommen: %v)", k, got)
			}
		}
		if enthaelt(got, "10H1") {
			t.Errorf("10H1 steht im Jahrgang 5 (bekommen: %v)", got)
		}
	})

	t.Run("Jahrgang 11 findet ET — der Fall, an dem ein SQL-Filter scheitert", func(t *testing.T) {
		got, leer, err := klassenFilter(ctx, lmfRepo, "", "11")
		if err != nil {
			t.Fatalf("klassenFilter: %v", err)
		}
		if leer || !enthaelt(got, "ET") {
			t.Errorf("Jahrgang 11 liefert %v (leer=%v) — „ET\" IST der elfte Jahrgang an "+
				"dieser Schule; wer ihn über führende Ziffern sucht, findet ihn nie", got, leer)
		}
	})

	t.Run("Jahrgang 13 findet 13T", func(t *testing.T) {
		got, _, err := klassenFilter(ctx, lmfRepo, "", "13")
		if err != nil {
			t.Fatalf("klassenFilter: %v", err)
		}
		if !enthaelt(got, "13T") {
			t.Errorf("13T fehlt im Jahrgang 13 (bekommen: %v)", got)
		}
	})

	t.Run("eine Klasse schlägt den Jahrgang", func(t *testing.T) {
		got, _, err := klassenFilter(ctx, lmfRepo, "05F1", "10")
		if err != nil {
			t.Fatalf("klassenFilter: %v", err)
		}
		if len(got) != 1 || got[0] != "05F1" {
			t.Errorf("klassenFilter = %v, want [05F1] — die Klasse ist die genauere Angabe", got)
		}
	})

	t.Run("ohne Angabe wird nicht gefiltert", func(t *testing.T) {
		got, leer, err := klassenFilter(ctx, lmfRepo, "", "")
		if err != nil {
			t.Fatalf("klassenFilter: %v", err)
		}
		if got != nil || leer {
			t.Errorf("klassenFilter = %v, leer = %v — ohne Angabe zeigt die Liste alles", got, leer)
		}
	})

	t.Run("unbekannter Jahrgang zeigt NICHTS, nicht alles", func(t *testing.T) {
		// Beide Wege in denselben Zustand: ein Jahrgang außerhalb 1–13 und ein Vertipper.
		// Ohne diesen Zweig stünde bei „Jahrgang 99" die ganze Kartei — und das sähe für
		// den Benutzer aus, als hätte der Filter funktioniert.
		for _, eingabe := range []string{"99", "0", "fünf", "-3"} {
			got, leer, err := klassenFilter(ctx, lmfRepo, "", eingabe)
			if err != nil {
				t.Fatalf("klassenFilter(%q): %v", eingabe, err)
			}
			if !leer || got != nil {
				t.Errorf("klassenFilter(%q) = %v, leer = %v — want leere Liste", eingabe, got, leer)
			}
		}
	})
}
