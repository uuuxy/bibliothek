// Package katalog trägt die Türen des Medienkatalogs zu einem Titel und seinen Exemplaren: die
// Exemplare, die Ausleiher und die Historie eines Titels lesen, Nummer, Zustand, Eigentum und
// Standort eines Exemplars ändern, aussondern und löschen, dazu Auflagen, Signaturen,
// Systematik und Schlagworte, die Abfrage eines Titels über die ISBN, die Cover, die Geräte
// und die Bestandsbücher (Zugangs- und Abgangsbuch). Die Titelliste, die Buchmaske mit Anlegen
// und Ändern und der Import stehen im Untermodul inventur/.
package katalog

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
