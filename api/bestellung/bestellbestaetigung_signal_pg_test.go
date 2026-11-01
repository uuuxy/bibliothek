package bestellung

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/bestelllink"
)

// Das Signal folgt dem Eintrag: Es kommt über den Link wie beim Nachtrag von Hand, und es bleibt
// aus, wo nichts geschrieben wurde. Sonst holten alle offenen Fenster bei einem Doppelklick des
// Händlers oder einem abgewiesenen Aufruf ihre Listen neu.
func TestBestaetigung_SignalNurNachDemEintrag(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	broker, signale := tuertest.BeobachteLeitung(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Broker: broker}}

	haupt := testdaten.LegeLieferantAn(t, pool, "Naacher", true)
	_, token := bestellungMitToken(t, pool, haupt, bestelllink.TageVorgabe)
	_, abgelaufen := bestellungMitToken(t, pool, haupt, -1)
	vonHand := bestellungFuerLieferant(t, pool, haupt)
	ohneSchritt := bestellungFuerLieferant(t, pool, testdaten.LegeLieferantAn(t, pool, "Normalhaendler", false))

	const klein = `{"etiketten_groesse":"klein"}`
	for _, s := range []struct {
		name            string
		aufruf          func() *httptest.ResponseRecorder
		status, signale int
	}{
		{"Link: unbekannte Größe", func() *httptest.ResponseRecorder {
			return bestaetigeUeberLink(srv, token, `{"etiketten_groesse":"riesig"}`)
		}, http.StatusBadRequest, 0},
		{"Link: abgelaufen", func() *httptest.ResponseRecorder {
			return bestaetigeUeberLink(srv, abgelaufen, `{}`)
		}, http.StatusNotFound, 0},
		{"Link: erster Klick", func() *httptest.ResponseRecorder {
			return bestaetigeUeberLink(srv, token, `{}`)
		}, http.StatusOK, 1},
		{"Link: zweiter Klick", func() *httptest.ResponseRecorder {
			return bestaetigeUeberLink(srv, token, `{}`)
		}, http.StatusConflict, 0},
		{"von Hand: ohne Größe", func() *httptest.ResponseRecorder {
			return bestaetigeVonHand(srv, vonHand, `{}`)
		}, http.StatusBadRequest, 0},
		{"von Hand: ohne Bestätigungsschritt", func() *httptest.ResponseRecorder {
			return bestaetigeVonHand(srv, ohneSchritt, klein)
		}, http.StatusBadRequest, 0},
		{"von Hand: Nachtrag", func() *httptest.ResponseRecorder {
			return bestaetigeVonHand(srv, vonHand, klein)
		}, http.StatusOK, 1},
		{"von Hand: schon bestätigt", func() *httptest.ResponseRecorder {
			return bestaetigeVonHand(srv, vonHand, klein)
		}, http.StatusConflict, 0},
	} {
		rec := s.aufruf()
		if rec.Code != s.status {
			t.Fatalf("%s: Status %d, erwartet %d: %s", s.name, rec.Code, s.status, rec.Body.String())
		}
		if n := signale("bestellstand"); n != s.signale {
			t.Errorf("%s: %d Signale, erwartet %d", s.name, n, s.signale)
		}
	}
}

// Scheitert das Schreiben, meldet die Antwort bereits=false. Ein Signal schickte dann alle
// Fenster auf den alten Stand zurück.
func TestBestaetigung_KeinSignalWennDasSchreibenScheitert(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	broker, signale := tuertest.BeobachteLeitung(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Broker: broker}}

	bestellung, _ := bestellungMitToken(t, pool, testdaten.LegeLieferantAn(t, pool, "Naacher", true), bestelllink.TageVorgabe)
	abgebrochen, abbrechen := context.WithCancel(context.Background())
	abbrechen()

	if _, err := srv.bestaetigeBestellung(abgebrochen, bestellung, "", "", "lieferant"); err == nil {
		t.Fatal("bestaetigeBestellung mit abgebrochenem Kontext: kein Fehler")
	}
	if n := signale("bestellstand"); n != 0 {
		t.Errorf("%d Signale nach dem gescheiterten Schreiben, erwartet 0", n)
	}
}
