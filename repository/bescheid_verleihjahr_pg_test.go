package repository

// Das Verleihjahr entscheidet, wie viel im Bescheid steht (#597).
//
// Die Staffel der Arbeitshilfe geht über VERLEIHJAHRE: 1. Jahr voller Preis, dann 80 %,
// 60 %, 40 %, 20 %, ab dem 6. Jahr 10 %. Ein Verleihjahr ist ein SCHULJAHR (Hessen:
// 1. August bis 31. Juli) — dieselbe Grenze, die Versetzung, Ausweis und Littera-Import
// benutzen.
//
// Bis zum 12.09.2026 lieferte die Abfrage zwei andere Größen: die Zahl der AUSLEIHEN
// (ein Buch, das in einem Schuljahr sechsmal hinausging, stand damit im 6. Verleihjahr
// → 10 % statt 100 %) und die Differenz der KALENDERJAHRE (ein im Februar gekauftes
// Buch blieb bis zum Dezember im 1. Verleihjahr, obwohl das zweite Schuljahr längst
// lief). Beide Zahlen stehen am Ende in einem Bescheid an Eltern.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/ersatzwert"
	"bibliothek/pkg/schulzeit"
)

func TestVerleihjahrZaehltSchuljahre(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewBescheidRepository(pool)

	schuljahrBeginn := SchuljahrBeginn(schulzeit.Jetzt())

	var schuelerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Verleih', 'Jahr', '08A', 2031) RETURNING id
	`, "SVJ-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Verleihjahr-Testband', 'Prüfer', 'Buch', true) RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM schadensfaelle WHERE schueler_id = $1`,
			`DELETE FROM ausleihen WHERE schueler_id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, schuelerID); err != nil {
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

	// forderung legt ein Exemplar mit Kaufdatum an, bucht `ausleihen` Ausleihen im
	// laufenden Schuljahr und meldet es als „nicht zurückgegeben".
	forderung := func(t *testing.T, name string, erworben time.Time, ausleihen int) string {
		t.Helper()
		var exemplarID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis, erworben_am)
			VALUES ($1, $2, true, 30.00, $3) RETURNING id
		`, titelID, "B-VJ-"+name+"-"+suffix, erworben).Scan(&exemplarID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		for i := 0; i < ausleihen; i++ {
			// Alle im laufenden Schuljahr, wenige Stunden auseinander: EIN Verleihjahr.
			zeitpunkt := schuljahrBeginn.Add(time.Duration(i+1) * time.Hour)
			if _, err := pool.Exec(ctx, `
				INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
				VALUES ($1, $2, $3::timestamptz, $3::timestamptz + interval '21 days', $3::timestamptz + interval '5 days')
			`, exemplarID, schuelerID, zeitpunkt); err != nil {
				t.Fatalf("Ausleihe %d buchen: %v", i+1, err)
			}
		}
		var schadensID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
			VALUES ($1, $2, $3, 0, 'nicht_zurueckgegeben') RETURNING id
		`, exemplarID, schuelerID, "Verleihjahr "+name).Scan(&schadensID); err != nil {
			t.Fatalf("Forderung anlegen: %v", err)
		}
		return schadensID
	}

	// Zwei Exemplare, einen Tag auseinander gekauft — und ein Schuljahr auseinander.
	// Dieses Paar ist der Kern: In JEDEM Monat des Jahres ist genau eine der beiden
	// Erwartungen mit Kalenderjahren nicht zu erreichen (im Aug–Dez die zweite, im
	// Jan–Jul die erste). Ein Gate, das nur eine davon prüfte, wäre ein halbes Jahr lang
	// grün, ohne etwas zu sehen.
	imNeuenJahr := forderung(t, "neu", schuljahrBeginn.AddDate(0, 0, 1), 0)
	imAltenJahr := forderung(t, "alt", schuljahrBeginn.AddDate(0, 0, -1), 0)
	// Vielfach verliehen, alles im selben Schuljahr: bleibt das 1. Verleihjahr.
	vielfach := forderung(t, "vielfach", schuljahrBeginn.AddDate(0, 0, 1), 4)

	forderungen, err := repo.OffeneForderungen(ctx, schuelerID)
	if err != nil {
		t.Fatalf("offene Forderungen lesen: %v", err)
	}
	verleihjahr := map[string]int{}
	for _, f := range forderungen {
		verleihjahr[f.SchadensfallID] = ersatzwert.Verleihjahr(f.SchuljahreMitAusleihe, f.SchuljahreImBestand)
	}

	faelle := []struct {
		name     string
		id       string
		erwartet int
		warum    string
	}{
		{"am Tag nach dem Schuljahresbeginn gekauft", imNeuenJahr, 1,
			"das Buch ist im laufenden Schuljahr in den Bestand gekommen"},
		{"am Tag vor dem Schuljahresbeginn gekauft", imAltenJahr, 2,
			"ein Schuljahr ist seither vergangen — auch wenn es dasselbe Kalenderjahr ist"},
		{"viermal im selben Schuljahr verliehen", vielfach, 1,
			"vier Ausleihen sind kein viertes Verleihjahr"},
	}
	for _, f := range faelle {
		if got := verleihjahr[f.id]; got != f.erwartet {
			t.Errorf("%s: Verleihjahr %d, erwartet %d — %s (Staffel: %d %% statt %d %%)",
				f.name, got, f.erwartet, f.warum,
				ersatzwert.Rechne(got, 30, 0).Prozent, ersatzwert.Rechne(f.erwartet, 30, 0).Prozent)
		}
	}
}
