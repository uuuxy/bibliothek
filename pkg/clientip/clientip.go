// Package clientip ermittelt zu einer Anfrage die Adresse des Clients, auch hinter einem
// Reverse-Proxy wie Caddy.
//
// X-Forwarded-For und X-Real-IP gelten nur, wenn die Anfrage von einem vertrauten Proxy kommt.
// Erreicht sie den Server direkt, hat der Client diese Köpfe selbst gesetzt, und sie zählen
// nicht.
//
// Geglaubt wird genau ein Proxy: Der Client ist der rechte Eintrag in X-Forwarded-For, den der
// eigene Proxy angehängt hat. Weiter links wird nicht gelesen. Die Clients der Schule liegen in
// denselben privaten Netzen wie die Proxys; wer vertraute Einträge von rechts überspringt, hält
// jedes Gerät der Schule für den Proxy und lässt einen Client mit einem vorangestellten Eintrag
// seine Adresse wählen. Steht ein zweiter Proxy vor Caddy, liefert das Paket deshalb dessen
// Adresse und nicht die des Clients.
package clientip

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

// loopbackNetze sind immer vertraut, damit die Entwicklung am eigenen Rechner und die
// Bereitschaftsprüfung ohne Einstellung laufen.
var loopbackNetze = []string{"127.0.0.0/8", "::1/128"}

// ermittler ordnet einer Anfrage die Adresse des Clients zu und glaubt dabei nur Proxys aus
// seinen vertrauten Netzen. Er wird nach dem Anlegen nicht mehr geändert.
type ermittler struct {
	vertraut []*net.IPNet
}

// neuerErmittler liest die vertrauten Netze aus CIDR-Angaben oder einzelnen Adressen. Leere
// und unlesbare Einträge übergeht er, damit ein Tippfehler in der Umgebung den Start nicht
// verhindert.
func neuerErmittler(netze []string) *ermittler {
	e := &ermittler{}
	for _, roh := range netze {
		netz := strings.TrimSpace(roh)
		if netz == "" {
			continue
		}
		if !strings.Contains(netz, "/") {
			if strings.Contains(netz, ":") {
				netz += "/128"
			} else {
				netz += "/32"
			}
		}
		if _, gelesen, err := net.ParseCIDR(netz); err == nil {
			e.vertraut = append(e.vertraut, gelesen)
		}
	}
	return e
}

// ausAnfrage liefert die Adresse des Clients ohne Port.
func (e *ermittler) ausAnfrage(r *http.Request) string {
	gegenstelle := ohnePort(r.RemoteAddr)
	if !e.istVertraut(gegenstelle) {
		return gegenstelle
	}
	if adresse := rechtesterEintrag(r.Header.Get("X-Forwarded-For")); adresse != "" {
		return adresse
	}
	// X-Real-IP trägt nur einen Wert und stammt vom selben Proxy.
	if adresse := strings.TrimSpace(r.Header.Get("X-Real-IP")); adresse != "" {
		return adresse
	}
	return gegenstelle
}

// rechtesterEintrag liefert den letzten nicht leeren Eintrag einer Liste mit Kommas, sonst
// eine leere Zeichenkette.
func rechtesterEintrag(liste string) string {
	eintraege := strings.Split(liste, ",")
	for i := len(eintraege) - 1; i >= 0; i-- {
		if adresse := strings.TrimSpace(eintraege[i]); adresse != "" {
			return adresse
		}
	}
	return ""
}

// istVertraut meldet, ob die Adresse in einem der vertrauten Netze liegt.
func (e *ermittler) istVertraut(adresse string) bool {
	ip := net.ParseIP(adresse)
	if ip == nil {
		return false
	}
	for _, netz := range e.vertraut {
		if netz.Contains(ip) {
			return true
		}
	}
	return false
}

// ohnePort nimmt den Port von einer Adresse. Eine Adresse ohne Port bleibt, wie sie ist; so
// kommt sie in Tests und über einen Unix-Socket.
func ohnePort(adresse string) string {
	if host, _, err := net.SplitHostPort(adresse); err == nil {
		return host
	}
	return adresse
}

// prozessweit ist der Ermittler, den AusAnfrage nimmt. Er vertraut zunächst nur Loopback;
// der Start ersetzt ihn über SetzeProxysAusUmgebung.
var prozessweit atomic.Pointer[ermittler]

func init() {
	prozessweit.Store(neuerErmittler(loopbackNetze))
}

// SetzeProxys legt fest, aus welchen Netzen ein Proxy als vertraut gilt. Loopback gilt
// zusätzlich immer.
func SetzeProxys(netze []string) {
	alle := make([]string, 0, len(netze)+len(loopbackNetze))
	alle = append(alle, netze...)
	alle = append(alle, loopbackNetze...)
	prozessweit.Store(neuerErmittler(alle))
}

// SetzeProxysAusUmgebung liest die vertrauten Netze aus TRUSTED_PROXIES, getrennt durch
// Kommas. Ist die Variable leer, bleibt nur Loopback vertraut.
func SetzeProxysAusUmgebung() {
	roh := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if roh == "" {
		SetzeProxys(nil)
		return
	}
	SetzeProxys(strings.Split(roh, ","))
}

// AusAnfrage liefert die Adresse des Clients zu einer Anfrage, ohne Port.
func AusAnfrage(r *http.Request) string {
	return prozessweit.Load().ausAnfrage(r)
}
