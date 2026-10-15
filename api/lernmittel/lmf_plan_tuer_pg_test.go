package lernmittel_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/schulzeit"
	"bibliothek/sse"
)

// Die Türen des LMF-Plans über den ganzen Router, mit Sitzung und CSRF. Der Test steht in einem
// eigenen Testpaket, weil api/ das Thema einbindet und ein Test im Paket des Themas api/ deshalb
// nicht einbinden kann. Geprüft wird, dass die Türen auf der Basis des Servers stehen: Sie
// schreiben in seine Datenbank und rechnen mit seiner Uhr, auch wenn sie erst nach dem Bau des
// Routers gestellt wird.
func TestLMFPlan_UeberDenRouter_StehtAufDerBasisDesServers(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	raeumePlaene := func() { pgtest.RaeumeAuf(t, pool, `DELETE FROM lmf_plaene`) }
	raeumePlaene()
	t.Cleanup(raeumePlaene)

	_, sitzung, anmeldung := tuertest.AdminSitzung(t, pool, "lmf-plan-tuer@example.org", "Lena", "Plan")
	srv := api.NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(), false)
	router := srv.Routes()
	// Die Uhr erst nach dem Bau des Routers: Mit einer Kopie der Basis rechneten die Türen des
	// Themas weiter mit der Uhr des Rechners.
	srv.Uhr = func() time.Time { return time.Date(2001, time.March, 1, 10, 0, 0, 0, schulzeit.Zone()) }

	rufe := func(methode, pfad, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, tuertest.MitCSRF(req))
		return rec
	}

	// Speichern legt den Entwurf in der Datenbank des Servers ab.
	rec := rufe(http.MethodPut, "/api/lmf-plan/ausgabe",
		`{"erster_tag":"2027-08-10","startstunde":2,"stunden_je_tag":6,"zeilen":[{"klassen":["5F1"]}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/lmf-plan/ausgabe: Status %d: %s", rec.Code, rec.Body.String())
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM lmf_plaene WHERE art = 'ausgabe'`); n != 1 {
		t.Errorf("%d Pläne der Art ausgabe in der Datenbank, erwartet 1", n)
	}

	// Die Liste für das Kollegium beginnt am 1. August des Schuljahres, das die Uhr des Servers
	// nennt.
	rec = rufe(http.MethodGet, "/api/lmf-termine", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/lmf-termine: Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Ab string `json:"ab"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein JSON (%s): %v", rec.Body.String(), err)
	}
	if antwort.Ab != "2000-08-01" {
		t.Errorf("Liste ab %q, erwartet 2000-08-01: Die Türen rechnen nicht mit der Uhr des Servers", antwort.Ab)
	}
}
