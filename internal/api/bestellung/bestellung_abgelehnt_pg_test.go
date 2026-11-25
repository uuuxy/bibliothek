package bestellung_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/routertest"
	"bibliothek/internal/testdaten"
)

// POST /api/bestellungen weist einen Warenkorb ab, den es nicht buchen kann.
//
// Über den ganzen Router mit Sitzung. Jede Ablehnung ist eine Auskunft mit einem Satz, an
// dem sich der Warenkorb berichtigen lässt, und sie hat nichts gebucht: keine Bestellung
// und kein Exemplar, auch nicht für die Positionen vor der abgelehnten.
func TestBestellung_AbgelehnterWarenkorbBuchtNichts(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	_, rufe := routertest.MitAufruf(t, pool)
	lieferant := testdaten.LegeLieferantAn(t, pool, "Abgelehnt", false)
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM lieferanten WHERE id = $1`, lieferant)
	})
	titel := testdaten.LegeTitelAn(t, pool, "Titel im Warenkorb")

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
		return pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM bestellungen_verlauf`),
			pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM buecher_exemplare`)
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

	// nummernstand liest barcode_seq, ohne die Folge vorzurücken. Das Zurückrollen gibt gezogene
	// Nummern nicht zurück; was vor dem Ziehen abgelehnt werden kann, darf keine verbrauchen.
	nummernstand := func(t *testing.T) string {
		t.Helper()
		var letzte int64
		var gezogen bool
		if err := pool.QueryRow(context.Background(), `SELECT last_value, is_called FROM barcode_seq`).Scan(&letzte, &gezogen); err != nil {
			t.Fatalf("Stand von barcode_seq lesen: %v", err)
		}
		return fmt.Sprintf("%d (gezogen: %t)", letzte, gezogen)
	}
	abgelehntOhneNummer := func(t *testing.T, steckt string, positionen ...string) {
		t.Helper()
		vorher := nummernstand(t)
		abgelehnt(t, http.StatusBadRequest, steckt, lieferant, positionen...)
		if nachher := nummernstand(t); nachher != vorher {
			t.Errorf("die Ablehnung hat Nummern gezogen: barcode_seq %s → %s", vorher, nachher)
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
		abgelehnt(t, http.StatusBadRequest, "„items[0].titel_id“", lieferant, position("", 1))
	})

	for _, menge := range []int{0, -1, 201} {
		t.Run(fmt.Sprintf("Menge %d: 400 mit der Position", menge), func(t *testing.T) {
			abgelehntOhneNummer(t, "Position 2", position(titel, 1), position(titel, menge))
		})
	}

	t.Run("Preis über 99.999,99 €: 400 mit der Position", func(t *testing.T) {
		abgelehntOhneNummer(t, "Position 2", position(titel, 1),
			fmt.Sprintf(`{"titel_id":%q,"menge":1,"preis":100000,"generate_barcodes":true}`, titel))
	})

	// Je Position gehen 200 Stück, die Zahl der Positionen begrenzt nur der Rumpf der Anfrage.
	// Zusammen gilt dieselbe Grenze wie für einen Listenimport.
	t.Run("zusammen mehr als 200.000 Exemplare: 400", func(t *testing.T) {
		positionen := make([]string, repository.HoechstensNummernJeVergabe/200+1)
		for i := range positionen {
			positionen[i] = position(titel, 200)
		}
		abgelehntOhneNummer(t, "höchstens 200.000 Exemplare", positionen...)
	})

	// Gegenprobe: Derselbe Warenkorb ohne die fehlerhafte Position wird gebucht. Ohne sie
	// bestünde der Test auch an einer Tür, die jede Bestellung abweist.
	t.Run("der berichtigte Warenkorb wird gebucht", func(t *testing.T) {
		_, exemplareVorher := gebucht(t)
		rec := bestelle(t, lieferant, position(titel, 2))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
		if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant); n != 1 {
			t.Errorf("%d Bestellungen beim Lieferanten, erwartet 1", n)
		}
		if _, e := gebucht(t); e != exemplareVorher+2 {
			t.Errorf("%d Exemplare angelegt, erwartet 2", e-exemplareVorher)
		}
	})
}
