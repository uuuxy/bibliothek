// Package druck trägt die Türen des Druck-Centers: die Suche nach einem Titel und die Etiketten
// aller seiner Exemplare oder eines Druckauftrags als PDF, die Liste der Exemplare ohne Etikett
// mit ihrem Zähler, der Vermerk „gedruckt" und sein Rückweg, der Stichtag für den Altbestand,
// die Etiketten für Schüler, das Design der Ausweise und ein Strichcode als Bild.
package druck

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
