package inventur

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Die Bestandsliste nennt je Exemplar Signatur, Schlagworte und Eigentum (PFLEGEKONZEPT.md 8):
// der Teil des Bestands, der sich bei einem Wechsel auf ein anderes Programm nicht neu
// erfassen lässt. An der echten Datenbank geprüft, weil die drei Spalten aus drei Tabellen
// kommen und das Eigentum der Regel mit drei Stufen folgt (repository.ExemplarTopfSQL).
func TestBestandslisteNenntSignaturSchlagworteUndEigentum(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	marke := fmt.Sprintf("EXPORT%d", time.Now().UnixNano())

	titel := func(name, signatur string, lernmittel bool) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_titel (titel, autor, medientyp, signatur, ist_lernmittel)
			VALUES ($1, $2, 'Buch', NULLIF($3, ''), $4) RETURNING id`,
			marke+" "+name, marke, signatur, lernmittel).Scan(&id); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		return id
	}
	// exemplar legt ein Exemplar an; eigentum und bestellung dürfen leer sein.
	exemplar := func(titelID, nummer, eigentum, bestellung string, ausgesondert bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare
				(titel_id, barcode_id, eigentum, eigentum_quelle, bestellung_id,
				 ist_ausgesondert, aussonderung_grund, ist_ausleihbar)
			VALUES ($1, $2, NULLIF($3, ''), CASE WHEN $3 = '' THEN NULL ELSE 'hand' END,
			        NULLIF($4, '')::uuid, $5, CASE WHEN $5 THEN 'AUSSORTIERT' END, NOT $5)`,
			titelID, "B-"+nummer+"-"+marke, eigentum, bestellung, ausgesondert); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", nummer, err)
		}
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE autor = $1)`,
			`DELETE FROM buecher_titel WHERE autor = $1`,
			`DELETE FROM bestellungen_verlauf WHERE lieferant_name = $1`,
			`DELETE FROM schlagworte WHERE wort LIKE $1 || '%'`,
		} {
			if _, err := pool.Exec(auf, sql, marke); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})

	var bestellungLand string
	if err := pool.QueryRow(ctx, `
		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
		VALUES ($1, 'export@example.invalid', $2) RETURNING id`,
		marke, repository.MittelLand).Scan(&bestellungLand); err != nil {
		t.Fatalf("Bestellung anlegen: %v", err)
	}

	schulbuch := titel("A Schulbuch", "LMF Ma 8", true)
	exemplar(schulbuch, "A1", "", "", false)
	exemplar(schulbuch, "A2", repository.MittelSchultraeger, "", false)
	exemplar(schulbuch, "A3", "", "", true)
	roman := titel("B Roman", "", false)
	exemplar(roman, "B1", "", bestellungLand, false)
	exemplar(roman, "B2", "", "", false)
	titel("C ohne Exemplar", "R 11", false)

	// Die Wörter tragen die Marke vorn, damit sie niemandem sonst gehören; gespeichert in
	// dieser Reihenfolge, erwartet alphabetisch ohne Rücksicht auf Groß- und Kleinschreibung.
	wortG, wortA := marke+" Geometrie", marke+" algebra"
	if _, err := repository.SetzeSchlagworte(ctx, pool, schulbuch, []string{wortG, wortA}); err != nil {
		t.Fatalf("Schlagworte setzen: %v", err)
	}

	var zeilen [][]string
	kopf := func() error { return nil }
	err := repo.StreamBooksForCSVExport(ctx, kopf, func(zeile []string) error {
		if strings.HasPrefix(zeile[0], marke) {
			zeilen = append(zeilen, slices.Clone(zeile))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Spalten: Titel, Barcode, Signatur, Schlagworte, Eigentum.
	var ist []string
	for _, z := range zeilen {
		if len(z) != 11 {
			t.Fatalf("Zeile hat %d Spalten, erwartet 11: %q", len(z), z)
		}
		ist = append(ist, strings.Join([]string{z[0], z[6], z[8], z[9], z[10]}, " ; "))
	}
	woerter := wortA + " | " + wortG
	soll := []string{
		// Ohne Vermerk und ohne Bestellung entscheidet der Titel: Lernmittel gehört dem Land.
		marke + " A Schulbuch ; B-A1-" + marke + " ; LMF Ma 8 ; " + woerter + " ; Land",
		// Was am Exemplar steht, geht dem Titel vor.
		marke + " A Schulbuch ; B-A2-" + marke + " ; LMF Ma 8 ; " + woerter + " ; Schulträger",
		// Der Topf der Bestellung geht dem Titel vor.
		marke + " B Roman ; B-B1-" + marke + " ;  ;  ; Land",
		marke + " B Roman ; B-B2-" + marke + " ;  ;  ; Schulträger",
		// Ein Titel ohne Exemplar hat seine Zeile und kein Eigentum.
		marke + " C ohne Exemplar ;  ; R 11 ;  ; ",
	}
	if !slices.Equal(ist, soll) {
		t.Errorf("Bestandsliste:\n ist:\n  %s\n soll:\n  %s", strings.Join(ist, "\n  "), strings.Join(soll, "\n  "))
	}
}
