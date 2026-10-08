package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// POST /api/bestellungen weist einen Warenkorb ab, den es nicht buchen kann. Gemessen am
// 08.10.2026 führte kein Go-Test die Zuordnung dieser Fehler aus (OFFEN.md 5.10).
//
// Über den ganzen Router mit Sitzung. Jede Ablehnung ist eine Auskunft mit einem Satz, an
// dem sich der Warenkorb berichtigen lässt, und sie hat nichts gebucht: keine Bestellung
// und kein Exemplar, auch nicht für die Positionen vor der abgelehnten.
func TestBestellung_AbgelehnterWarenkorbBuchtNichts(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	_, rufe := protokollWelt(t, pool)
	lieferant := haendler(t, pool, "Abgelehnt", false)
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant)
		aufraeumen(t, pool, `DELETE FROM lieferanten WHERE id = $1`, lieferant)
	})
	titel := titelMitMeldebestand(t, pool, "Titel im Warenkorb", 0)

	position := func(titelID string, menge int) string {
		return fmt.Sprintf(`{"titel_id":%q,"menge":%d,"preis":9.5,"generate_barcodes":true}`, titelID, menge)
	}
	bestelle := func(t *testing.T, lieferantID string, positionen ...string) *httptest.ResponseRecorder {
		t.Helper()
		return rufe(t, http.MethodPost, "/api/bestellungen", fmt.Sprintf(
			`{"supplier_id":%q,"mittel":"land","items":[%s]}`, lieferantID, strings.Join(positionen, ",")))
	}
	gebucht := func(t *testing.T) (bestellungen, exemplare int) {
		t.Helper()
		return zaehleZeilen(t, pool, `SELECT count(*) FROM bestellungen_verlauf`),
			zaehleZeilen(t, pool, `SELECT count(*) FROM buecher_exemplare`)
	}
	// abgelehnt schickt den Warenkorb und prüft Status, Meldung und dass nichts gebucht ist.
	abgelehnt := func(t *testing.T, status int, steckt, lieferantID string, positionen ...string) {
		t.Helper()
		bestellungenVorher, exemplareVorher := gebucht(t)
		rec := bestelle(t, lieferantID, positionen...)
		if rec.Code != status {
			t.Errorf("Status %d, erwartet %d: %s", rec.Code, status, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), steckt) {
			t.Errorf("die Meldung nennt %q nicht: %s", steckt, rec.Body.String())
		}
		if b, e := gebucht(t); b != bestellungenVorher || e != exemplareVorher {
			t.Errorf("die Ablehnung hat gebucht: Bestellungen %d → %d, Exemplare %d → %d",
				bestellungenVorher, b, exemplareVorher, e)
		}
	}

	t.Run("Lieferant inzwischen gelöscht: 404", func(t *testing.T) {
		abgelehnt(t, http.StatusNotFound, "Lieferant", uuid.NewString(), position(titel, 2))
	})

	// Der Warenkorb steht nur im Browser. Löscht ein anderer Platz inzwischen einen Titel
	// daraus, muss die Meldung sagen, welche Position es ist — sonst lässt sich der
	// Warenkorb nicht berichtigen.
	t.Run("Titel inzwischen gelöscht: 404 mit der Position", func(t *testing.T) {
		abgelehnt(t, http.StatusNotFound, "Position 2", lieferant,
			position(titel, 2), position(uuid.NewString(), 1))
	})

	t.Run("Position ohne Titel: 400", func(t *testing.T) {
		abgelehnt(t, http.StatusBadRequest, "TitelID", lieferant, position("", 1))
	})

	for _, menge := range []int{0, -1, 201} {
		t.Run(fmt.Sprintf("Menge %d: 400 mit der Position", menge), func(t *testing.T) {
			abgelehnt(t, http.StatusBadRequest, "Position 2", lieferant,
				position(titel, 1), position(titel, menge))
		})
	}

	// Gegenprobe: Derselbe Warenkorb ohne die fehlerhafte Position wird gebucht. Ohne sie
	// bestünde der Test auch an einer Tür, die jede Bestellung abweist.
	t.Run("der berichtigte Warenkorb wird gebucht", func(t *testing.T) {
		_, exemplareVorher := gebucht(t)
		rec := bestelle(t, lieferant, position(titel, 2))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		if n := zaehleBestellungen(t, pool, lieferant); n != 1 {
			t.Errorf("%d Bestellungen beim Lieferanten, erwartet 1", n)
		}
		if _, e := gebucht(t); e != exemplareVorher+2 {
			t.Errorf("%d Exemplare angelegt, erwartet 2", e-exemplareVorher)
		}
	})
}
