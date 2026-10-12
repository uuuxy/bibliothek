// Package benutzer trägt die Türen zu „Benutzer & Rechte": die Konten des Personals lesen,
// anlegen, ändern und löschen und die Matrix, die je Rolle nennt, was sie darf. Wer ein
// Admin-Konto vergeben, ändern oder sich selbst herabstufen will, scheitert an den Regeln dieser
// Türen.
package benutzer

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
