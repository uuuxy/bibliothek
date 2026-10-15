package clientip

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func anfrage(gegenstelle string, koepfe map[string]string) *http.Request {
	r := &http.Request{RemoteAddr: gegenstelle, Header: http.Header{}}
	for name, wert := range koepfe {
		r.Header.Set(name, wert)
	}
	return r
}

// Wer den Server direkt erreicht, setzt die Köpfe selbst; sie dürfen die Adresse der
// Verbindung nicht ersetzen.
func TestErmittler_FremdeGegenstelleZaehltKoepfeNicht(t *testing.T) {
	var e Ermittler

	adresse := e.ausAnfrage(anfrage("203.0.113.9:5000", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
		"X-Real-IP":       "5.6.7.8",
	}))
	if adresse != "203.0.113.9" {
		t.Fatalf("ausAnfrage = %q, erwartet die Gegenstelle 203.0.113.9", adresse)
	}
}

func TestErmittler_LoopbackGlaubtForwardedFor(t *testing.T) {
	var e Ermittler

	for _, gegenstelle := range []string{"127.0.0.1:5000", "[::1]:5000"} {
		adresse := e.ausAnfrage(anfrage(gegenstelle, map[string]string{
			"X-Forwarded-For": "198.51.100.7",
		}))
		if adresse != "198.51.100.7" {
			t.Errorf("Gegenstelle %s: ausAnfrage = %q, erwartet den weitergereichten Client 198.51.100.7", gegenstelle, adresse)
		}
	}
}

// Caddy steht im Docker-Netz 172.16/12 und hängt den Client an.
func TestErmittler_VertrauterProxyLiefertClient(t *testing.T) {
	e := NewErmittler([]string{"172.16.0.0/12"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	}))
	if adresse != "198.51.100.7" {
		t.Fatalf("ausAnfrage = %q, erwartet 198.51.100.7", adresse)
	}
}

// Der Client stellt einen Eintrag voran, Caddy hängt die echte Gegenstelle rechts an. Nur
// der rechte Eintrag zählt.
func TestErmittler_VorangestellterEintragZaehltNicht(t *testing.T) {
	e := NewErmittler([]string{"172.16.0.0/12"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "10.9.9.9, 198.51.100.7",
	}))
	if adresse != "198.51.100.7" {
		t.Fatalf("ausAnfrage = %q, erwartet den vom Proxy angehängten Client 198.51.100.7", adresse)
	}
}

// Die Clients der Schule liegen selbst in den Netzen, die als Proxy-Netze vertraut sind.
// Überspringt der Ermittler vertraute Einträge, werden alle Geräte der Schule für die Bremse
// der Anmeldung und das Protokoll ein Client mit Caddys Adresse.
func TestErmittler_LanClientHinterProxy(t *testing.T) {
	e := NewErmittler([]string{"172.16.0.0/12", "10.0.0.0/8", "192.168.0.0/16"})

	a := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{"X-Forwarded-For": "192.168.1.50"}))
	b := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{"X-Forwarded-For": "192.168.1.77"}))
	if a != "192.168.1.50" || b != "192.168.1.77" {
		t.Fatalf("ausAnfrage = %q und %q, erwartet 192.168.1.50 und 192.168.1.77", a, b)
	}
}

// Reicht der Proxy den eingehenden Kopf durch und hängt die echte Adresse an, darf der
// Client seine Adresse nicht wählen, auch wenn seine echte in einem vertrauten Netz liegt.
func TestErmittler_DurchgereichterHeaderIstNichtWaehlbar(t *testing.T) {
	e := NewErmittler([]string{"172.16.0.0/12", "10.0.0.0/8", "192.168.0.0/16"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "8.8.8.8, 192.168.1.50",
	}))
	if adresse != "192.168.1.50" {
		t.Fatalf("ausAnfrage = %q, erwartet die vom Proxy angehängte 192.168.1.50", adresse)
	}
}

// Ein zweiter Proxy vor Caddy wird nicht übersprungen: Wer eine Kette betreibt, sieht dessen
// Adresse statt einer still falschen.
func TestErmittler_KetteLiefertLetztenHop(t *testing.T) {
	e := NewErmittler([]string{"172.16.0.0/12", "192.0.2.0/24"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "203.0.113.9, 192.0.2.10",
	}))
	if adresse != "192.0.2.10" {
		t.Fatalf("ausAnfrage = %q, erwartet den letzten Hop 192.0.2.10", adresse)
	}
}

