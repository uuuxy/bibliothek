package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/bereitschaft"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"
)

// Die Seite „Betriebsbereitschaft" nennt die Einträge aus TRUSTED_PROXIES, die der Server
// übergeht, und zwar aus der Liste, mit der er selbst arbeitet. Die Umgebung des Tests trägt
// einen lesbaren Wert: Läse die Selbstprüfung sie ein zweites Mal, meldete sie nichts.
func TestBetriebsbereitschaft_NenntUnlesbarenProxyDesServers(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "172.16.0.0/12")
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	_, sitzung, anmeldung := tuertest.AdminSitzung(t, pool, "proxy-bereitschaft@example.org", "Paula", "Proxy")
	router := NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(),
		ServerKonfig{VertrauteProxys: []string{"172.16.0.0/12", "caddy"}}).Routes()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/system/betriebsbereitschaft", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Befunde []bereitschaft.Befund `json:"befunde"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	for _, b := range antwort.Befunde {
		if b.Bereich != "Vertraute Proxys" {
			continue
		}
		if b.Stufe != bereitschaft.StufeWarnung || !strings.Contains(b.Befund, "caddy") {
			t.Errorf("Befund „Vertraute Proxys“ = %q (%s), erwartet eine Warnung, die caddy nennt", b.Befund, b.Stufe)
		}
		return
	}
	t.Fatal("kein Befund „Vertraute Proxys“ in der Antwort")
}
