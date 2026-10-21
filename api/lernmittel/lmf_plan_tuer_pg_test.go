package lernmittel_test

import (
	"context"
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
	srv := api.NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(), api.ServerKonfig{})
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

// Ein Speichern gilt nur auf dem Stand, den der Planer geladen hat. Von einer Änderung an einem
// anderen Platz erfährt er sonst nur über die Live-Leitung; war sie unterbrochen, überschriebe
// er die Änderung still. Die Tür antwortet dann mit 409 und lässt den Plan, wie er ist.
func TestLMFPlan_SpeichernAufUeberholtemStand_WirdAbgelehnt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	raeumePlaene := func() { pgtest.RaeumeAuf(t, pool, `DELETE FROM lmf_plaene`) }
	raeumePlaene()
	t.Cleanup(raeumePlaene)

	_, sitzung, anmeldung := tuertest.AdminSitzung(t, pool, "lmf-plan-stand@example.org", "Lena", "Stand")
	router := api.NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(), api.ServerKonfig{}).Routes()
	rufe := func(methode, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, "/api/lmf-plan/ausgabe", strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, tuertest.MitCSRF(req))
		return rec
	}
	// geladenerStand ist der Stempel, den der Planer beim Laden bekommt.
	geladenerStand := func() string {
		t.Helper()
		rec := rufe(http.MethodGet, "")
		var antwort struct {
			Plan *struct {
				AktualisiertAm string `json:"aktualisiert_am"`
			} `json:"plan"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || antwort.Plan == nil {
			t.Fatalf("GET /api/lmf-plan/ausgabe: %d %s", rec.Code, rec.Body.String())
		}
		return antwort.Plan.AktualisiertAm
	}
	speichere := func(stand, vermerk string) *httptest.ResponseRecorder {
		t.Helper()
		return rufe(http.MethodPut, `{"erster_tag":"2027-08-10","startstunde":2,"stunden_je_tag":6,`+
			`"zeilen":[{"vermerk":"`+vermerk+`"}],"aktualisiert_am":"`+stand+`"}`)
	}
	vermerk := func() string {
		t.Helper()
		var v string
		if err := pool.QueryRow(context.Background(), `SELECT string_agg(t.vermerk, ',')
			FROM lmf_termine t JOIN lmf_plaene p ON p.id = t.plan_id WHERE p.art = 'ausgabe'`).Scan(&v); err != nil {
			t.Fatalf("Zeilen lesen: %v", err)
		}
		return v
	}
	abgelehnt := func(rec *httptest.ResponseRecorder, fall string) {
		t.Helper()
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "an einem anderen Platz geändert") {
			t.Errorf("%s: %d %s, erwartet 409 mit dem Hinweis auf den anderen Platz", fall, rec.Code, rec.Body.String())
		}
	}

	// Zwei Plätze öffnen den Planer, als es noch keinen Plan gibt; der erste speichert.
	if rec := speichere("", "erster Platz"); rec.Code != http.StatusOK {
		t.Fatalf("erstes Speichern: %d %s", rec.Code, rec.Body.String())
	}
	abgelehnt(speichere("", "zweiter Platz"), "zweiter Platz ohne Plan geladen")

	// Der erste speichert auf seinem neuen Stand weiter; mit dem Stand davor wird abgelehnt.
	vorher := geladenerStand()
	if rec := speichere(vorher, "erster Platz, zweites Mal"); rec.Code != http.StatusOK {
		t.Fatalf("Speichern auf dem geladenen Stand: %d %s", rec.Code, rec.Body.String())
	}
	if geladenerStand() == vorher {
		t.Fatal("Der Stempel ist nach dem Speichern derselbe; der Vergleich unterschiede nichts")
	}
	abgelehnt(speichere(vorher, "überholt"), "Speichern auf einem überholten Stand")
	if v := vermerk(); v != "erster Platz, zweites Mal" {
		t.Errorf("Der Plan trägt %q, erwartet den Stand des letzten erlaubten Speicherns", v)
	}

	// Die Vorschau schreibt nichts und rechnet auf jedem Stand.
	if rec := rufe(http.MethodPut, `{"erster_tag":"2027-08-10","startstunde":2,"stunden_je_tag":6,`+
		`"zeilen":[{"vermerk":"Vorschau"}],"vorschau":true,"aktualisiert_am":"`+vorher+`"}`); rec.Code != http.StatusOK {
		t.Errorf("Vorschau auf einem überholten Stand: %d %s", rec.Code, rec.Body.String())
	}
}
