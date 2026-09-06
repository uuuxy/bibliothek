package api

import (
	"strings"
	"testing"

	"bibliothek/repository"
)

// pruefeLmfPlan: fester Platz und freie Tage (Migration 099) werden geprüft, bevor
// irgendetwas gerechnet wird — ein fester Termin ohne Datum ist kein „fließt eben",
// sondern ein Fehler, den der Planer sehen muss.
func TestPruefeLmfPlan_FestUndFreieTage(t *testing.T) {
	var req lmfPlanRequest
	req.LetzterTag, req.LetzteStunde, req.StundenJeTag = "2026-06-25", 4, 6
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

// Der Anker hängt an der Art (Migration 101): Der Rückgabe-Plan braucht das Ende und
// verwirft einen mitgeschickten Beginn (der Server rechnet ihn); der Ausgabe-Plan
// braucht den Beginn und lässt das Ende leer. Ohne Zeilen steht beim Rückgabe-Plan das
// Ende auch als Beginn — ein Plan ohne Tag gäbe ein leeres Schuljahr.
func TestPruefeLmfPlan_AnkerJeArt(t *testing.T) {
	var req lmfPlanRequest
	req.ErsterTag, req.Startstunde, req.StundenJeTag = "2026-06-11", 3, 6
	if _, err := pruefeLmfPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "letzter_tag") {
		t.Errorf("Rückgabe ohne Ende: %v", err)
	}
	req.LetzterTag, req.LetzteStunde = " 2026-06-25 ", 4
	e, err := pruefeLmfPlan("rueckgabe", req)
	if err != nil {
		t.Fatal(err)
	}
	if e.Plan.LetzterTag != "2026-06-25" || e.Plan.LetzteStunde != 4 || e.Plan.ErsterTag != "2026-06-25" || e.Plan.Startstunde != 4 {
		t.Errorf("Rückgabe-Anker: %+v", e.Plan)
	}
	req.LetzteStunde = 7
	if _, err := pruefeLmfPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "letzte_stunde") {
		t.Errorf("letzte Stunde hinter dem Tagesende: %v", err)
	}
	req.LetzteStunde = 4
	if e, err = pruefeLmfPlan("ausgabe", req); err != nil || e.Plan.LetzterTag != "" || e.Plan.LetzteStunde != 0 ||
		e.Plan.ErsterTag != "2026-06-11" || e.Plan.Startstunde != 3 {
		t.Errorf("Ausgabe-Anker: %+v (%v)", e.Plan, err)
	}
	req.ErsterTag = ""
	if _, err := pruefeLmfPlan("ausgabe", req); err == nil || !strings.Contains(err.Error(), "erster_tag") {
		t.Errorf("Ausgabe ohne Beginn: %v", err)
	}
}

// Die eine Regel, welche Klasse nicht in den Plan einer Art gehört — gelesen vom
// Vorschlag UND von der Liste „bleiben draußen" des Planers (ausgelassen_regel).
func TestLmfPlanRegelLaesstAus(t *testing.T) {
	eingang := []int{5, 7}
	f := func(name string, jg int, ober bool) repository.KlasseImPlan {
		return repository.KlasseImPlan{Name: name, Jahrgang: jg, Oberstufe: ober}
	}
	faelle := []struct {
		art  string
		k    repository.KlasseImPlan
		soll bool
	}{
		{repository.LmfTerminRueckgabe, f("05F1", 5, false), false},
		{repository.LmfTerminRueckgabe, f("10R1", 10, false), false},
		{repository.LmfTerminRueckgabe, f("12T1", 12, true), true},
		{repository.LmfTerminAusgabe, f("05F1", 5, false), false},
		{repository.LmfTerminAusgabe, f("07G1", 7, false), false},
		{repository.LmfTerminAusgabe, f("06F1", 6, false), true},
		{repository.LmfTerminAusgabe, f("12T1", 12, true), true},
	}
	for _, c := range faelle {
		if ist := lmfPlanRegelLaesstAus(c.art, eingang, c.k); ist != c.soll {
			t.Errorf("%s %s: ausgelassen=%v, erwartet %v", c.art, c.k.Name, ist, c.soll)
		}
	}
}

// Der Regel-Vorschlag endet wie Peters Excel: nach den Klassen „Nachzügler" und
// „Aufräumen" (Zeilen ohne Klasse); beim Ausgabe-Plan stehen davor nur die
// Eingangsjahrgänge. Das Vorjahr bringt seine eigenen Zeilen mit — ohne Zusatz.
func TestLmfPlanVorschlag_RegelEndetMitNachzueglerUndAufraeumen(t *testing.T) {
	klassen := []repository.KlasseImPlan{
		{Name: "10R1", Jahrgang: 10, Abschluss: true},
		{Name: "07G1", Jahrgang: 7},
		{Name: "05F1", Jahrgang: 5},
		{Name: "12T1", Jahrgang: 12, Oberstufe: true},
	}
	v := lmfPlanVorschlag(repository.LmfTerminAusgabe, []int{5, 7}, false, repository.LmfPlanStand{}, klassen)
	vermerke := make([]string, 0, len(v.Zeilen))
	for _, z := range v.Zeilen {
		if len(z.Klassen) > 0 {
			vermerke = append(vermerke, z.Klassen[0])
		} else {
			vermerke = append(vermerke, z.Vermerk)
		}
	}
	if got := strings.Join(vermerke, ","); got != "07G1,05F1,Nachzügler,Aufräumen" {
		t.Errorf("Ausgabe-Vorschlag: %s", got)
	}
	if got := strings.Join(v.Ausgelassen, ","); got != "10R1,12T1" {
		t.Errorf("Ausgabe ausgelassen: %s", got)
	}
	// Vorjahr: nur die eigenen Zeilen.
	st := repository.LmfPlanStand{Zeilen: []repository.LmfPlanZeile{{Klassen: []string{"07G1"}}}}
	if v := lmfPlanVorschlag(repository.LmfTerminAusgabe, []int{5, 7}, true, st, klassen); len(v.Zeilen) != 2 {
		t.Errorf("Vorjahr-Vorschlag: %d Zeilen, erwartet 2 (07G1 + neue 05F1)", len(v.Zeilen))
	}
}
