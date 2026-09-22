package inventur

import (
	"bibliothek/pkg/lmf"
	"regexp"
	"strings"

	"bibliothek/pkg/isbnutil"
)

// konvertiereISBN10zu13 wandelt eine 10-stellige ISBN in das moderne 13-stellige Format um.
// Dies ist wichtig für APIs wie OpenLibrary oder DNB, die oft 13-stellig erwarten.
func konvertiereISBN10zu13(isbn string) string {
	var clean [10]byte
	idx := 0

	for i := 0; i < len(isbn); i++ {
		c := isbn[i]
		if c != '-' {
			if idx < 10 {
				clean[idx] = c
			}
			idx++
		}
	}

	if idx != 10 {
		return isbnutil.CleanISBN(isbn)
	}

	summe := 38 // 9*1 + 7*3 + 8*1
	faktor := 3
	for i := 0; i < 9; i++ {
		ziffer := int(clean[i] - '0')
		summe += ziffer * faktor
		if faktor == 3 {
			faktor = 1
		} else {
			faktor = 3
		}
	}

	pruefsumme := (10 - (summe % 10)) % 10

	var res [13]byte
	res[0] = '9'
	res[1] = '7'
	res[2] = '8'
	copy(res[3:12], clean[:9])
	res[12] = byte(pruefsumme + '0')

	return string(res[:])
}

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
