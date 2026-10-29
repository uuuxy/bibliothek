// Package katalog trägt die Türen des Medienkatalogs: die Titelliste, die Buchmaske mit Anlegen
// und Ändern, das Löschen markierter Titel, der Listenimport und der CSV-Export, die Abfrage
// eines Titels über die ISBN, die Cover samt Upload und Auslieferung, die Klassensätze, dazu
// je Titel und Exemplar: Exemplare, Ausleiher und Historie lesen, Nummer, Zustand, Eigentum und
// Standort eines Exemplars ändern, die nächste freie Nummer nennen, aussondern und löschen,
// Auflagen, Signaturen, Systematik und Schlagworte, die Geräte und die Bestandsbücher (Zugangs-
// und Abgangsbuch).
package katalog

import (
	"bibliothek/api/tuer"
	"bibliothek/internal/katalogdienste"
	"bibliothek/repository"
)

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
	// cover holt Cover von fremden Quellen; der Nullwert ist der Betrieb.
	cover coverAbruf
	// Katalogdienste fragt DNB, Google Books und OpenLibrary. Der Server teilt einen Client mit
	// allen Türen, damit die Verbindungen zu den Diensten wiederverwendet werden.
	Katalogdienste *katalogdienste.MetadatenClient
}

// titelAbfragen sind die Abfragen des Medienkatalogs am Pool der Basis.
func (t *Tueren) titelAbfragen() *repository.TitelAbfragen {
	return repository.NewTitelAbfragen(t.DB.Pool)
}

// katalogdienste ist der Client, den der Server stellt; ohne ihn, etwa in einem Test, ein eigener.
func (t *Tueren) katalogdienste() *katalogdienste.MetadatenClient {
	if t.Katalogdienste != nil {
		return t.Katalogdienste
	}
	return katalogdienste.NewMetadatenClient()
}

// Sätze der Türen zum Exemplar, die mehrere Türen dieses Pakets ausgeben.
const (
	meldungExemplarVerliehen     = "Das Exemplar ist noch verliehen. Bitte zuerst die Rückgabe buchen."
	meldungExemplarNichtGefunden = "Exemplar nicht gefunden oder schon ausgebucht"
	// meldungExemplarInzwischenGeaendert steht im Dialog der Buchakte; die Seite zeigt den
	// neuen Status erst nach dem Neuladen.
	meldungExemplarInzwischenGeaendert = "Ein anderer Platz hat den Status dieses Exemplars inzwischen geändert. Bitte die Buchakte neu laden und dann noch einmal speichern."
)

// Sätze, hinter die die Tür den Grund einer Ablehnung aus repository setzt (repository.MitGrund).
const (
	meldungEigentumNichtGeaendert      = "Eigentum nicht geändert"
	meldungStandortNichtGeaendert      = "Standort nicht geändert"
	meldungSchlagworteNichtGespeichert = "Schlagworte nicht gespeichert"
	meldungSchlagwortNichtGeaendert    = "Schlagwort nicht geändert"
	meldungSchlagwortRegel             = "Nicht möglich"
	meldungAuflagenNichtGeaendert      = "Auflagen nicht geändert"
)
