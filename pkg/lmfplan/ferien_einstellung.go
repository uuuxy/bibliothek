package lmfplan

// ferien_einstellung.go — die Sommerferien als Einstellung (06.09.2026). Die
// Programmtabelle (ferien.go) reicht bis zum Ende des aktuellen KMK-Blocks; was danach
// kommt, trägt die Schule selbst ein: Einstellungen → LUSD & Versetzung → Sommerferien,
// gespeichert als JSON-Liste unter dem Schlüssel „sommerferien". Hier steht, wie der
// Text gelesen, geprüft und in Normalform gebracht wird — an EINER Stelle, damit
// Speichern (api/settings.go), Planer (api/lmf_plan.go) und Selbstprüfung
// (api/betriebsbereitschaft_handler.go) dieselbe Lesart haben.

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// SommerferienSchluessel ist der Schlüssel in system_einstellungen.
const SommerferienSchluessel = "sommerferien"

// SommerferienEintrag ist ein Jahr aus der Einstellung: Beginn und Ende als JJJJ-MM-TT.
type SommerferienEintrag struct {
	Jahr int    `json:"jahr"`
	Von  string `json:"von"`
	Bis  string `json:"bis"`
}

func (e SommerferienEintrag) zeitraum() (Zeitraum, bool) {
	von, err1 := time.Parse("2006-01-02", e.Von)
	bis, err2 := time.Parse("2006-01-02", e.Bis)
	if err1 != nil || err2 != nil {
		return Zeitraum{}, false
	}
	return Zeitraum{Von: von, Bis: bis, Name: "Sommerferien"}, true
}

// ParseSommerferien liest die Einstellung und prüft jeden Eintrag: Datum lesbar, Beginn
// vor Ende, beide im genannten Jahr, kein Jahr doppelt, plausible Länge (Sommerferien
// dauern 5–8 Wochen; ein Zahlendreher „2031-07-07 bis 2031-08-51" kommt hier nicht
// durch). Leer oder nur Leerraum ist eine leere Liste — die Vorgabe.
func ParseSommerferien(text string) ([]SommerferienEintrag, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	var eintraege []SommerferienEintrag
	if err := json.Unmarshal([]byte(text), &eintraege); err != nil {
		return nil, errors.New("Sommerferien: die Liste ist nicht lesbar")
	}
	gesehen := map[int]bool{}
	for _, e := range eintraege {
		z, ok := e.zeitraum()
		if !ok {
			return nil, fmt.Errorf("Sommerferien %d: Datum nicht lesbar (JJJJ-MM-TT)", e.Jahr)
		}
		if z.Von.Year() != e.Jahr || z.Bis.Year() != e.Jahr {
			return nil, fmt.Errorf("Sommerferien %d: Beginn und Ende müssen im Jahr %d liegen", e.Jahr, e.Jahr)
		}
		if !z.Bis.After(z.Von) {
			return nil, fmt.Errorf("Sommerferien %d: das Ende liegt nicht nach dem Beginn", e.Jahr)
		}
		if tage := z.Bis.Sub(z.Von).Hours() / 24; tage < 28 || tage > 63 {
			return nil, fmt.Errorf("Sommerferien %d: %d Tage sind keine Sommerferien (erwartet 4 bis 9 Wochen)", e.Jahr, int(tage)+1)
		}
		if gesehen[e.Jahr] {
			return nil, fmt.Errorf("Sommerferien %d: das Jahr steht zweimal in der Liste", e.Jahr)
		}
		gesehen[e.Jahr] = true
	}
	sort.Slice(eintraege, func(i, j int) bool { return eintraege[i].Jahr < eintraege[j].Jahr })
	return eintraege, nil
}

// NormalisiereSommerferien prüft den Text wie ParseSommerferien und gibt ihn in
// Normalform zurück (nach Jahr sortiert, kompaktes JSON, "" für die leere Liste) —
// das, was gespeichert wird.
func NormalisiereSommerferien(text string) (string, error) {
	eintraege, err := ParseSommerferien(text)
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

// FerientabelleAus baut die Tabelle für den Betrieb: Programmtabelle plus die
// eingestellten Jahre. Ein unlesbarer Text zählt nicht — der Planer läuft dann mit der
// Programmtabelle weiter, statt gar keine Vorgabe zu haben (gespeichert wird ohnehin
// nur, was NormalisiereSommerferien durchlässt).
func FerientabelleAus(text string) Ferientabelle {
	eintraege, err := ParseSommerferien(text)
	if err != nil {
		return Hessen()
	}
	return Hessen().Mit(eintraege)
}

// ProgrammEintraege sind die Jahre der Programmtabelle als Liste — die Oberfläche
// zeigt sie neben den eigenen Einträgen, damit man sieht, was schon da ist.
func ProgrammEintraege() []SommerferienEintrag {
	out := make([]SommerferienEintrag, 0, len(sommerferienHessen))
	for jahr, z := range sommerferienHessen {
		out = append(out, SommerferienEintrag{Jahr: jahr, Von: z.Von.Format("2006-01-02"), Bis: z.Bis.Format("2006-01-02")})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Jahr < out[j].Jahr })
	return out
}
