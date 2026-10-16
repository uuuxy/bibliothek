// Package theke trägt die Türen der Theke: den Scan mit Ausleihe und Rückgabe, das Nachbuchen
// der Warteschlange ohne Netz mit seinen Meldungen und der Liste der Buchnummern, die Suche der
// Omnibox, Verlängern und Frist von Hand an einer Ausleihe und die Vormerkungen.
package theke

import (
	"time"

	"bibliothek/api/tuer"
)

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
	// IdempotenzWartezeit: wie lange eine Anfrage auf die Antwort einer laufenden mit
	// demselben Schlüssel wartet (erlangeIdempotenz); 0 = 3 s. Nur Tests setzen es.
	IdempotenzWartezeit time.Duration
}

// meldungVerlaengerungGesperrt antwortet auf die Verlängerung einer gesperrten Ausleihe; mit dem
// Recht view_students steht der Grund der Sperre dahinter.
const meldungVerlaengerungGesperrt = "Die Ausleihe ist gesperrt und lässt sich nicht verlängern"
