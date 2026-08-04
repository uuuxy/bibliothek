// Package coverquelle hält die Host-Allowlists für Buchcover und Buchmetadaten und
// baut geprüfte URLs aus validierten Teilen neu auf.
//
// Warum das Neubauen nötig ist, obwohl der Hostname schon geprüft wurde: Geprüft wird
// mit url.Parse, angefragt wurde bisher der ROHE Eingabe-String. Zwischen beiden kann
// eine Differenz liegen — verschiedene Parser sind sich über Zeichen wie \, @, #, ?
// und über die Behandlung von Backslashes nicht immer einig ("Parsing Differential").
// Setzt der HTTP-Client die URL anders zusammen als der Prüfer sie gelesen hat, geht
// die Anfrage an einen anderen Host als den freigegebenen. Die Prüfung hätte dann
// zugestimmt, ohne dass sie dasselbe Ziel gesehen hat.
//
// Der Neuaufbau beseitigt das: Schema fest HTTPS, Host aus der ALLOWLIST-Konstante
// (nicht aus der Eingabe), und nur Pfad und Query stammen vom Aufrufer — die können
// weder Host noch Schema noch Port beeinflussen. Ein "https://covers.openlibrary.org
// :2222/x.jpg" verliert damit auch seinen Port, ein "http://…" wird zu HTTPS.
//
// Die Listen standen bis zum 04.08.2026 dreifach im Code (api/image_caching.go,
// inventur/cover_downloader.go, inventur/metadaten_client.go), zusammengehalten von
// einem Kommentar, der zur Pflege aufrief. Eine Sicherheitsliste, deren Gleichheit nur
// ein Kommentar behauptet, driftet — sie tat es bereits: Die Metadaten-Liste kannte
// books.google.* nicht.
package coverquelle

import (
	"net/url"
	"strings"
)

// CoverHosts sind die Bezugsquellen für Cover-BILDER.
var CoverHosts = []string{
	"covers.openlibrary.org",
	"portal.dnb.de",
	"services.dnb.de",
	"www.googleapis.com",
	"openlibrary.org",
	"books.google.com",
	"books.google.de",
}

// MetadatenHosts sind die Quellen für Titel-/Autorendaten. Bewusst enger als
// CoverHosts: Von books.google.* holt das Inventur-Modul nur Bilder, keine Datensätze.
var MetadatenHosts = []string{
	"services.dnb.de",
	"www.googleapis.com",
	"openlibrary.org",
	"covers.openlibrary.org",
	"portal.dnb.de",
}

// kanonischerHost bildet einen Hostnamen auf die passende Allowlist-Konstante ab;
// "" heißt: nicht erlaubt.
//
// Zurückgegeben wird bewusst das LISTEN-ELEMENT und nicht die Eingabe, obwohl beide
// bei einem Treffer gleich sind: Damit kann in der neu gebauten URL garantiert kein
// Zeichen aus der Eingabe im Host-Teil landen, auch wenn der Vergleich später einmal
// unschärfer wird (Groß-/Kleinschreibung, Unicode-Normalisierung).
func kanonischerHost(hostname string, erlaubte []string) string {
	for _, h := range erlaubte {
		if hostname == h {
			return h
		}
	}
	return ""
}

// SichereURL prüft rohURL gegen erlaubte und liefert eine aus geprüften Teilen neu
// gebaute HTTPS-URL. ok=false heißt: Host nicht erlaubt oder URL unlesbar.
func SichereURL(rohURL string, erlaubte []string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rohURL))
	if err != nil {
		return "", false
	}
	host := kanonischerHost(parsed.Hostname(), erlaubte)
	if host == "" {
		return "", false
	}
	sicher := "https://" + host + "/" + strings.TrimPrefix(parsed.EscapedPath(), "/")
	if parsed.RawQuery != "" {
		sicher += "?" + parsed.RawQuery
	}
	return sicher, true
}

// IstErlaubt meldet, ob rohURL von einem zugelassenen Host stammt — für Stellen, die
// eine URL nur SPEICHERN und nicht selbst abrufen (manuelles Cover-Update).
func IstErlaubt(rohURL string, erlaubte []string) bool {
	_, ok := SichereURL(rohURL, erlaubte)
	return ok
}
