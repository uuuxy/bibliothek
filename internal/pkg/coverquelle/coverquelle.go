// Package coverquelle führt die Listen der Hosts, von denen das Programm Cover und Metadaten
// holt, baut eine geprüfte URL aus ihren geprüften Teilen neu auf und holt von dort ein Cover
// (Hole).
//
// Der Neuaufbau schließt eine Lücke zwischen Prüfen und Abrufen: Geprüft wird mit url.Parse,
// und zwei Parser lesen Zeichen wie \, @, # und ? nicht immer gleich. Ginge die rohe Eingabe an
// den HTTP-Client, könnte die Anfrage einen anderen Host erreichen als den geprüften. In der
// neuen URL ist das Schema fest HTTPS, der Host stammt aus der Liste, und vom Rufer kommen nur
// Pfad und Query; ein abweichender Port fällt dabei weg.
//
// Die Listen stehen nur hier, damit jede Stelle, die eine fremde Adresse abruft, dieselben
// Hosts erlaubt.
package coverquelle

import (
	"net/url"
	"strings"
)

// BildHosts sind die Hosts, von denen Cover-Bilder kommen dürfen.
var BildHosts = []string{
	"covers.openlibrary.org",
	"portal.dnb.de",
	"services.dnb.de",
	"www.googleapis.com",
	"openlibrary.org",
	"books.google.com",
	"books.google.de",
}

// MetadatenHosts sind die Hosts, von denen Daten zu Titel und Autor kommen dürfen. Die Liste
// ist enger als BildHosts: Von books.google.* holen die Katalogdienste nur Bilder, keine
// Datensätze.
var MetadatenHosts = []string{
	"services.dnb.de",
	"www.googleapis.com",
	"openlibrary.org",
	"covers.openlibrary.org",
	"portal.dnb.de",
}

// kanonischerHost liefert zu einem Hostnamen den gleichlautenden Eintrag der Liste oder "",
// wenn er nicht erlaubt ist.
//
// Zurück kommt der Eintrag der Liste und nicht die Eingabe: So steht im Host der neu gebauten
// URL kein Zeichen aus der Eingabe, auch wenn der Vergleich einmal Groß- und Kleinschreibung
// übergeht.
func kanonischerHost(hostname string, erlaubte []string) string {
	for _, h := range erlaubte {
		if hostname == h {
			return h
		}
	}
	return ""
}

// SichereURL prüft rohURL gegen die erlaubten Hosts und liefert eine aus geprüften Teilen neu
// gebaute HTTPS-URL; ok ist false, wenn der Host nicht erlaubt oder die URL unlesbar ist.
func SichereURL(rohURL string, erlaubte []string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(rohURL))
	if err != nil {
		return "", false
	}
	host := kanonischerHost(u.Hostname(), erlaubte)
	if host == "" {
		return "", false
	}
	sicher := "https://" + host + "/" + strings.TrimPrefix(u.EscapedPath(), "/")
	if u.RawQuery != "" {
		sicher += "?" + u.RawQuery
	}
	return sicher, true
}
