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

// domainAntwort sagt der Anmeldeseite, was sie vor der Anmeldung über das Feld wissen muss.
type domainAntwort struct {
	// Domain ist die Domain der Schule, die der Server an einen Namen ohne @ hängt; leer ohne.
	Domain string `json:"domain"`
	// LDAP sagt, ob ein Verzeichnisdienst eingerichtet ist. Dann tippt manche Person statt der
	// E-Mail ihre LDAP-Kennung, und das Feld darf keine Adresse verlangen.
	LDAP bool `json:"ldap"`
}

// Domain liefert der Anmeldeseite die Domain der Schule und ob es LDAP gibt, bevor jemand
// angemeldet ist: Beschriftung und Hinweis am Feld richten sich danach.
func Domain() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		httpresp.SchreibeJSON(w, domainAntwort{Domain: SelbstanmeldeDomain(), LDAP: leseLDAPKonfig().istEingerichtet()})
	}
}
