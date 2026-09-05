package api

import (
	"fmt"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Die Abgängerliste hat eine Saison. Sie zeigt die Abschlussklassen mit offenen Büchern —
// von August bis April wären das schlicht alle 9H, 10R und 13 mit ihren laufenden
// Schulbüchern, eine Liste ohne Handlungswert. Gebraucht wird sie zum Einsammeln vor der
// Entlassung: Der LMF-Plan der Schule legt die Rückgabe der Abschlussklassen auf Ende
// Juni/Anfang Juli, die Vorbereitung beginnt im Mai (Peter, 05.09.2026: „vielleicht wäre
// der Mai gut"). Das Schuljahr endet in Hessen am 31. Juli; danach gehören die Abgänger
// der LUSD und dem Mahnwesen.
//
// Fester Wert, keine Einstellung: Ein Fenster, das jemand erst setzen muss, ist ein Feature,
// das an einer ungesetzten Einstellung hängt. Wer die Liste außerhalb der Saison öffnet,
// bekommt den Hinweis mit beiden Daten — keine leere Seite.
const (
	abgaengerSaisonVon = time.May
	abgaengerSaisonBis = time.July
)

// AbgaengerFenster ist der Teil der Antwort von GET /api/abgaenger, der der Oberfläche
// sagt, WARUM die Liste leer ist: außerhalb der Saison oder weil alle entlastet sind.
type AbgaengerFenster struct {
	Offen bool   `json:"offen"`
	Von   string `json:"von"` // „01.05."
	Bis   string `json:"bis"` // „31.07."
}

// abgaengerFensterFuer bewertet einen Zeitpunkt in der Schulzeitzone. Die Grenzen werden
// aus den Konstanten gerechnet, damit Anzeige und Regel nicht auseinanderlaufen können.
func abgaengerFensterFuer(jetzt time.Time) AbgaengerFenster {
	monat := jetzt.In(schulzeit.Zone()).Month()
	// Tag 0 des Folgemonats ist der letzte Tag des Saisonmonats.
	letzter := time.Date(2001, abgaengerSaisonBis+1, 0, 0, 0, 0, 0, time.UTC)
	return AbgaengerFenster{
		Offen: monat >= abgaengerSaisonVon && monat <= abgaengerSaisonBis,
		Von:   fmt.Sprintf("01.%02d.", int(abgaengerSaisonVon)),
		Bis:   letzter.Format("02.01."),
	}
}

// abgaengerAusserhalbDerSaison ist die Meldung für Druck und Versand außerhalb des Fensters.
func abgaengerAusserhalbDerSaison(f AbgaengerFenster) error {
	return fmt.Errorf("die Abgängerliste zeigt die Abschlussklassen vom %s bis %s — außerhalb dieser Zeit gibt es keine Kontoauszüge", f.Von, f.Bis)
}

// jetzt ist die Uhr des Servers: in Tests setzbar (Server.Uhr), sonst die Schulzeit.
func (s *Server) jetzt() time.Time {
	if s.Uhr != nil {
		return s.Uhr()
	}
	return schulzeit.Jetzt()
}
