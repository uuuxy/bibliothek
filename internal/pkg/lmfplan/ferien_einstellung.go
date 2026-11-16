package lmfplan

// Die Sommerferien als Einstellung: Die Tabelle des Programms reicht bis zum Ende des
// laufenden Beschlusses der KMK, spätere Jahre trägt die Schule in den Einstellungen ein,
// gespeichert als JSON-Liste. Gelesen, geprüft und in Normalform gebracht wird der Text hier,
// damit Speichern, Planer und Selbstprüfung ihn gleich lesen.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// SommerferienSchluessel ist der Schlüssel der Einstellung in system_einstellungen.
const SommerferienSchluessel = "sommerferien"

// SommerferienEintrag ist ein Jahr aus der Einstellung, Beginn und Ende als JJJJ-MM-TT.
type SommerferienEintrag struct {
	Jahr int    `json:"jahr"`
	Von  string `json:"von"`
	Bis  string `json:"bis"`
}

// SommerferienFehler meldet Sommerferien, die nicht gespeichert werden. Jahr ist der Eintrag,
// der nicht stimmt, 0 für die Liste als Ganzes; Grund sagt, was nicht stimmt. Die Tür setzt
// beides zur Meldung zusammen.
type SommerferienFehler struct {
	Jahr  int
	Grund string
}

func (e *SommerferienFehler) Error() string {
	if e.Jahr == 0 {
		return "sommerferien: " + e.Grund
	}
	return fmt.Sprintf("sommerferien %d: %s", e.Jahr, e.Grund)
}

// zeitraum liefert den Eintrag als Zeitraum; ok ist false, wenn ein Datum nicht lesbar ist.
func (e SommerferienEintrag) zeitraum() (Zeitraum, bool) {
	von, err1 := time.Parse(time.DateOnly, e.Von)
	bis, err2 := time.Parse(time.DateOnly, e.Bis)
	if err1 != nil || err2 != nil {
		return Zeitraum{}, false
	}
	return Zeitraum{Von: von, Bis: bis, Name: "Sommerferien"}, true
}

// leseSommerferien liest die Einstellung und prüft jeden Eintrag: Datum lesbar, Beginn vor
// Ende, beide im genannten Jahr, kein Jahr doppelt und eine Länge von vier bis neun Wochen.
// Leer oder nur Leerraum ergibt die leere Liste, die Vorgabe.
func leseSommerferien(text string) ([]SommerferienEintrag, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	var eintraege []SommerferienEintrag
	if err := json.Unmarshal([]byte(text), &eintraege); err != nil {
		return nil, &SommerferienFehler{Grund: "die Liste ist nicht lesbar"}
	}
	gesehen := map[int]bool{}
	for _, e := range eintraege {
		if err := e.pruefe(); err != nil {
			return nil, err
		}
		if gesehen[e.Jahr] {
			return nil, &SommerferienFehler{Jahr: e.Jahr, Grund: "das Jahr steht zweimal in der Liste"}
		}
		gesehen[e.Jahr] = true
	}
	sort.Slice(eintraege, func(i, j int) bool { return eintraege[i].Jahr < eintraege[j].Jahr })
	return eintraege, nil
}

// pruefe prüft einen Eintrag für sich; ein doppeltes Jahr sieht nur die Liste.
func (e SommerferienEintrag) pruefe() error {
	z, ok := e.zeitraum()
	if !ok {
		return &SommerferienFehler{Jahr: e.Jahr, Grund: "Datum nicht lesbar (JJJJ-MM-TT)"}
	}
	// Ein Zahlendreher im Jahr (2131 statt 2031) machte die Selbstprüfung sonst hundert Jahre
	// stumm.
	if e.Jahr < 2000 || e.Jahr > 2100 {
		return &SommerferienFehler{Jahr: e.Jahr, Grund: "das Jahr liegt außerhalb von 2000 bis 2100. Ein Zahlendreher?"}
	}
	if z.Von.Year() != e.Jahr || z.Bis.Year() != e.Jahr {
		return &SommerferienFehler{Jahr: e.Jahr, Grund: fmt.Sprintf("Beginn und Ende müssen im Jahr %d liegen", e.Jahr)}
	}
	if !z.Bis.After(z.Von) {
		return &SommerferienFehler{Jahr: e.Jahr, Grund: "das Ende liegt nicht nach dem Beginn"}
	}
	if tage := z.Bis.Sub(z.Von).Hours() / 24; tage < 28 || tage > 63 {
		return &SommerferienFehler{Jahr: e.Jahr,
			Grund: fmt.Sprintf("%d Tage sind keine Sommerferien (erwartet 4 bis 9 Wochen)", int(tage)+1)}
	}
	return nil
}

// NormalisiereSommerferien prüft den Text wie leseSommerferien und liefert, was gespeichert
// wird: nach Jahr sortiert, als kompaktes JSON, "" für die leere Liste.
func NormalisiereSommerferien(text string) (string, error) {
	eintraege, err := leseSommerferien(text)
	if err != nil {
		return "", err
	}
	if len(eintraege) == 0 {
		return "", nil
	}
	b, err := json.Marshal(eintraege)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FerientabelleAus baut die Tabelle für den Betrieb: die des Programms und dazu die
// eingestellten Jahre. Ein unlesbarer Text zählt nicht; der Planer läuft dann mit der Tabelle
// des Programms weiter und steht nicht ohne Vorgabe da.
func FerientabelleAus(text string) Ferientabelle {
	eintraege, err := leseSommerferien(text)
	if err != nil {
		return Hessen()
	}
	return Hessen().mit(eintraege)
}

// ProgrammEintraege sind die Jahre der Tabelle des Programms als Liste. Die Oberfläche zeigt
// sie neben den eigenen Einträgen, damit zu sehen ist, was schon hinterlegt ist.
func ProgrammEintraege() []SommerferienEintrag {
	eintraege := make([]SommerferienEintrag, 0, len(sommerferienHessen))
	for jahr, z := range sommerferienHessen {
		eintraege = append(eintraege, SommerferienEintrag{Jahr: jahr, Von: z.Von.Format(time.DateOnly), Bis: z.Bis.Format(time.DateOnly)})
	}
	sort.Slice(eintraege, func(i, j int) bool { return eintraege[i].Jahr < eintraege[j].Jahr })
	return eintraege
}
