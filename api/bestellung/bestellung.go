// Package bestellung trägt die Türen des Bestellwesens: die Vorschläge zum Nachbestellen und
// ihre Liste als PDF, die Suche nach Titeln für eine Bestellung, die Bestellung mit ihrer Mail
// an den Lieferanten, Zulauf und Wareneingang, die Bestellhistorie mit Übersicht, Einzelheiten
// und Bericht, die Lieferanten, die Bestätigung im Haus und über den Link des Lieferanten mit
// seinen Etiketten, das erneute Senden der Mail und der Topf einer Bestellung.
package bestellung

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
