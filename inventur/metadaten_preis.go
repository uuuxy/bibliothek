package inventur

import (
	"regexp"
	"strconv"
	"strings"
)

// MARC21 020 $c ("Terms of availability") enthält bei DNB-Datensätzen den Ladenpreis.
// Das Feld ist Fliesstext und keine Zahl; echte Antworten der DNB sehen so aus:
//
//	"Pp. : EUR 27.00, sfr 46.90"      → 27.00
//	"kart. : EUR 24,95"               → 24.95
//	"Pp. : DM 35.20"                  → kein Vorschlag
//	"Pp. : DM 42.00, EUR 21.47"       → kein Vorschlag
//
// Die letzten beiden Zeilen sind der Grund für die DM-Regel weiter unten.
var (
	preisEUR   = regexp.MustCompile(`(?i)\bEUR\s*([0-9]+(?:[.,][0-9]{1,2})?)`)
	enthaeltDM = regexp.MustCompile(`(?i)\bDM\s*[0-9]`)
)

// preisAusVerfuegbarkeit liest einen Euro-Betrag aus MARC21 020 $c.
// 0 bedeutet: kein brauchbarer Preis — der Aufrufer schlägt dann gar nichts vor.
//
// Zwei Regeln, beide an echten DNB-Antworten entstanden:
//
//  1. NUR EURO. Ein reiner DM-Betrag ist als Preis wertlos.
//
//  2. NENNT DER SATZ D-MARK, GILT AUCH SEIN EURO-BETRAG NICHT. Solche Sätze stammen aus
//     der Umstellungszeit, der Euro-Wert ist die Umrechnung eines Preises von damals
//     ("DM 42.00, EUR 21.47" — derselbe Titel kostet heute laut neuerem Satz 27.00).
//     Ein plausibel aussehender falscher Preis ist schlimmer als gar keiner: Er wird
//     übernommen, ohne dass jemand stutzt.
func preisAusVerfuegbarkeit(verfuegbarkeit string) float64 {
	if verfuegbarkeit == "" || enthaeltDM.MatchString(verfuegbarkeit) {
		return 0
	}
	treffer := preisEUR.FindStringSubmatch(verfuegbarkeit)
	if treffer == nil {
		return 0
	}
	// Deutsche Schreibweise ("24,95") wie englische ("27.00") — beide kommen vor.
	betrag, err := strconv.ParseFloat(strings.Replace(treffer[1], ",", ".", 1), 64)
	if err != nil || betrag <= 0 {
		return 0
	}
	return betrag
}
