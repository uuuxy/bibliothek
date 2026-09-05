package api

import (
	"strings"
	"testing"
)

// pruefeLmfPlan: fester Platz und freie Tage (Migration 099) werden geprüft, bevor
// irgendetwas gerechnet wird — ein fester Termin ohne Datum ist kein „fließt eben",
// sondern ein Fehler, den der Planer sehen muss.
func TestPruefeLmfPlan_FestUndFreieTage(t *testing.T) {
	var req lmfPlanRequest
	req.ErsterTag, req.Startstunde, req.StundenJeTag = "2026-06-11", 3, 6
	req.FreieTage = append(req.FreieTage, struct {
		Datum string `json:"datum"`
		Grund string `json:"grund"`
	}{Datum: " 2026-06-05 ", Grund: " Brückentag "})
	req.Zeilen = append(req.Zeilen, struct {
		Klassen []string `json:"klassen"`
		Vermerk string   `json:"vermerk"`
		Fest    *struct {
			Datum  string `json:"datum"`
			Stunde int    `json:"stunde"`
		} `json:"fest"`
	}{Klassen: []string{"9H1"}}, struct {
		Klassen []string `json:"klassen"`
		Vermerk string   `json:"vermerk"`
		Fest    *struct {
			Datum  string `json:"datum"`
			Stunde int    `json:"stunde"`
		} `json:"fest"`
	}{Klassen: []string{"7G2"}, Vermerk: "Ausflug am Freitag", Fest: &struct {
		Datum  string `json:"datum"`
		Stunde int    `json:"stunde"`
	}{Datum: "2026-06-15", Stunde: 2}})

	e, err := pruefeLmfPlan("rueckgabe", req)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Plan.FreieTage) != 1 || e.Plan.FreieTage[0].Datum != "2026-06-05" || e.Plan.FreieTage[0].Grund != "Brückentag" {
		t.Errorf("freie Tage: %+v", e.Plan.FreieTage)
	}
	if len(e.Fest) != 2 || e.Fest[0] != nil || e.Fest[1] == nil {
		t.Fatalf("Fest-Vorgaben: %+v", e.Fest)
	}
	if e.Fest[1].Datum.Format("2006-01-02") != "2026-06-15" || e.Fest[1].Stunde != 2 {
		t.Errorf("fester Platz: %s/%d", e.Fest[1].Datum.Format("2006-01-02"), e.Fest[1].Stunde)
	}
	if !e.Zeilen[1].Fest || e.Zeilen[0].Fest {
		t.Errorf("fest-Marke: %v / %v", e.Zeilen[0].Fest, e.Zeilen[1].Fest)
	}

	// Fester Termin ohne Datum, feste Stunde 0, freier Tag ohne Datum: jeweils 400.
	req.Zeilen[1].Fest.Datum = ""
	if _, err := pruefeLmfPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "zeile 2") {
		t.Errorf("fester Termin ohne Datum: %v", err)
	}
	req.Zeilen[1].Fest.Datum, req.Zeilen[1].Fest.Stunde = "2026-06-15", 0
	if _, err := pruefeLmfPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "Stunde") {
		t.Errorf("feste Stunde 0: %v", err)
	}
	req.Zeilen[1].Fest.Stunde = 2
	req.FreieTage[0].Datum = "Freitag"
	if _, err := pruefeLmfPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "freier Tag 1") {
		t.Errorf("freier Tag ohne Datum: %v", err)
	}
}
