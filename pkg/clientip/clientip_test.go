package clientip

import (
	"net/http"
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
	e := neuerErmittler(loopbackNetze)

	adresse := e.ausAnfrage(anfrage("203.0.113.9:5000", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
		"X-Real-IP":       "5.6.7.8",
	}))
	if adresse != "203.0.113.9" {
		t.Fatalf("ausAnfrage = %q, erwartet die Gegenstelle 203.0.113.9", adresse)
	}
}

func TestErmittler_LoopbackGlaubtForwardedFor(t *testing.T) {
	e := neuerErmittler(loopbackNetze)

	adresse := e.ausAnfrage(anfrage("127.0.0.1:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	}))
	if adresse != "198.51.100.7" {
		t.Fatalf("ausAnfrage = %q, erwartet den weitergereichten Client 198.51.100.7", adresse)
	}
}

// Caddy steht im Docker-Netz 172.16/12 und hängt den Client an.
func TestErmittler_VertrauterProxyLiefertClient(t *testing.T) {
	e := neuerErmittler([]string{"172.16.0.0/12"})

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
	e := neuerErmittler([]string{"172.16.0.0/12"})

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
	e := neuerErmittler([]string{"172.16.0.0/12", "10.0.0.0/8", "192.168.0.0/16"})

	a := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{"X-Forwarded-For": "192.168.1.50"}))
	b := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{"X-Forwarded-For": "192.168.1.77"}))
	if a != "192.168.1.50" || b != "192.168.1.77" {
		t.Fatalf("ausAnfrage = %q und %q, erwartet 192.168.1.50 und 192.168.1.77", a, b)
	}
}

// Reicht der Proxy den eingehenden Kopf durch und hängt die echte Adresse an, darf der
// Client seine Adresse nicht wählen, auch wenn seine echte in einem vertrauten Netz liegt.
func TestErmittler_DurchgereichterHeaderIstNichtWaehlbar(t *testing.T) {
	e := neuerErmittler([]string{"172.16.0.0/12", "10.0.0.0/8", "192.168.0.0/16"})

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
	e := neuerErmittler([]string{"172.16.0.0/12", "192.0.2.0/24"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "203.0.113.9, 192.0.2.10",
	}))
	if adresse != "192.0.2.10" {
		t.Fatalf("ausAnfrage = %q, erwartet den letzten Hop 192.0.2.10", adresse)
	}
}

func TestErmittler_OhneForwardedForFaelltAufRealIPZurueck(t *testing.T) {
	e := neuerErmittler([]string{"172.16.0.0/12"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Real-IP": "198.51.100.7",
	}))
	if adresse != "198.51.100.7" {
		t.Fatalf("ausAnfrage = %q, erwartet X-Real-IP 198.51.100.7", adresse)
	}
}

func TestErmittler_VertrauterProxyOhneKoepfeLiefertGegenstelle(t *testing.T) {
	e := neuerErmittler([]string{"172.16.0.0/12"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5:5000", nil))
	if adresse != "172.18.0.5" {
		t.Fatalf("ausAnfrage = %q, erwartet die Gegenstelle 172.18.0.5", adresse)
	}
}

// Die Gegenstelle kommt ohne Port, und X-Forwarded-For endet auf einen leeren Eintrag.
func TestErmittler_AdresseOhnePortUndLeererEintrag(t *testing.T) {
	e := neuerErmittler([]string{"172.16.0.0/12"})

	adresse := e.ausAnfrage(anfrage("172.18.0.5", map[string]string{
		"X-Forwarded-For": "198.51.100.7, ",
	}))
	if adresse != "198.51.100.7" {
		t.Fatalf("ausAnfrage = %q, erwartet 198.51.100.7", adresse)
	}
}

func TestSetzeProxysAusUmgebung(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "172.16.0.0/12")
	SetzeProxysAusUmgebung()
	t.Cleanup(func() {
		prozessweit.Store(neuerErmittler(loopbackNetze))
	})

	if adresse := AusAnfrage(anfrage("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	})); adresse != "198.51.100.7" {
		t.Fatalf("eingestellter Proxy: AusAnfrage = %q, erwartet 198.51.100.7", adresse)
	}
	if adresse := AusAnfrage(anfrage("127.0.0.1:5000", map[string]string{
		"X-Forwarded-For": "203.0.113.1",
	})); adresse != "203.0.113.1" {
		t.Fatalf("Loopback gilt zusätzlich: AusAnfrage = %q, erwartet 203.0.113.1", adresse)
	}
	if adresse := AusAnfrage(anfrage("203.0.113.9:5000", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
	})); adresse != "203.0.113.9" {
		t.Fatalf("fremde Gegenstelle: AusAnfrage = %q, erwartet 203.0.113.9", adresse)
	}
}
