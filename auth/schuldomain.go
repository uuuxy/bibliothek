package auth

import (
	"bibliothek/pkg/httpresp"
	"net/http"
	"strings"
)

// ErgaenzeSchulDomain hängt an einen Namen ohne „@" die Domain der Schule, für die Anmeldung und
// den Sammelversand an die Klassenleitungen. Ohne eingetragene Domain und bei einer Adresse mit
// „@" bleibt die Eingabe, wie sie ist: lieber eine klare Ablehnung als eine geratene Domain, und
// Adressen anderer Domains bleiben möglich.
func ErgaenzeSchulDomain(eingabe string) string {
	if eingabe == "" || strings.Contains(eingabe, "@") {
		return eingabe
	}
	domain := SelbstanmeldeDomain()
	if domain == "" {
		return eingabe
	}
	return eingabe + "@" + domain
}

// Domain liefert der Anmeldeseite die Domain der Schule, bevor jemand angemeldet ist: Der Hinweis
// am Feld nennt damit, was der Server ergänzt. Ohne Domain ist die Antwort leer.
func Domain() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		httpresp.SchreibeJSON(w, SelbstanmeldeDomain())
	}
}
