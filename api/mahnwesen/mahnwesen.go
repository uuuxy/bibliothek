// Package mahnwesen trägt die Türen des Mahnwesens und der Abgänger: die überfälligen Ausleihen
// je Klasse, Mahnlisten und Mahnbriefe als PDF und als Mail an die Klassenleitungen, deren
// Adressen je Klasse, dazu die Liste der Abschlussklassen mit offenen Büchern in der Saison
// und ihre Kontoauszüge als PDF und als Mail.
package mahnwesen

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
