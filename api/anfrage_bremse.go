package api

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/ratelimit"
)

// meldungZuVieleAnfragen steht an der Oberfläche, wenn die Bremse eine Anfrage abweist.
const meldungZuVieleAnfragen = "Zu viele Anfragen in kurzer Zeit. Bitte einen Moment warten."

// anfrageBremse lässt je Adresse höchstens jeSekunde Anfragen in der Sekunde durch.
// Ausgenommen sind Pfade, bei denen ein einzelner Seitenaufruf Dutzende Anfragen erzeugt; die
// Begründung steht an der Ausnahmeliste.
func anfrageBremse(jeSekunde int) func(http.Handler) http.Handler {
	bremse := ratelimit.New(jeSekunde)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bilder und Uploads sind ausgenommen, weil ein Seitenaufruf oft Dutzende Bilder
			// gleichzeitig lädt. /api/barcode gehört in dieselbe Klasse: Der Stapeldruck der
			// Ausweise einer Klasse (designer/Druckvorschau.svelte, je Leser eine Kartenseite) ruft
			// je Karte ein <img src="/api/barcode…"> fast gleichzeitig ab; mit Bremse käme ein Teil
			// davon als 429 zurück. Der Browser legt die Bilder nicht ab
			// (internal/middleware/sicherheitskopfzeilen.go), jeder Seitenaufruf holt sie neu.
			// Der Pfad muss genau passen, damit /api/barcode/next (JSON) gebremst bleibt.
			// /events (SSE) ist eine lange Verbindung, die sonst bei jedem neuen Aufbau vom
			// Kontingent nähme und unruhige Verbindungen unnötig ausbremste.
			if strings.HasPrefix(r.URL.Path, "/api/images/cover") || strings.HasPrefix(r.URL.Path, "/uploads/") || r.URL.Path == "/api/barcode" || r.URL.Path == "/events" {
				next.ServeHTTP(w, r)
				return
			}

			adresse := clientip.AusAnfrage(r)
			if !bremse.LaesstDurch(adresse) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusTooManyRequests, meldungZuVieleAnfragen, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// anfragenJeSekundeAusUmgebung liefert, wie viele Anfragen je Adresse und Sekunde durchgehen.
//
// Die Vorgabe ist 50, der Wert des Betriebs. RATE_LIMIT setzt einen anderen nur für die
// E2E-Suite: Ein Browser an einer Adresse läuft dort ohne menschliche Pausen durch alle
// Abläufe, und jeder Seitenaufruf löst mehrere Anfragen aus. Der Wert macht den Testlauf
// verlässlich, statt ihn an einem Zufall scheitern zu lassen.
//
// Nicht an APP_ENV gekoppelt: Eine Grenze, die sich je nach Umgebung selbst verstellt, ist in
// dem Moment weg, in dem jemand APP_ENV falsch setzt. So steht der abweichende Wert sichtbar
// in docker-compose.local.yml und nirgends sonst.
func anfragenJeSekundeAusUmgebung() int {
	roh := os.Getenv("RATE_LIMIT")
	if roh == "" {
		return 50
	}
	n, err := strconv.Atoi(roh)
	if err != nil || n <= 0 {
		slog.Warn("RATE_LIMIT ist keine positive Zahl — es gilt die Vorgabe 50", "wert", roh)
		return 50
	}
	return n
}
