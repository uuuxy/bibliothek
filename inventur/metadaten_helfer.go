package inventur

import (
	"bibliothek/pkg/isbnutil"
	"regexp"
	"strconv"
	"strings"
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
		return isbnutil.Clean(isbn)
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

// kategorisierungFachZuweisungen ordnet Suchbegriffe den Fächern zu.
var kategorisierungFachZuweisungen = map[string]string{
	"mathematik": "Mathe", "algebra": "Mathe", "geometrie": "Mathe", "mathe": "Mathe",
	"english": "Englisch", "englisch": "Englisch", "grammar": "Englisch",
	"französisch": langFrench, "franzosisch": langFrench, "français": langFrench, "francais": langFrench, "fremdsprache": langFrench,
	"deutsch": "Deutsch", "grammatik": "Deutsch", "literatur": "Deutsch",
	"geschichte": "Geschichte", "histor": "Geschichte", "europa": "Geschichte",
	"biologie": "Biologie", "chemie": "Chemie", "physik": "Physik",
	"geographie": "Geographie", "erdkunde": "Geographie", "politik": "Politik",
	"informatik": "Informatik", "musik": "Musik", "arbeitslehre": "Arbeitslehre",
}

var zielgruppenAlterRegex = regexp.MustCompile(`\d{1,2}`)

// leiteBibKategorieAb übersetzt die DNB-Genre-Begriffe (MARC 655, z. B.
// "Kinderbücher bis 11 Jahre", "Jugendbücher ab 12 Jahre") in die
// Signatur-Kategorien der Schülerbücherei — dieselben Werte wie
// bibKategorien in signatur_optionen.js, damit der Vorschlag direkt als
// "BIB {Kategorie}" auf dem Rücken-Etikett landen kann. Fehlt ein
// Genre-Treffer, entscheidet die Altersangabe der Zielgruppe (DNB-Grenze:
// bis 11 Jahre Kinderbuch, ab 12 Jahre Jugendbuch).
func leiteBibKategorieAb(genres []string, zielgruppe string) string {
	alle := strings.ToLower(strings.Join(genres, " | "))
	switch {
	case strings.Contains(alle, "manga"):
		return "Manga"
	case strings.Contains(alle, "comic"):
		return "Comic"
	case strings.Contains(alle, "jugendbuch"), strings.Contains(alle, "jugendbücher"):
		return "Jugendbuch"
	case strings.Contains(alle, "kinderbuch"), strings.Contains(alle, "kinderbücher"):
		return "Kinderbuch"
	}

	if treffer := zielgruppenAlterRegex.FindString(zielgruppe); treffer != "" {
		if alter, err := strconv.Atoi(treffer); err == nil {
			if alter >= 12 {
				return "Jugendbuch"
			}
			return "Kinderbuch"
		}
	}
	return ""
}

// automatischeKategorisierung liest Titel und Untertitel eines Buches und
// versucht mit Regex-Wörterbüchern ein passendes Fach und die Klassenstufe
// zu extrahieren. Dies beschleunigt das Anlegen von Schulbüchern ungemein.
func automatischeKategorisierung(titel, untertitel string) (fach, klassenStufe string) {
	text := strings.ToLower(strings.TrimSpace(titel + " " + untertitel))
	fach = ""
	klassenStufe = ""

	for schluessel, wert := range kategorisierungFachZuweisungen {
		if strings.Contains(text, schluessel) {
			fach = wert
			break
		}
	}

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
