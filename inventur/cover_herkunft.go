package inventur

import (
	"net/url"
	"strings"
)

// erlaubteCoverHosts ist die EINE Liste der Bezugsquellen für Buchcover.
//
// Sie stand bis zum 04.08.2026 nur inline im Download-Pfad (ladeCoverBytes). Das
// manuelle Cover-Update (handleUpdateCover) prüfte dagegen bloß, ob die URL mit
// "https://" beginnt — jede beliebige Fremd-URL ließ sich damit dauerhaft in
// buecher_titel.cover_url schreiben. Serverseitig geladen wurde sie zwar nie (der
// Download-Pfad hat seine Allowlist), aber jeder Browser, der einen Katalog, eine
// Trefferliste oder den Monitor öffnet, ruft sie als <img src> ab: Der fremde Server
// sieht IP, User-Agent und Referrer jedes Arbeitsplatzes und kann das Bild jederzeit
// austauschen. Beide Pfade fragen jetzt dieselbe Liste.
//
// Die Liste ist absichtlich identisch mit erlaubteCoverHosts in api/image_caching.go
// (öffentlicher Cover-Proxy). Wird sie hier ergänzt, gehört sie dort mit ergänzt.
var erlaubteCoverHosts = []string{
	"covers.openlibrary.org",
	"portal.dnb.de",
	"services.dnb.de",
	"www.googleapis.com",
	"openlibrary.org",
	"books.google.com",
	"books.google.de",
}

// IstErlaubteCoverHerkunft prüft, ob eine Cover-URL von einem der zugelassenen Hosts
// stammt. Der Vergleich läuft über url.Hostname() und nicht über Präfixe: "https://
// covers.openlibrary.org.angreifer.example/x.jpg" und "https://covers.openlibrary.org
// @angreifer.example/x.jpg" tragen beide den erlaubten Namen im String, zeigen aber
// auf einen fremden Host.
func IstErlaubteCoverHerkunft(rohURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rohURL))
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	for _, erlaubt := range erlaubteCoverHosts {
		if host == erlaubt {
			return true
		}
	}
	return false
}
