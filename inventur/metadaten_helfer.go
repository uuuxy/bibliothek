package inventur

import (
	"bibliothek/pkg/lmf"
	"regexp"
	"strings"
)

var (
	stufenRegex      = regexp.MustCompile(`(?i)(klasse|band|stufe|teil|level|jahrgangsstufe)\s*(\d{1,2})`)
	klassenZahlRegex = regexp.MustCompile(`\b([5-9]|1[0-3])\b`)
)

// automatischeKategorisierung liest Titel und Untertitel eines Buches und
// versucht mit Regex-Wörterbüchern ein passendes Fach und die Klassenstufe
// zu extrahieren. Dies beschleunigt das Anlegen von Schulbüchern ungemein.
//
// Das Fach kommt aus der einen Stichwortliste in pkg/lmf — bis zum 02.09.2026 hatte
// dieser Pfad eine eigene Map („Mathe"), der Excel-Import eine zweite („Mathematik"),
// und die Systematik führte beide Fächer nebeneinander.
func automatischeKategorisierung(titel, untertitel string) (fach, klassenStufe string) {
	text := strings.ToLower(strings.TrimSpace(titel + " " + untertitel))
	fach = lmf.FachAusText(text)
	klassenStufe = ""

	if match := stufenRegex.FindStringSubmatch(text); len(match) == 3 {
		klassenStufe = match[2]
	} else {
		// Suche nach Zahl zwischen 5 und 13, falls es offenbar ein Schulbuch ist
		if match := klassenZahlRegex.FindStringSubmatch(text); len(match) == 2 {
			klassenStufe = match[1]
		}
	}
	return fach, klassenStufe
}
