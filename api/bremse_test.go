package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/pkg/clientip"
)

// scheitereNMal schickt n Anfragen durch die Middleware, deren innere Tür immer 401 liefert
// (gescheiterte Anmeldung), und gibt den letzten Statuscode zurück.
func scheitereNMal(t *testing.T, remoteAddr, xff string, n int) int {
	t.Helper()
	handler := anmeldeBremse(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	last := 0
	for i := 0; i < n; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = remoteAddr
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		last = rec.Code
	}
	return last
}

func leereFehlversuche() {
	fehlversucheMu.Lock()
	fehlversucheJeAdresse = make(map[string]*fehlversuche)
	fehlversucheMu.Unlock()
}

// schickeNAnfragen schickt n GET-Anfragen auf path durch die allgemeine Bremse je Adresse
// (Limit=5, damit der Burst klein bleibt) und zählt, wie viele 429 zurückkamen.
func schickeNAnfragen(t *testing.T, path string, n int) (blocked int) {
	t.Helper()
	handler := anfrageBremse(5)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < n; i++ {
		req := httptest.NewRequest("GET", path, nil)
		req.RemoteAddr = "127.0.0.1:5000"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			blocked++
		}
	}
	return blocked
}

// Regressionstest für die 429-Flut der Etiketten-Vorschau: ein A4-Bogen rendert
// dutzende <img src="/api/barcode…"> gleichzeitig. /api/barcode muss — wie
// /api/images/cover — vom Bucket ausgenommen sein, sonst blockiert ein einziger
// Seitenaufruf einen Teil seiner eigenen Barcodes.
func TestAnfrageBremse_BarcodeSalveIstAusgenommen(t *testing.T) {
	clientip.SetzeProxys(nil) // nur Loopback
	t.Cleanup(func() { clientip.SetzeProxys(nil) })

	// 40 Barcodes in einem Rutsch, Bucket=5 → ohne Ausnahme wären 35 geblockt.
	if got := schickeNAnfragen(t, "/api/barcode?content=112622&width=150&height=50", 40); got != 0 {
		t.Fatalf("/api/barcode: %d Requests mit 429; want 0 (Endpoint ist vom Rate-Limiter ausgenommen)", got)
	}

	// Gegenprobe: ein normaler API-Pfad wird nach dem 5. Token weiter limitiert.
	if got := schickeNAnfragen(t, "/api/books", 40); got == 0 {
		t.Fatal("/api/books: 0 Requests mit 429; want >0 (normale Pfade bleiben limitiert)")
	}

	// /api/barcode/next (JSON, kein Fan-out) bleibt ebenfalls limitiert — der
	// exakte Pfad-Match darf den Unterpfad nicht mit ausnehmen.
	if got := schickeNAnfragen(t, "/api/barcode/next", 40); got == 0 {
		t.Fatal("/api/barcode/next: 0 Requests mit 429; want >0 (nur exakt /api/barcode ist ausgenommen)")
	}
}

// Hinter einem vertrauten Proxy haben zwei echte Aufrufer getrennte Zähler für Fehlversuche.
// Zählte die Bremse an r.RemoteAddr, also an der Adresse des Proxys, sperrten die Fehlversuche
// irgendeines Nutzers alle aus.
func TestAnmeldeBremse_AufruferHinterProxyZaehlenGetrennt(t *testing.T) {
	clientip.SetzeProxys([]string{"172.16.0.0/12"})
	t.Cleanup(func() { clientip.SetzeProxys(nil) })
	leereFehlversuche()

	const proxy = "172.18.0.5:40000"

	// Angreifer verbrennt sein Kontingent und wird gesperrt.
	if got := scheitereNMal(t, proxy, "203.0.113.9", maxAnmeldeFehlversucheJeAdresse); got != http.StatusUnauthorized {
		t.Fatalf("letzter erlaubter Versuch des Angreifers: Status %d; want 401 (noch nicht gesperrt)", got)
	}
	if got := scheitereNMal(t, proxy, "203.0.113.9", 1); got != http.StatusTooManyRequests {
		t.Fatalf("Versuch über dem Limit: Status %d; want 429 (gesperrt)", got)
	}

	// Ein anderer, unbeteiligter Nutzer hinter demselben Proxy darf sich
	// weiterhin einloggen — sein Zähler ist eigenständig.
	if got := scheitereNMal(t, proxy, "198.51.100.7", 1); got != http.StatusUnauthorized {
		t.Fatalf("unbeteiligter Nutzer: Status %d; want 401 (nicht mitgesperrt)", got)
	}
}

