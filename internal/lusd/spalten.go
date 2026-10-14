package lusd

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

// trennzeichen entfernt beim Normalisieren alle Trennzeichen (Leerzeichen, Bindestrich,
// Unterstrich, Punkt …), damit "Schüler_Vorname", "schueler vorname" und
// "SchuelerVorname" identisch matchen.
var trennzeichen = regexp.MustCompile("[^a-z0-9]+")

// normiereKopf vereinheitlicht eine CSV-Kopfzelle für den Abgleich: Kleinschreibung,
// deutsche Umlaute/ß aufgelöst, danach alle Nicht-Alphanumerik entfernt. So matchen
// "Straße" == "strasse" und "E-Mail" == "email".
func normiereKopf(zelle string) string {
	s := strings.ToLower(strings.TrimSpace(zelle))
	s = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss").Replace(s)
	return trennzeichen.ReplaceAllString(s, "")
}

// kopfSchreibweisen bildet jedes logische Feld auf die akzeptierten Kopfzeilen-
// Schreibweisen ab. Deckt beide LUSD-Export-Stile ab: mit Tabellen-Präfix
// (Individueller Bericht, z. B. "Schueler_Vorname") und ohne (Standardexport,
// z. B. "Vorname"). Erweiterbar ohne Datenbank-/Migrationsaufwand.
//
// Dritter Stil (26.08.2026, eine Klassenliste mit LUSD-Feldkürzeln aus der Schule —
// kein LUSD-Export, aber dieselben Kürzel): SLR_ (Schüler), KLA_ (Klasse): SLR_Nachname,
// SLR_Vorname, SLR_Strasse, SLR_PLZ, SLR_ORT, KLA_Klassennamen. Ohne diese Aliase bricht
// der Import an solchen Köpfen mit „Pflichtspalte 'vorname' fehlt" ab.
var kopfSchreibweisen = map[string][]string{
	spalteID:           {"lusd_id", "schueler_id", "lusdid", "slr_id", "slr_schuelerid"},
	spalteVorname:      {"vorname", "schueler_vorname", "slr_vorname"},
	spalteNachname:     {"nachname", "schueler_nachname", "slr_nachname"},
	spalteKlasse:       {"klasse", "klassenbezeichnung", "klassen_klassenbezeichnung", "kla_klassennamen", "kla_klassenname", "kla_klassenbezeichnung", "schueler_klasse"},
	spalteGeburtsdatum: {"geburtsdatum", "schueler_geburtsdatum", "slr_geburtsdatum"},
	spalteStrasse:      {"strasse", "schueler_strasse", "anschrift_strasse", "slr_strasse"},
	spalteHausnummer:   {"hausnummer", "schueler_hausnummer", "anschrift_hausnummer", "slr_hausnummer"},
	spaltePLZ:          {"plz", "postleitzahl", "schueler_plz", "schueler_postleitzahl", "anschrift_plz", "slr_plz"},
	spalteOrt:          {"ort", "wohnort", "schueler_ort", "anschrift_ort", "slr_ort"},
	spalteElternEmail:  {"eltern_email", "email", "ansprechpartner_email", "erziehungsberechtigte_email", "erziehungsberechtigter_email", "slr_email"},
	// Schuleintritt (02.09.2026): So heißt die Spalte im Individuellen Bericht der LUSD
	// („All_Inklusiv"-Demo-Export geprüft); die kurzen Formen für handgebaute Listen.
	spalteEintritt: {"eintritt", "eintrittsdatum", "schueler_eintritt_aktuelleschule", "eintritt_aktuelleschule", "schueler_eintritt", "slr_eintritt", "aufnahmedatum"},
}

// pflichtspalten müssen in jedem Export stehen. Den ZUORDNUNGSSCHLÜSSEL bestimmt
// die Datei selbst (datei.go): LUSD-ID, sonst Name + Geburtsdatum, sonst nur der
// Name — der LANIS-Klassenlisten-Export der Schule hat genau diese drei Spalten.
var pflichtspalten = []string{spalteVorname, spalteNachname, spalteKlasse}

// optionaleSpalten sind die Adress-/Kontaktspalten und der Schuleintritt (dürfen fehlen).
var optionaleSpalten = []string{spalteStrasse, spalteHausnummer, spaltePLZ, spalteOrt, spalteElternEmail, spalteEintritt}

// spalteZuKopf mappt jede normalisierte Alias-Schreibweise auf ihr logisches Feld.
var spalteZuKopf = baueSpalteZuKopf()

func baueSpalteZuKopf() map[string]string {
	nachKopf := make(map[string]string)
	for spalte, schreibweisen := range kopfSchreibweisen {
		for _, s := range schreibweisen {
			nachKopf[normiereKopf(s)] = spalte
		}
	}
	return nachKopf
}

// ordneSpaltenZu ordnet die erkannten logischen Spalten ihren Zeilen-Indizes zu und
// prüft die Pflichtspalten. Unbekannte Spalten werden ignoriert. Die Meldungen landen 1:1 im Import-Dialog des Sekretariats, daher
// deutsche Großschreibung statt kleingeschriebener Go-Fehler.
func ordneSpaltenZu(kopf []string) (map[string]int, error) {
	spalten := make(map[string]int)
	for i, zelle := range kopf {
		spalte, ok := spalteZuKopf[normiereKopf(zelle)]
		if !ok {
			continue
		}
		if _, schonZugeordnet := spalten[spalte]; schonZugeordnet {
			continue // erste passende Spalte gewinnt
		}
		spalten[spalte] = i
	}

	for _, spalte := range pflichtspalten {
		if _, da := spalten[spalte]; !da {
			return nil, fmt.Errorf("Pflichtspalte '%s' fehlt in der CSV-Kopfzeile — ist das die richtige LUSD-Exportdatei?", spalte) //nolint:staticcheck // ST1005: nutzer-sichtbarer Text, siehe oben
		}
	}

	meldeErkannteOptionaleSpalten(spalten)
	return spalten, nil
}

// meldeErkannteOptionaleSpalten macht sichtbar, welche Adressspalten erkannt wurden —
// so bleibt ein Tippfehler im Header nie unbemerkt („still leer importiert").
func meldeErkannteOptionaleSpalten(spalten map[string]int) {
	var erkannt, fehlend []string
	for _, spalte := range optionaleSpalten {
		if _, ok := spalten[spalte]; ok {
			erkannt = append(erkannt, spalte)
		} else {
			fehlend = append(fehlend, spalte)
		}
	}
	if len(erkannt) > 0 {
		log.Printf("LUSD-Import: Adress-/Kontaktspalten erkannt: %v — nicht gefunden: %v", erkannt, fehlend)
	}
}
