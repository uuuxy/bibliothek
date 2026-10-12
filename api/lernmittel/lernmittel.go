// Package lernmittel trägt die Türen des LMF-Plans: wann welche Klasse ihre Schulbücher abgibt
// und bekommt. Die Bibliothek schreibt den Plan als Reihenfolge, das Kollegium liest die Termine
// im Portal und als PDF.
package lernmittel

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
