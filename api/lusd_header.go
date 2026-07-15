package api

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

// nonAlnum entfernt beim Normalisieren alle Trennzeichen (Leerzeichen, Bindestrich,
// Unterstrich, Punkt …), damit "Schüler_Vorname", "schueler vorname" und
// "SchuelerVorname" identisch matchen.
var nonAlnum = regexp.MustCompile("[^a-z0-9]+")

// normalizeHeader vereinheitlicht eine CSV-Kopfzelle für den Abgleich: Kleinschreibung,
// deutsche Umlaute/ß aufgelöst, danach alle Nicht-Alphanumerik entfernt. So matchen
// "Straße" == "strasse" und "E-Mail" == "email".
func normalizeHeader(h string) string {
	s := strings.ToLower(strings.TrimSpace(h))
	s = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss").Replace(s)
	return nonAlnum.ReplaceAllString(s, "")
}

// lusdFieldAliases bildet jedes logische Feld auf die akzeptierten Kopfzeilen-
// Schreibweisen ab. Deckt beide LUSD-Export-Stile ab: mit Tabellen-Präfix
// (Individueller Bericht, z. B. "Schueler_Vorname") und ohne (Standardexport,
// z. B. "Vorname"). Erweiterbar ohne Datenbank-/Migrationsaufwand.
var lusdFieldAliases = map[string][]string{
	lusdColID:           {"lusd_id", "schueler_id", "lusdid"},
	lusdColVorname:      {"vorname", "schueler_vorname"},
	lusdColNachname:     {"nachname", "schueler_nachname"},
	lusdColKlasse:       {"klasse", "klassenbezeichnung", "klassen_klassenbezeichnung"},
	lusdColGeburtsdatum: {"geburtsdatum", "schueler_geburtsdatum"},
	lusdColStrasse:      {"strasse", "schueler_strasse", "anschrift_strasse"},
	lusdColHausnummer:   {"hausnummer", "schueler_hausnummer", "anschrift_hausnummer"},
	lusdColPLZ:          {"plz", "postleitzahl", "schueler_plz", "anschrift_plz"},
	lusdColOrt:          {"ort", "wohnort", "schueler_ort", "anschrift_ort"},
	lusdColElternEmail:  {"eltern_email", "email", "ansprechpartner_email", "erziehungsberechtigte_email", "erziehungsberechtigter_email"},
}

// lusdPflichtspalten müssen im Export vorhanden sein. Die LUSD-ID ist der stabile
// Schlüssel für Klassenwechsel-/Abgänger-Abgleich über Importe hinweg.
var lusdPflichtspalten = []string{lusdColID, lusdColVorname, lusdColNachname, lusdColKlasse}

// lusdOptionaleSpalten sind die Adress-/Kontaktspalten (dürfen fehlen).
var lusdOptionaleSpalten = []string{lusdColStrasse, lusdColHausnummer, lusdColPLZ, lusdColOrt, lusdColElternEmail}

// lusdHeaderLookup mappt jede normalisierte Alias-Schreibweise auf ihr logisches Feld.
var lusdHeaderLookup = buildLusdHeaderLookup()

func buildLusdHeaderLookup() map[string]string {
	lookup := make(map[string]string)
	for canonical, aliases := range lusdFieldAliases {
		for _, alias := range aliases {
			lookup[normalizeHeader(alias)] = canonical
		}
	}
	return lookup
}

// lusdHeaderMap ordnet die erkannten logischen Spalten ihren Zeilen-Indizes zu und
// prüft, ob alle Pflichtspalten vorhanden sind. Unbekannte Spalten werden ignoriert.
func lusdHeaderMap(headers []string) (map[string]int, error) {
	headerMap := make(map[string]int)
	for idx, h := range headers {
		canonical, ok := lusdHeaderLookup[normalizeHeader(h)]
		if !ok {
			continue
		}
		if _, bereitsGemappt := headerMap[canonical]; bereitsGemappt {
			continue // erste passende Spalte gewinnt
		}
		headerMap[canonical] = idx
	}

	for _, col := range lusdPflichtspalten {
		if _, exists := headerMap[col]; !exists {
			return nil, fmt.Errorf("pflichtspalte '%s' fehlt in der CSV-Kopfzeile — ist das die richtige LUSD-Exportdatei?", col)
		}
	}

	logErkannteOptionaleSpalten(headerMap)
	return headerMap, nil
}

// logErkannteOptionaleSpalten macht sichtbar, welche Adressspalten erkannt wurden —
// so bleibt ein Tippfehler im Header nie unbemerkt („still leer importiert").
func logErkannteOptionaleSpalten(headerMap map[string]int) {
	var found, missing []string
	for _, col := range lusdOptionaleSpalten {
		if _, ok := headerMap[col]; ok {
			found = append(found, col)
		} else {
			missing = append(missing, col)
		}
	}
	if len(found) > 0 {
		log.Printf("LUSD-Import: Adress-/Kontaktspalten erkannt: %v — nicht gefunden: %v", found, missing)
	}
}
