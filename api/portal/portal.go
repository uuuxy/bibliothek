// Package portal trägt die Türen, über die das Kollegium und die Öffentlichkeit die Bibliothek
// erreichen: „Mein Portal" mit der Suche im Katalog, den Reservierungen von Klassensätzen und
// den Meldungen an die Bibliothek, dazu der öffentliche Katalog und die Folien für den Monitor.
// Die Bibliothek liest hier die offenen Reservierungen und Meldungen und erledigt sie.
package portal

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
