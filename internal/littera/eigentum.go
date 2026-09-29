package littera

import (
	"strings"

	"bibliothek/repository"
)

// Der Eigentumsvermerk aus Littera (docs/OFFEN.md 4.24, entschieden am 29.09.2026).
//
// Littera führt das Eigentum je Exemplar als Freitext mit Wertehilfe. In der Medienliste vom
// 12.06.2026 stehen genau acht Schreibweisen an 16.878 von 67.109 Exemplaren; die übrigen
// tragen keinen Vermerk. Übernommen wird nur, was in dieser Liste steht („feste Liste"): Ein
// Freitext kann alles enthalten — in der Sicherung von 2010 steht an einem Exemplar ein
// Personenname —, und ein Wert, den niemand gesehen hat, gehört nicht in den Bestand.
//
// Zwei Vermerke sind eindeutig und setzen das Eigentum (buecher_exemplare.eigentum, Migration
// 150): „Land Hessen" und der Schulträger. Die übrigen fünf bleiben ohne Zuordnung, bis die
// Schule sagt, wem diese Bücher gehören (docs/OFFEN.md 4.24); bis dahin gilt für sie die
// Faustregel aus dem Titel. Ihr Wortlaut kommt trotzdem mit, denn die Übernahme läuft einmal.
type vermerkZuordnung struct {
	// Wortlaut ist die Schreibweise, die am Exemplar aufgehoben wird. „land hessen" (33 Mal)
	// wird dabei zu „Land Hessen".
	Wortlaut string
	// Eigentum ist repository.MittelLand oder repository.MittelSchultraeger, leer heißt:
	// keine Zuordnung, es gilt die Faustregel.
	Eigentum string
}

// vermerkeLittera ist die feste Liste, Schlüssel in Kleinschreibung. Die Zahlen sind die der
// Medienliste vom 12.06.2026.
var vermerkeLittera = map[string]vermerkZuordnung{
	"land hessen":         {"Land Hessen", repository.MittelLand},             // 13.303
	"hochtaunuskreis":     {"Hochtaunuskreis", repository.MittelSchultraeger}, // 2.942
	"philipp-reis-schule": {"Philipp-Reis-Schule", ""},                        // 355
	"bibliothek":          {"Bibliothek", ""},                                 // 157
	"förderverein":        {"Förderverein", ""},                               // 86
	"info schulprojekt":   {"Info Schulprojekt", ""},                          // 31
	"dauerleihgabe":       {"Dauerleihgabe", ""},                              // 4
}

// zuordnungAusVermerk liefert die Zuordnung eines Vermerks. bekannt ist false, wenn der
// Vermerk nicht leer ist und nicht in der festen Liste steht — dann wird nichts davon
// übernommen. Ein leerer Vermerk ist bekannt und ergibt eine leere Zuordnung.
func zuordnungAusVermerk(vermerk string) (z vermerkZuordnung, bekannt bool) {
	schluessel := strings.ToLower(strings.Join(strings.Fields(vermerk), " "))
	if schluessel == "" {
		return vermerkZuordnung{}, true
	}
	z, bekannt = vermerkeLittera[schluessel]
	return z, bekannt
}
