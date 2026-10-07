package littera

import (
	"io"
	"sort"
	"strconv"
	"strings"
)

// funktionVerfasser ist der Funktionsschlüssel des VERFASSERS in Personen_Zuordnung.
//
// Die Tabelle Personen_Funktionen führt 1 = Illustrator, 2 = Herausgeber,
// 3 = Redakteur und so fort — die 0 steht dort NICHT. Sie ist die Vorgabe: eine
// Zuordnung ohne besondere Funktion ist die Verfasserschaft selbst. Gemessen am
// Altbestand entfallen 19.191 von 20.562 Zuordnungen auf die 0.
const funktionVerfasser = "0"

// bestandsmarken sind Einträge aus `Personen`, die keine Personen sind.
//
// In der Bibliothek wurde das Verfasserfeld über Jahre auch zum Kennzeichnen des
// Bestands benutzt. Für Littera ist das nicht unterscheidbar — die Tabelle `Personen`
// hat außer dem Namen kein Merkmal, und diese Einträge tragen dieselben Flags wie
// „Neebe, Reinhard". Erst der Blick in den geschriebenen Katalog zeigt es:
//
//	Buchbestand Bibliothek   6.711 Titel
//	Bibliothek                 760
//	Klassensatz/Bibliothek     584
//	LMF                        363
//	U plus                     202
//
// Ohne diese Liste stünde bei 7.131 Titeln ein Standortvermerk mitten in der
// Autorenangabe („Shaw, George Bernard; Buchbestand Bibliothek") und bei 1.486 stünde er
// als einziger Autor da. Die fünf gelten an jeder Stelle der Zuordnung als Standortvermerk
// und kommen als Standort ans Exemplar; weitere erkennt standort.go an ihrer Stelle.
var bestandsmarken = map[string]bool{
	"Buchbestand Bibliothek": true,
	"Bibliothek":             true,
	"Klassensatz/Bibliothek": true,
	"LMF":                    true,
	"U plus":                 true,
}

// ohneBestandsmarken räumt eine Autorenangabe auf.
//
// Nötig auch für den Freitext `Titel.Verfasserangabe`, nicht nur für die
// Personen-Zuordnung: Dort stehen dieselben Vermerke, teils allein („Bibliothek"), teils
// zwischen echten Namen („Stefan Wolf ; Bibliothek"). Getrennt wird an „;", weil Littera
// beide Quellen so schreibt.
func ohneBestandsmarken(autor string) string {
	if autor == "" {
		return ""
	}
	var behalten []string
	for _, teil := range strings.Split(autor, ";") {
		teil = strings.TrimSpace(teil)
		if teil != "" && !bestandsmarken[teil] {
			behalten = append(behalten, teil)
		}
	}
	return strings.Join(behalten, "; ")
}

// LesePersonen liest die Tabelle `Personen` als Schlüssel → Name.
//
// Die Namen stehen in Katalogform („Neebe, Reinhard"), teils auch nur als Nachname
// („Linder") oder als Sammelangabe („Dorn . Bader") — so, wie sie am Buch stehen.
// Bewusst KEINE Umsortierung auf „Vorname Nachname": Der Katalog dieser Anwendung
// zeigt den Autor als Text, und eine automatische Umstellung würde bei den
// Sammelangaben Unsinn erzeugen.
func LesePersonen(r io.Reader) (map[string]string, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}
	namen := make(map[string]string, len(zeilen))
	for _, z := range zeilen {
		id := strings.TrimSpace(z["Buchungsnummer"])
		name := freitext(z["Name"])
		if id == "" || name == "" {
			continue
		}
		namen[id] = name
	}
	return namen, nil
}

// autorZuordnung ist eine Verfasser-Zuordnung in ihrer Erfassungsreihenfolge.
type autorZuordnung struct {
	titelID  string
	personID string
	lfd      int // Buchungsnummer der Zuordnung — hält die Reihenfolge des Katalogs
	// dritteStelle: An der Stelle des dritten Verfassers trägt die Bibliothek den Standort
	// des Titels ein (standort.go).
	dritteStelle bool
}

// Zuordnungen sind, was `Personen_Zuordnung` je Titel hergibt: die Verfasser und die
// Standortvermerke. VermerkNamen nennt jeden Namen, der an irgendeinem Titel als Vermerk
// erkannt ist.
type Zuordnungen struct {
	Autoren          map[string]string
	Standortvermerke map[string][]string
	VermerkNamen     map[string]bool
}