// Ein direkt (nicht über einen vertrauten Proxy) verbundener Angreifer darf sich nicht durch
// einen gefälschten Kopf X-Forwarded-For eine frische Identität erschummeln.
func TestAnmeldeBremse_FremderAufruferKannXFFNichtFaelschen(t *testing.T) {
	clientip.SetzeProxys([]string{"172.16.0.0/12"})
	t.Cleanup(func() { clientip.SetzeProxys(nil) })
	leereFehlversuche()

	const attacker = "203.0.113.50:40000"

	// Trotz rotierender, gefälschter XFF-Werte zählt nur die echte Peer-IP: Die
	// Versuche addieren sich über alle XFF-Identitäten hinweg zum selben Zähler.
	scheitereNMal(t, attacker, "1.1.1.1", maxAnmeldeFehlversucheJeAdresse-10)
	scheitereNMal(t, attacker, "2.2.2.2", 10)
	if got := scheitereNMal(t, attacker, "3.3.3.3", 1); got != http.StatusTooManyRequests {
		t.Fatalf("Angreifer über dem Limit: Status %d; want 429 (XFF-Spoof wirkungslos)", got)
	}
}

// In einer Schule hinter einer einzigen NAT-Adresse ist die Adresse eines Kollegen die aller.
// Sperrte die Bremse je Adresse schon nach wenigen Fehlversuchen, legten fünf Vertipper die
// Anmeldung des ganzen Standorts für 15 Minuten lahm, obwohl die Anmeldung selbst bewusst je
// E-Mail und Adresse drosselt.
//
// Der Test hält die Grenze fest, ab der die Sperre eintritt: deutlich oberhalb dessen, was ein
// Standort an ehrlichen Vertippern produziert.
func TestAnmeldeBremse_NATStandortWirdNichtVonWenigenVertippernGesperrt(t *testing.T) {
	clientip.SetzeProxys([]string{"172.16.0.0/12"})
	t.Cleanup(func() { clientip.SetzeProxys(nil) })
	leereFehlversuche()

	const proxy = "172.18.0.5:40000"
	const nat = "203.0.113.9" // eine einzige NAT-IP für die ganze Schule

	// Ein Dutzend Fehlversuche über den Tag verteilt — mehrere Nutzer, ein Standort.
	if got := scheitereNMal(t, proxy, nat, 12); got != http.StatusUnauthorized {
		t.Fatalf("12 Fehlversuche vom Schulstandort: Status %d; want 401 — der Standort darf "+
			"nicht gesperrt werden, sonst kommt niemand mehr rein", got)
	}

	// Gegenprobe: Ein Skript, das den Standort als Tarnung nutzt, läuft weiterhin
	// in die Sperre. Ohne diese Zusicherung wäre die Schicht wirkungslos statt weit.
	scheitereNMal(t, proxy, nat, maxAnmeldeFehlversucheJeAdresse)
	if got := scheitereNMal(t, proxy, nat, 1); got != http.StatusTooManyRequests {
		t.Fatalf("Credential-Stuffing über dem Limit: Status %d; want 429", got)
	}
}

func TestAnmeldeBremse_ErfolgSetztZaehlerZurueck(t *testing.T) {
	clientip.SetzeProxys(nil) // nur Loopback
	t.Cleanup(func() { clientip.SetzeProxys(nil) })
	leereFehlversuche()

	handler := anmeldeBremse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /login mit korrektem Passwort → 200
		if r.Header.Get("X-Test-Outcome") == "ok" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))

	do := func(outcome string) int {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "127.0.0.1:5000"
		if outcome != "" {
			req.Header.Set("X-Test-Outcome", outcome)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	do("")   // 1 Fehlversuch
	do("")   // 2
	do("ok") // Erfolg setzt den Zähler zurück
	for i := 0; i < maxAnmeldeFehlversucheJeAdresse; i++ {
		do("") // das volle Kontingent ist wieder nötig
	}
	if got := do(""); got != http.StatusTooManyRequests {
		t.Fatalf("nach Reset + vollem Kontingent an Fehlversuchen: Status %d; want 429", got)
	}
}

// 403 ist kein Fehlversuch: Bei „Zugang beantragt" (Selbstanmeldung) und „Konto deaktiviert"
// war das Passwort richtig, und die Anmeldung zählt deshalb bewusst nicht (auth/handlers.go).
// Zählte die äußere Bremse je Adresse 403 trotzdem, wäre die Anmeldung zum Schuljahresbeginn
// für die ganze Schule 15 Minuten dicht: Zwanzig Kollegen probieren die Selbstanmeldung
// mehrfach, alle hinter einer einzigen NAT-Adresse, und nach 50 solcher 403 sperrte die Bremse
// auch die Bibliothekskraft aus, die die Freischaltungen vornehmen müsste.
func TestAnmeldeBremse_403ZaehltNichtAlsFehlversuch(t *testing.T) {
	fehlversucheMu.Lock()
	fehlversucheJeAdresse = make(map[string]*fehlversuche)
	fehlversucheMu.Unlock()

	zugangBeantragt := anmeldeBremse(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))

	for versuch := 1; versuch <= maxAnmeldeFehlversucheJeAdresse+1; versuch++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "198.51.100.20:9999"
		rec := httptest.NewRecorder()
		zugangBeantragt.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("Versuch %d: 429 — 403 (richtiges Passwort, Zugang nur noch nicht frei) wurde als Fehlversuch gezählt", versuch)
		}
	}
}
