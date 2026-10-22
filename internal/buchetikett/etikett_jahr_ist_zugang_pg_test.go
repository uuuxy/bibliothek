package buchetikett

import (
	"context"
	"strconv"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/schulzeit"
)

// „Ansch.J.“ auf dem Etikett ist das Jahr, in dem das Buch in den Bestand kam (zugang_am);
// erworben_am ist im Bestellweg der Bestelltag. Ein im Dezember bestelltes, im Januar
// geliefertes Buch trägt auf dem Etikett das Jahr, unter dem es im Zugangsbuch steht.
func TestEtikett_AnschaffungsjahrIstDasZugangsjahr(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp)
		VALUES ('Etikettenjahr-Band', 'Prüfer', 'Buch') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, titelID); err != nil {
			t.Errorf("Exemplare aufräumen: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Titel aufräumen: %v", err)
		}
	})

	heute := schulzeit.Jetzt()
	vorJahr := heute.AddDate(-1, 0, 0)

	// Wie im Bestellweg: Zeile im Zulauf vom Vorjahr, Wareneingang heute (Trigger 129).
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, erworben_am, bestellstatus)
		VALUES ($1, 'EJ-BESTELLT', false, $2, 'bestellt')`, titelID, vorJahr); err != nil {
		t.Fatalf("Exemplar bestellen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET bestellstatus = NULL, ist_ausleihbar = true
		WHERE barcode_id = 'EJ-BESTELLT'`); err != nil {
		t.Fatalf("Wareneingang: %v", err)
	}
	// Gegenprobe: Altbestand vom Vorjahr ohne Bestellung — sein Jahr bleibt das Vorjahr.
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, erworben_am)
		VALUES ($1, 'EJ-ALT', true, $2)`, titelID, vorJahr); err != nil {
		t.Fatalf("Altbestand anlegen: %v", err)
	}

	// Wie ein Druckauftrag des Druck-Centers: nur die Nummern, das Jahr trägt der Server nach.
	auftrag := []Etikett{{BarcodeID: "EJ-BESTELLT"}, {BarcodeID: "EJ-ALT"}}
	ErgaenzeServerfelder(ctx, pool, auftrag)
	jahr := map[string]string{}
	for _, e := range auftrag {
		jahr[e.BarcodeID] = e.AnschaffungsJahr
	}
	if got, want := jahr["EJ-BESTELLT"], strconv.Itoa(heute.Year()); got != want {
		t.Errorf("Ansch.J. des gelieferten Exemplars = %q, erwartet %q (Jahr des Zugangs, nicht des Bestelltags)", got, want)
	}
	if got, want := jahr["EJ-ALT"], strconv.Itoa(vorJahr.Year()); got != want {
		t.Errorf("Ansch.J. des Altbestands = %q, erwartet %q", got, want)
	}
}
