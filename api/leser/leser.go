// Package leser trägt die Türen der Leserdatei: die Liste der Leser mit Klassen und
// Jahrgängen, die Leserakte, Anlegen, Ändern, Sperren, Löschen mit Papierkorb und das
// Zusammenführen zweier Datensätze, das Passbild, den Abgleich mit der LUSD und die Versetzung
// zum Schuljahreswechsel, die Auskunft nach Art. 15 DSGVO und den Kontoauszug.
package leser

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