func TestErmittler_OhneForwardedForFaelltAufRealIPZurueck(t *testing.T) {
	e := NewErmittler([]string{"172.16.0.0/12"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Real-IP": "198.51.100.7",
	}))
	if adresse != "198.51.100.7" {
		t.Fatalf("ausAnfrage = %q, erwartet X-Real-IP 198.51.100.7", adresse)
	}
}

func TestErmittler_VertrauterProxyOhneKoepfeLiefertGegenstelle(t *testing.T) {
	e := NewErmittler([]string{"172.16.0.0/12"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", nil))
	if adresse != "172.18.0.5" {
		t.Fatalf("ausAnfrage = %q, erwartet die Gegenstelle 172.18.0.5", adresse)
	}
}

// Die Gegenstelle kommt ohne Port, und X-Forwarded-For endet auf einen leeren Eintrag.
func TestErmittler_AdresseOhnePortUndLeererEintrag(t *testing.T) {
	e := NewErmittler([]string{"172.16.0.0/12"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5", map[string]string{
		"X-Forwarded-For": "198.51.100.7, ",
	}))
	if adresse != "198.51.100.7" {
		t.Fatalf("ausAnfrage = %q, erwartet 198.51.100.7", adresse)
	}
}

// Die Tür liest die Adresse, die die Middleware des Ermittlers abgelegt hat; so gilt überall
// der Ermittler, den der Server mit seinen Netzen angelegt hat.
func TestAusAnfrage_LiestDieAdresseDerMiddleware(t *testing.T) {
	var adresse string
	tuer := NewErmittler([]string{"172.16.0.0/12"}).Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		adresse = AusAnfrage(r)
	}))

	tuer.ServeHTTP(httptest.NewRecorder(), anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	}))
	if adresse != "198.51.100.7" {
		t.Errorf("AusAnfrage hinter der Middleware = %q, erwartet 198.51.100.7", adresse)
	}
}

// Ohne Middleware glaubt AusAnfrage nur einem Proxy auf Loopback.
func TestAusAnfrage_OhneMiddlewareNurLoopback(t *testing.T) {
	if adresse := AusAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	})); adresse != "172.18.0.5" {
		t.Errorf("Proxy im Docker-Netz: AusAnfrage = %q, erwartet die Gegenstelle 172.18.0.5", adresse)
	}
	if adresse := AusAnfrage(anfrage("127.0.0.1:5000", map[string]string{
		"X-Forwarded-For": "203.0.113.1",
	})); adresse != "203.0.113.1" {
		t.Errorf("Proxy auf Loopback: AusAnfrage = %q, erwartet 203.0.113.1", adresse)
	}
}

// Der Start schneidet TRUSTED_PROXIES an den Kommas: Leere Einträge, Leerzeichen und einzelne
// Adressen kommen so an, ein unlesbarer Eintrag verhindert den Start nicht.
func TestNewErmittler_EintraegeAusDerUmgebung(t *testing.T) {
	ohne := NewErmittler(strings.Split("", ","))
	if adresse := ohne.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	})); adresse != "172.18.0.5" {
		t.Errorf("ohne TRUSTED_PROXIES: ausAnfrage = %q, erwartet die Gegenstelle 172.18.0.5", adresse)
	}

	e := NewErmittler(strings.Split(" 172.16.0.0/12 , 10.1.2.3,kaputt", ","))
	faelle := []struct {
		gegenstelle, erwartet string
	}{
		{"172.18.0.5:5000", "198.51.100.7"},
		{"10.1.2.3:5000", "198.51.100.7"},
		{"10.1.2.4:5000", "10.1.2.4"},
	}
	for _, f := range faelle {
		if adresse := e.ausAnfrage(anfrage(f.gegenstelle, map[string]string{
			"X-Forwarded-For": "198.51.100.7",
		})); adresse != f.erwartet {
			t.Errorf("Gegenstelle %s: ausAnfrage = %q, erwartet %q", f.gegenstelle, adresse, f.erwartet)
		}
	}
}
