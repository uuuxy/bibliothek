package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Ladenhüter sind Kandidaten zum Aussondern. Ein Titel, dessen Exemplare alle ausgesondert sind,
// gehört nicht mehr dazu, und ein Titel, dessen Exemplar seit über zwei Jahren verliehen ist (ein
// Mehrjahresband), ist in Gebrauch.
func TestLadenhueter_NurWasImRegalSteht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	marke := fmt.Sprintf("LH%d", time.Now().UnixNano())

	var leserID string
	if err := pool.QueryRow(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art, klasse, abgaenger_jahr)
		VALUES ($1, 'Lea', 'Langleih', 'schueler', '07A', 2031) RETURNING id`, "SCH-"+marke).Scan(&leserID); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}
	// titel legt einen Titel mit einem Exemplar an, das vor drei Jahren kam und vor drei Jahren
	// ausgeliehen wurde; offen heißt, es ist noch nicht zurück.
	titel := func(name string, offen bool) (titelID, exemplarID string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ($1, 'A', 'Buch') RETURNING id`,
			name+" "+marke).Scan(&titelID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, erstellt_am)
			VALUES ($1, $2, true, NOW() - interval '3 years') RETURNING id`, titelID, "B-"+name+"-"+marke).Scan(&exemplarID); err != nil {
			t.Fatal(err)
		}
		rueckgabe := "NOW() - interval '3 years' + interval '14 days'"
		if offen {
			rueckgabe = "NULL"
		}
		if _, err := pool.Exec(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, NOW() - interval '3 years', NOW() + interval '1 year', `+rueckgabe+`)`, exemplarID, leserID); err != nil {
			t.Fatal(err)
		}
		return titelID, exemplarID
	}
	ladenhueter, _ := titel("Ladenhueter", false)
	ausgesondert, ausgesondertExemplar := titel("Ausgesondert", false)
	inGebrauch, _ := titel("InGebrauch", true)
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false,
		aussonderung_grund = 'AUSSORTIERT' WHERE id = $1`, ausgesondertExemplar); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		if _, err := pool.Exec(auf, `DELETE FROM ausleihen WHERE schueler_id = $1`, leserID); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
		for _, id := range []string{ladenhueter, ausgesondert, inGebrauch} {
			if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, id); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
			if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
		if _, err := pool.Exec(auf, `DELETE FROM leser WHERE id = $1`, leserID); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})

	liste, err := LadeLadenhueter(ctx, pool, "", 100000)
	if err != nil {
		t.Fatal(err)
	}
	drin := map[string]bool{}
	for _, l := range liste {
		drin[l.ID] = true
	}
	if !drin[ladenhueter] {
		t.Error("der Titel, der seit drei Jahren im Regal steht, fehlt")
	}
	if drin[ausgesondert] {
		t.Error("ein Titel ohne Exemplar im Bestand steht auf der Liste")
	}
	if drin[inGebrauch] {
		t.Error("ein Titel, dessen Exemplar verliehen ist, steht auf der Liste")
	}
}
