package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Der Reiter „Schadensersatz" zeigt die Stufe zwischen Mahnliste und Bescheid: Kinder,
// deren Forderung noch auf keinem Brief steht. Vier Fälle grenzen die Zeile ab —
// bezahlt, storniert, schon auf einem Bescheid und im Papierkorb bleiben draußen.
func TestAusstehend_NurOffeneForderungenOhneBescheid(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewBescheidRepository(pool)

	var titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Ausstehend-Testband', 'Prüfer', 'Buch', true) RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	schueler := []string{}
	kind := func(t *testing.T, name string, geloescht bool) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
			VALUES ($1, $2, 'Ausstehend', '07B', 2031, CASE WHEN $3 THEN now() END) RETURNING id
		`, "SAU-"+name+"-"+suffix, name, geloescht).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		schueler = append(schueler, id)
		return id
	}
	forderung := func(t *testing.T, schuelerID string, betrag float64, bezahlt bool, storno bool) string {
		t.Helper()
		var exemplarID, id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
			VALUES ($1, $2, true, 20.00) RETURNING id
		`, titelID, fmt.Sprintf("B-AU-%s-%d", suffix, len(schueler)*10+int(betrag))).Scan(&exemplarID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art, ist_bezahlt,
			                            storniert_am, stornierungsgrund)
			VALUES ($1, $2, 'Test', $3, 'nicht_zurueckgegeben', $4,
			        CASE WHEN $5 THEN now() END, CASE WHEN $5 THEN 'Test' END) RETURNING id
		`, exemplarID, schuelerID, betrag, bezahlt || storno, storno).Scan(&id); err != nil {
			t.Fatalf("Forderung anlegen: %v", err)
		}
		return id
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, id := range schueler {
			for _, sql := range []string{
				`DELETE FROM schadensfaelle WHERE schueler_id = $1`,
				`DELETE FROM schadensersatz_bescheide WHERE schueler_id = $1`,
				`DELETE FROM schueler WHERE id = $1`,
			} {
				if _, err := pool.Exec(auf, sql, id); err != nil {
					t.Errorf("Aufräumen: %v", err)
				}
			}
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Exemplare: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
	})

	// Das eine Kind, das in der Liste stehen muss: zwei offene Forderungen, eine
	// bezahlte daneben — gezählt und summiert werden nur die offenen.
	offen := kind(t, "Offen", false)
	forderung(t, offen, 12, false, false)
	forderung(t, offen, 8, false, false)
	forderung(t, offen, 99, true, false)

	// Die drei, die NICHT erscheinen dürfen.
	storniert := kind(t, "Storno", false)
	forderung(t, storniert, 5, false, true)
	geloescht := kind(t, "Papierkorb", true)
	forderung(t, geloescht, 5, false, false)
	mitBrief := kind(t, "Brief", false)
	aufBrief := forderung(t, mitBrief, 5, false, false)
	if _, err := pool.Exec(ctx, `
		WITH b AS (
			INSERT INTO schadensersatz_bescheide
				(schueler_id, mittel, kassenjahr, laufende_nr, referenznummer, frist_bis,
				 gesamtbetrag, empfaenger_snapshot)
			VALUES ($1, 'land', 2099, 9001, 'AUS-'||$3, CURRENT_DATE + 28, 5, '{}'::jsonb)
			RETURNING id
		)
		UPDATE schadensfaelle SET bescheid_id = (SELECT id FROM b) WHERE id = $2
	`, mitBrief, aufBrief, suffix); err != nil {
		t.Fatalf("Bescheid zuordnen: %v", err)
	}

	liste, err := repo.Ausstehend(ctx)
	if err != nil {
		t.Fatalf("Ausstehend: %v", err)
	}
	je := map[string]ForderungOhneBescheid{}
	for _, z := range liste {
		je[z.SchuelerID] = z
	}
	z, ok := je[offen]
	if !ok {
		t.Fatalf("das Kind mit offenen Forderungen fehlt in der Liste")
	}
	if z.Anzahl != 2 || z.Summe != 20 || !z.Lernmittel || z.Klasse != "07B" {
		t.Errorf("Zeile falsch: %+v (erwartet 2 Forderungen, 20,00 €, Lernmittel, 07B)", z)
	}
	for name, id := range map[string]string{"storniert": storniert, "gelöscht": geloescht, "auf Bescheid": mitBrief} {
		if _, da := je[id]; da {
			t.Errorf("Kind „%s\" steht in der Liste, gehört aber nicht hinein", name)
		}
	}
}
