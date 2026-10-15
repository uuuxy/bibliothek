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
//
// Der Server legt einen Ermittler mit den vertrauten Netzen an und setzt seine Middleware an
// den Anfang der Kette; jede Tür liest die Adresse danach mit AusAnfrage.
package clientip

import (
	"context"
	"net"
	"net/http"
	"strings"
)

// Ermittler ordnet einer Anfrage die Adresse des Clients zu und glaubt dabei nur Proxys aus
// seinen vertrauten Netzen und auf Loopback. Der Nullwert vertraut nur Loopback, damit die
// Entwicklung am eigenen Rechner und die Bereitschaftsprüfung ohne Einstellung laufen.
type Ermittler struct {
	vertraut []*net.IPNet
}

// NewErmittler liest die vertrauten Netze aus CIDR-Angaben oder einzelnen Adressen. Leere
// und unlesbare Einträge übergeht er, damit ein Tippfehler in der Umgebung den Start nicht
// verhindert; welche unlesbar waren, nennt UnlesbareEintraege.
func NewErmittler(netze []string) *Ermittler {
	e := &Ermittler{}
	for _, roh := range netze {
		if netz, ok := leseNetz(roh); ok {
			e.vertraut = append(e.vertraut, netz)
		}
	}
	return e
}

// UnlesbareEintraege liefert die Einträge, die NewErmittler übergeht, weil sie weder Netz noch
// Adresse sind; leere zählen nicht. Fehlt so das Netz des Proxys, sieht der Server jeden Browser
// unter der Adresse des Proxys, deshalb nennen der Start und die Selbstprüfung sie.
func UnlesbareEintraege(netze []string) []string {
	var unlesbar []string
	for _, roh := range netze {
		eintrag := strings.TrimSpace(roh)
		if eintrag == "" {
			continue
		}
		if _, ok := leseNetz(eintrag); !ok {
			unlesbar = append(unlesbar, eintrag)
		}
	}
	return unlesbar
}

// leseNetz liest einen Eintrag als Netz; eine einzelne Adresse ist ein Netz aus ihr allein.
// false heißt: leer oder weder Netz noch Adresse.
func leseNetz(roh string) (*net.IPNet, bool) {
	netz := strings.TrimSpace(roh)
	if netz == "" {
		return nil, false
	}
	if !strings.Contains(netz, "/") {
		if strings.Contains(netz, ":") {
			netz += "/128"
		} else {
			netz += "/32"
		}
	}
	_, gelesen, err := net.ParseCIDR(netz)
	if err != nil {
		return nil, false
	}
	return gelesen, true
}

// Middleware legt die Adresse des Clients in den Kontext der Anfrage, bevor next sie bekommt.
func (e *Ermittler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), adresseSchluessel{}, e.ausAnfrage(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adresseSchluessel ist der Schlüssel der ermittelten Adresse im Kontext einer Anfrage.
type adresseSchluessel struct{}

// AusAnfrage liefert die Adresse des Clients zu einer Anfrage, ohne Port: die, die die
// Middleware eines Ermittlers abgelegt hat. Kam die Anfrage an keiner vorbei, glaubt sie nur
// einem Proxy auf Loopback.
func AusAnfrage(r *http.Request) string {
	if adresse, ok := r.Context().Value(adresseSchluessel{}).(string); ok {
		return adresse
	}
	var nurLoopback Ermittler
	return nurLoopback.ausAnfrage(r)
}

// ausAnfrage liefert die Adresse des Clients ohne Port.
func (e *Ermittler) ausAnfrage(r *http.Request) string {
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

// istVertraut meldet, ob die Adresse auf Loopback oder in einem der vertrauten Netze liegt.
func (e *Ermittler) istVertraut(adresse string) bool {
	ip := net.ParseIP(adresse)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
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
