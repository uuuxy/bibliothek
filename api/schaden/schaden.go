// Package schaden trägt die Türen zu Schaden und Ersatz: einen Verlust oder Schaden melden, die
// Forderungen eines Schülers lesen und als bezahlt oder storniert erledigen, den Betrag für den
// Ersatz vorschlagen, den Bescheid über den Schadensersatz vorschlagen, erstellen, drucken und
// übergeben, dazu die Rechnung an die Eltern und den Brief zu einem einzelnen Schadensfall.
package schaden

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
