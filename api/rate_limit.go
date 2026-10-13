package api

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/ratelimit"
)

// RateLimitMiddleware begrenzt Anfragen je Client-IP auf limit pro Zeitfenster.
// Ausgenommen sind Pfade, bei denen ein einzelner Seitenaufruf dutzende Anfragen
// erzeugt — die Begründung dafür steht unten an der Ausnahmeliste.
func RateLimitMiddleware(limit int) func(http.Handler) http.Handler {
	limiter := ratelimit.New(limit)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bilder und Uploads vom Rate-Limiter ausschließen, da ein Seitenaufruf oft dutzende Bilder gleichzeitig lädt.
			// /api/barcode gehört genau in diese Klasse: die Etiketten-Vorschau (LabelPreview.svelte) rendert einen
			// kompletten A4-Bogen (Standard 52 = 52 Etiketten) und feuert pro Etikett ein <img src="/api/barcode…">
			// nahezu gleichzeitig ab — das sprengt sonst das 50-Requests/s-Bucket und liefert 429 (authentifiziert,
			// view_books, winzige PNGs; seit dem 30.09.2026 legt der Browser sie nicht mehr ab, jeder Seitenaufruf
			// holt sie also neu — internal/middleware/security.go). Exakter Pfad-Match, damit /api/barcode/next (JSON)
			// weiter limitiert bleibt.
			// /events (SSE) ist eine langlebige Verbindung, die bei jedem (Re-)Connect sonst einen Token verbraucht
			// und flaky Clients unnötig ausbremst.
			if strings.HasPrefix(r.URL.Path, "/api/images/cover") || strings.HasPrefix(r.URL.Path, "/uploads/") || r.URL.Path == "/api/barcode" || r.URL.Path == "/events" {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientip.AusAnfrage(r)
			if !limiter.Allow(ip) {
				apierrors.SendHTTPError(w, http.StatusTooManyRequests, errors.New("rate limit exceeded"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitAusUmgebung liefert das Anfragen-pro-Sekunde-Budget je IP.
//
// Vorgabe 50 — der Produktionswert, unveraendert. Ueberschreibbar per RATE_LIMIT nur,
// weil die E2E-Suite ein Sonderfall ist, den es in der Praxis nicht gibt: EIN Browser
// auf EINER IP feuert ohne menschliche Pausen durch neunundneunzig Flows. Jeder
// Seitenaufruf loest mehrere API-Anfragen aus, und seit die lokale Datenbank von 5.494
// auf 2 Bestellungen geschrumpft ist, laden die Seiten so schnell, dass mehr davon in
// dieselbe Sekunde fallen.
//
// Der Schaden war nicht der 429 — der ist richtig. Der Schaden war, dass das Frontend
// ihn schluckte und der Warnhinweis ueber dem Bestellwesen verschwand (siehe
// orderStore.loadKonfiguration). Das ist getrennt behoben. Diese Schraube macht nur den
// Testlauf deterministisch, statt ihn an einer Zufaelligkeit scheitern zu lassen.
//
// Bewusst NICHT an APP_ENV gekoppelt: Eine Grenze, die sich je nach Umgebung selbst
// verstellt, ist in dem Moment weg, in dem jemand APP_ENV falsch setzt. So steht der
// abweichende Wert sichtbar in docker-compose.local.yml und nirgends sonst.
func rateLimitAusUmgebung() int {
	roh := os.Getenv("RATE_LIMIT")
	if roh == "" {
		return 50
	}
	n, err := strconv.Atoi(roh)
	if err != nil || n <= 0 {
		log.Printf("RATE_LIMIT=%q ist keine positive Zahl — es gilt die Vorgabe 50", roh)
		return 50
	}
	return n
}
