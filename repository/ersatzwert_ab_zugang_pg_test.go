package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/schulzeit"
)

// Der Zugang eines Exemplars ist der Tag der Lieferung (zugang_am), nicht der Tag, an dem die
// Zeile beim Bestellen entstand (erworben_am). Die Staffel des Schadensersatzes rechnet deshalb
// wie das Zugangsbuch ab zugang_am.
//
// Der Regelfall der Lernmittel ist die Lücke: bestellt im Juni für das kommende Schuljahr,
// geliefert im August. Am Bestelltag gerechnet wäre das Buch am Tag seiner Ausgabe schon ein
// Schuljahr alt, und jede Forderung läge eine Stufe der Staffel zu niedrig.
func TestErsatzwert_ZaehltAbZugangNichtAbBestelltag(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewBescheidAbfragen(pool)

	// Der Bestelltag liegt im vorigen Schuljahr, der Zugang (heute) im laufenden.
	bestelltAm := SchuljahrBeginn(schulzeit.Jetzt()).AddDate(0, 0, -20)

	var schuelerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Zugang', 'Staffel', '07B', 2032) RETURNING id
	`, "SZS-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Zugangs-Testband', 'Prüfer', 'Buch', true) RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, q := range []string{
			`DELETE FROM schadensfaelle WHERE schueler_id = $1`,
			`DELETE FROM ausleihen WHERE schueler_id = $1`,
		} {
			if _, err := pool.Exec(auf, q, schuelerID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Exemplare: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM schueler WHERE id = $1`, schuelerID); err != nil {
			t.Errorf("Aufräumen Schüler: %v", err)
		}
	})

	// bestelltUndGeliefert legt ein Exemplar so an, wie es der Bestellweg tut: Die Zeile
	// entsteht am Bestelltag im Zulauf, der Wareneingang räumt den Bestellstatus, und der
	// Trigger stempelt dabei den heutigen Tag als Zugang.
	bestelltUndGeliefert := func(t *testing.T, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis, erworben_am, bestellstatus)
			VALUES ($1, $2, false, 20, $3, 'bestellt') RETURNING id
		`, titelID, barcode, bestelltAm).Scan(&id); err != nil {
			t.Fatalf("Exemplar bestellen: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET bestellstatus = NULL, ist_ausleihbar = true WHERE id = $1`, id); err != nil {
			t.Fatalf("Wareneingang: %v", err)
		}
		var zugang, erworben time.Time
		if err := pool.QueryRow(ctx, `SELECT zugang_am, erworben_am FROM buecher_exemplare WHERE id = $1`, id).
			Scan(&zugang, &erworben); err != nil {
			t.Fatalf("Datum lesen: %v", err)
		}
		if schuljahrVon(zugang) == schuljahrVon(erworben) {
			t.Fatalf("Aufbau: Zugang %s und Bestelltag %s liegen im selben Schuljahr — der Fall misst nichts",
				zugang.Format("2006-01-02"), erworben.Format("2006-01-02"))
		}
		return id
	}

	exemplarID := bestelltUndGeliefert(t, "B-ZS"+suffix+"-1")

	t.Run("Betragsvorschlag: null Schuljahre im Bestand", func(t *testing.T) {
		g, err := repo.LadeGroessenFuerExemplar(ctx, exemplarID)
		if err != nil {
			t.Fatalf("LadeGroessenFuerExemplar: %v", err)
		}
		if g.SchuljahreImBestand != 0 {
			t.Fatalf("SchuljahreImBestand = %d, erwartet 0 — gerechnet ab dem Bestelltag statt ab dem Zugang", g.SchuljahreImBestand)
		}
	})

	t.Run("Überfällige Ausleihe: null Schuljahre im Bestand", func(t *testing.T) {
		jetzt := schulzeit.Jetzt()
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
			VALUES ($1, $2, $3, $4)
		`, exemplarID, schuelerID, jetzt.AddDate(0, 0, -60), jetzt.AddDate(0, 0, -30)); err != nil {
			t.Fatalf("Ausleihe anlegen: %v", err)
		}
		liste, err := repo.LadeUeberfaelligeAusleihen(ctx, schuelerID)
		if err != nil {
			t.Fatalf("LadeUeberfaelligeAusleihen: %v", err)
		}
		if len(liste) != 1 {
			t.Fatalf("%d überfällige Ausleihen, erwartet 1", len(liste))
		}
		if liste[0].SchuljahreImBestand != 0 {
			t.Fatalf("SchuljahreImBestand = %d, erwartet 0", liste[0].SchuljahreImBestand)
		}
	})

	t.Run("Offene Forderung: null Schuljahre im Bestand", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
			VALUES ($1, $2, 'Einband gerissen', 0, 'beschaedigt')
		`, exemplarID, schuelerID); err != nil {
			t.Fatalf("Schadensfall anlegen: %v", err)
		}
		liste, err := repo.LadeOffeneForderungen(ctx, schuelerID)
		if err != nil {
			t.Fatalf("LadeOffeneForderungen: %v", err)
		}
		if len(liste) != 1 {
			t.Fatalf("%d offene Forderungen, erwartet 1", len(liste))
		}
		if liste[0].SchuljahreImBestand != 0 {
			t.Fatalf("SchuljahreImBestand = %d, erwartet 0", liste[0].SchuljahreImBestand)
		}
	})

	t.Run("Gegenprobe: Altbestand ohne Bestellung zählt wie bisher", func(t *testing.T) {
		// Angelegt außerhalb des Bestellwegs: Der Trigger übernimmt erworben_am als Zugang,
		// die Zahl darf sich durch die Umstellung nicht bewegen.
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis, erworben_am)
			VALUES ($1, $2, true, 20, $3) RETURNING id
		`, titelID, "B-ZS"+suffix+"-2", bestelltAm).Scan(&id); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		g, err := repo.LadeGroessenFuerExemplar(ctx, id)
		if err != nil {
			t.Fatalf("LadeGroessenFuerExemplar: %v", err)
		}
		if g.SchuljahreImBestand != 1 {
			t.Fatalf("SchuljahreImBestand = %d, erwartet 1", g.SchuljahreImBestand)
		}
	})
}
