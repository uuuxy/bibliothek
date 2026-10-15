package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/sse"
)

// Der Server setzt die vertrauten Proxys aus seiner Konfiguration an den Anfang der Kette:
// Hinter Caddy zieht jeder Browser von seinem eigenen Vorrat der Bremse je Adresse. Ohne das
// Glied zählten alle unter der Adresse des Proxys, und ein Browser bremste die ganze Schule.
func TestServer_BremseSiehtDenBrowserHinterDemProxy(t *testing.T) {
	t.Setenv("RATE_LIMIT", "2")
	t.Chdir(t.TempDir())
	router := NewServer(&db.Verbindung{}, nil, sse.NewBroker(),
		ServerKonfig{VertrauteProxys: []string{"172.16.0.0/12"}}).Routes()
	hole := func(browser string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/gibt-es-nicht", nil)
		req.RemoteAddr = "172.18.0.5:40000"
		req.Header.Set("X-Forwarded-For", browser)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	hole("198.51.100.7")
	hole("198.51.100.7")
	if code := hole("198.51.100.7"); code != http.StatusTooManyRequests {
		t.Fatalf("dritte Anfrage desselben Browsers: Status %d, erwartet 429; die Bremse greift nicht", code)
	}
	if code := hole("198.51.100.8"); code == http.StatusTooManyRequests {
		t.Errorf("erste Anfrage eines anderen Browsers hinter demselben Proxy: Status 429; " +
			"die Bremse zählt die Adresse des Proxys")
	}
}