// LeseZuordnungen liest Verfasser und Standortvermerke in einem Durchgang: Ein Eintrag ist
// das eine oder das andere, nie beides.
//
// Die Verfasser kommen von hier, weil Titel.Verfasserangabe nur bei 2.877 von 10.732 Titeln
// gefüllt ist (27 %); über die Personen-Zuordnung sind es 9.904 (92 %). Mehrere Verfasser
// stehen mit „; " in der Reihenfolge der Erfassung (Buchungsnummer der Zuordnung), nicht
// alphabetisch: Bei einem Schulbuch ist der erstgenannte der Hauptverfasser.
func LeseZuordnungen(personen map[string]string, zuordnungen io.Reader, regel Standortregel) (Zuordnungen, error) {
	zeilen, err := leseTabelle(zuordnungen)
	if err != nil {
		return Zuordnungen{}, err
	}

	jeTitel := sammleVerfasserZuordnungen(zeilen)

	erg := Zuordnungen{
		Autoren:          make(map[string]string, len(jeTitel)),
		Standortvermerke: map[string][]string{},
		VermerkNamen:     map[string]bool{},
	}
	for titelID, liste := range jeTitel {
		if namen := namenInErfassungsreihenfolge(liste, personen, regel); len(namen) > 0 {
			erg.Autoren[titelID] = strings.Join(namen, "; ")
		}
		if vermerke := regel.vermerkeInErfassungsreihenfolge(liste, personen); len(vermerke) > 0 {
			erg.Standortvermerke[titelID] = vermerke
			for _, v := range vermerke {
				erg.VermerkNamen[v] = true
			}
		}
	}
	return erg, nil
}

// sammleVerfasserZuordnungen gruppiert die Verfasser-Zeilen nach Titel.
func sammleVerfasserZuordnungen(zeilen []map[string]string) map[string][]autorZuordnung {
	jeTitel := map[string][]autorZuordnung{}
	for _, z := range zeilen {
		if strings.TrimSpace(z["Funktion"]) != funktionVerfasser {
			continue // Illustrator, Herausgeber, Redakteur … sind keine Verfasser
		}
		titelID := strings.TrimSpace(z["Titel"])
		personID := strings.TrimSpace(z["Person"])
		if titelID == "" || personID == "" {
			continue
		}
		// Eine unlesbare Buchungsnummer kostet nur die Reihenfolge, nicht den Autor:
		// lfd bleibt 0, der Eintrag wandert nach vorn. Ihn deshalb zu verwerfen waere
		// der teurere Fehler.
		lfd, err := strconv.Atoi(strings.TrimSpace(z["Buchungsnummer"]))
		if err != nil {
			lfd = 0
		}
		jeTitel[titelID] = append(jeTitel[titelID],
			autorZuordnung{titelID, personID, lfd, anDritterStelle(z["Flags"])})
	}
	return jeTitel
}

// namenInErfassungsreihenfolge löst die Zuordnungen eines Titels zu Klarnamen auf —
// sortiert nach Buchungsnummer, ohne Dubletten und ohne Standortvermerke.
func namenInErfassungsreihenfolge(liste []autorZuordnung, personen map[string]string, regel Standortregel) []string {
	sort.Slice(liste, func(i, j int) bool { return liste[i].lfd < liste[j].lfd })

	var namen []string
	gesehen := map[string]bool{}
	for _, z := range liste {
		name := personen[z.personID]
		if name == "" || gesehen[name] || bestandsmarken[name] || regel.istVermerk(name, z.dritteStelle) {
			continue // unbekannte Person, Dublette in der Zuordnung oder Standortvermerk
		}
		gesehen[name] = true
		namen = append(namen, name)
	}
	return namen
}

// MitAutoren ergänzt die Titel um die aufgelösten Verfasser.
//
// Die Personen-Zuordnung gewinnt gegen Titel.Verfasserangabe: Sie ist die gepflegte,
// normalisierte Quelle (eine Person, ein Datensatz), während die Verfasserangabe ein
// freier Text am Titel ist. Wo die Zuordnung nichts liefert, bleibt die vorhandene
// Angabe stehen — sie ist besser als ein leeres Feld.
func MitAutoren(titel []Titel, autoren map[string]string) []Titel {
	ergebnis := make([]Titel, len(titel))
	copy(ergebnis, titel)
	for i := range ergebnis {
		if name := autoren[ergebnis[i].ID]; name != "" {
			ergebnis[i].Autor = name
		}
	}
	return ergebnis
}

// MedienartNamen liest die Nachschlagetabelle `Medienart` als Schlüssel → Bezeichnung.
//
// Titel.Medienart ist eine Zahl; ohne diese Auflösung stünde im Katalog „3" statt „Buch".
func MedienartNamen(r io.Reader) (map[string]string, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}
	namen := make(map[string]string, len(zeilen))
	for _, z := range zeilen {
		id := strings.TrimSpace(z["Buchungsnummer"])
		if id == "" {
			continue
		}
		namen[id] = strings.TrimSpace(z["Medienart"])
	}
	return namen, nil
}
