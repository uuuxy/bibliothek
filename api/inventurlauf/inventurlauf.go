// Package inventurlauf trägt die Türen der Inventur: einen Lauf über die ganze Bibliothek, eine
// Signatur oder Fach und Klasse beginnen, Exemplare scannen, den Lauf abschließen oder
// verwerfen, den Fehlbestand lesen und ein als Verlust gebuchtes Exemplar wieder aufnehmen oder
// endgültig löschen. Das Paket heißt nicht inventur: Diesen Namen trägt der Medienkatalog
// (bibliothek/inventur), und die Routenliste bindet beide ein.
package inventurlauf

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
