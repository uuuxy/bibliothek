package repository

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/mitteltopf"
)

// Die Bestandsliste nennt je Exemplar Signatur, Schlagworte, Eigentum, Standort und die
// LMF-Nummer (PFLEGEKONZEPT.md 8): der Teil des Bestands, der sich bei einem Wechsel auf ein
// anderes Programm nicht neu erfassen lässt. An der echten Datenbank geprüft, weil die Spalten
// aus drei Tabellen kommen und das Eigentum der Regel mit drei Stufen folgt
// (ExemplarTopfSQL). Standort und LMF-Nummer stehen am Exemplar, nicht am Titel.
func TestBestandslisteNenntSignaturSchlagworteEigentumStandortUndLMFNummer(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	abfragen := NewTitelAbfragen(pool)
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
	// exemplar legt ein Exemplar an; eigentum, bestellung und standort dürfen leer sein.
	exemplar := func(titelID, nummer, eigentum, bestellung, standort string, ausgesondert bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare
				(titel_id, barcode_id, eigentum, eigentum_quelle, bestellung_id,
				 ist_ausgesondert, aussonderung_grund, ist_ausleihbar, standort)
			VALUES ($1, $2, NULLIF($3, ''), CASE WHEN $3 = '' THEN NULL ELSE 'hand' END,
			        NULLIF($4, '')::uuid, $5, CASE WHEN $5 THEN 'AUSSORTIERT' END, NOT $5,
			        NULLIF($6, ''))`,
			titelID, "B-"+nummer+"-"+marke, eigentum, bestellung, ausgesondert, standort); err != nil {
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
		marke, mitteltopf.Land).Scan(&bestellungLand); err != nil {
		t.Fatalf("Bestellung anlegen: %v", err)
	}

	// Zwei Exemplare desselben Schulbuchs an zwei Orten: Der Standort gehört zum Exemplar.
	schulbuch := titel("A Schulbuch", "LMF Ma 8", true)
	exemplar(schulbuch, "A1", "", "", "LMF", false)
	exemplar(schulbuch, "A2", mitteltopf.Schultraeger, "", "Bibliothek, Regal 3B", false)
	exemplar(schulbuch, "A3", "", "", "Keller", true)
	roman := titel("B Roman", "", false)
	exemplar(roman, "B1", "", bestellungLand, "", false)
	exemplar(roman, "B2", "", "", "", false)
	titel("C ohne Exemplar", "R 11", false)

	// A1 trägt noch das alte LMF-Etikett; seine Nummer bleibt neben der kurzen am Exemplar.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET lmf_barcode = $1 WHERE barcode_id = $2`,
		"LMF-"+marke, "B-A1-"+marke); err != nil {
		t.Fatalf("LMF-Nummer setzen: %v", err)
	}

	// Die Wörter tragen die Marke vorn, damit sie niemandem sonst gehören; gespeichert in
	// dieser Reihenfolge, erwartet alphabetisch ohne Rücksicht auf Groß- und Kleinschreibung.
	wortG, wortA := marke+" Geometrie", marke+" algebra"
	if _, err := SetzeSchlagworte(ctx, pool, schulbuch, []string{wortG, wortA}); err != nil {
		t.Fatalf("Schlagworte setzen: %v", err)
	}

	var zeilen [][]string
	kopf := func() error { return nil }
	err := abfragen.LadeTitelFuerCSVExport(ctx, kopf, func(zeile []string) error {
		if strings.HasPrefix(zeile[0], marke) {
			zeilen = append(zeilen, slices.Clone(zeile))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Spalten: Titel, Barcode, Signatur, Schlagworte, Eigentum, Standort, LMF-Nummer.
	var ist []string
	for _, z := range zeilen {
		if len(z) != 13 {
			t.Fatalf("Zeile hat %d Spalten, erwartet 13: %q", len(z), z)
		}
		ist = append(ist, strings.Join([]string{z[0], z[6], z[8], z[9], z[10], z[11], z[12]}, " ; "))
	}
	woerter := wortA + " | " + wortG
	soll := []string{
		// Ohne Vermerk und ohne Bestellung entscheidet der Titel: Lernmittel gehört dem Land.
		// Neben der kurzen Nummer steht die des alten Etiketts, das noch am Buch klebt.
		marke + " A Schulbuch ; B-A1-" + marke + " ; LMF Ma 8 ; " + woerter + " ; Land ; LMF ; LMF-" + marke,
		// Was am Exemplar steht, geht dem Titel vor. Das ausgesonderte dritte Exemplar fehlt.
		marke + " A Schulbuch ; B-A2-" + marke + " ; LMF Ma 8 ; " + woerter + " ; Schulträger ; Bibliothek, Regal 3B ; ",
		// Der Topf der Bestellung geht dem Titel vor. Ein Exemplar ohne Standort hat keinen.
		marke + " B Roman ; B-B1-" + marke + " ;  ;  ; Land ;  ; ",
		marke + " B Roman ; B-B2-" + marke + " ;  ;  ; Schulträger ;  ; ",
		// Ein Titel ohne Exemplar hat seine Zeile, kein Eigentum, keinen Standort und keine LMF-Nummer.
		marke + " C ohne Exemplar ;  ; R 11 ;  ;  ;  ; ",
	}
	if !slices.Equal(ist, soll) {
		t.Errorf("Bestandsliste:\n ist:\n  %s\n soll:\n  %s", strings.Join(ist, "\n  "), strings.Join(soll, "\n  "))
	}
}
