package lernmittel

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bibliothek/pkg/lmfplan"
)

// pruefeLMFPlan: fester Platz und freie Tage (Migration 099) werden geprüft, bevor
// irgendetwas gerechnet wird — ein fester Termin ohne Datum ist kein „fließt eben",
// sondern ein Fehler, den der Planer sehen muss.
func TestPruefeLMFPlan_FestUndFreieTage(t *testing.T) {
	var req lmfPlanAnfrage
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

	e, err := pruefeLMFPlan("rueckgabe", req)
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
	if _, err := pruefeLMFPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "zeile 2") {
		t.Errorf("fester Termin ohne Datum: %v", err)
	}
	req.Zeilen[1].Fest.Datum, req.Zeilen[1].Fest.Stunde = "2026-06-15", 0
	if _, err := pruefeLMFPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "Stunde") {
		t.Errorf("feste Stunde 0: %v", err)
	}
	req.Zeilen[1].Fest.Stunde = 2
	req.FreieTage[0].Datum = "Freitag"
	if _, err := pruefeLMFPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "freier Tag 1") {
		t.Errorf("freier Tag ohne Datum: %v", err)
	}
}

// Der Anker hängt an der Art (Migration 101): Der Rückgabe-Plan braucht das Ende und
// verwirft einen mitgeschickten Beginn (der Server rechnet ihn); der Ausgabe-Plan
// braucht den Beginn und lässt das Ende leer. Ohne Zeilen steht beim Rückgabe-Plan das
// Ende auch als Beginn — ein Plan ohne Tag gäbe ein leeres Schuljahr.
func TestPruefeLMFPlan_AnkerJeArt(t *testing.T) {
	var req lmfPlanAnfrage
	req.ErsterTag, req.Startstunde, req.StundenJeTag = "2026-06-11", 3, 6
	if _, err := pruefeLMFPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "letzter_tag") {
		t.Errorf("Rückgabe ohne Ende: %v", err)
	}
	req.LetzterTag, req.LetzteStunde = " 2026-06-25 ", 4
	e, err := pruefeLMFPlan("rueckgabe", req)
	if err != nil {
		t.Fatal(err)
	}
	if e.Plan.LetzterTag != "2026-06-25" || e.Plan.LetzteStunde != 4 || e.Plan.ErsterTag != "2026-06-25" || e.Plan.Startstunde != 4 {
		t.Errorf("Rückgabe-Anker: %+v", e.Plan)
	}
	req.LetzteStunde = 7
	if _, err := pruefeLMFPlan("rueckgabe", req); err == nil || !strings.Contains(err.Error(), "letzte_stunde") {
		t.Errorf("letzte Stunde hinter dem Tagesende: %v", err)
	}
	req.LetzteStunde = 4
	if e, err = pruefeLMFPlan("ausgabe", req); err != nil || e.Plan.LetzterTag != "" || e.Plan.LetzteStunde != 0 ||
		e.Plan.ErsterTag != "2026-06-11" || e.Plan.Startstunde != 3 {
		t.Errorf("Ausgabe-Anker: %+v (%v)", e.Plan, err)
	}
	req.ErsterTag = ""
	if _, err := pruefeLMFPlan("ausgabe", req); err == nil || !strings.Contains(err.Error(), "erster_tag") {
		t.Errorf("Ausgabe ohne Beginn: %v", err)
	}
}

// Die Ausfälle der Verteilung stehen in der Antwort mit dem Tag als JJJJ-MM-TT und ihrem Grund,
// in der Folge der Verteilung. Ohne Ausfall bekommt der Planer eine leere Liste und nicht null.
func TestLMFPlanAusfaelle_FormDerAntwort(t *testing.T) {
	ist := lmfPlanAusfaelle([]lmfplan.Ausfall{
		{Datum: time.Date(2026, time.June, 4, 0, 0, 0, 0, time.UTC), Grund: "Fronleichnam"},
		{Datum: time.Date(2026, time.June, 5, 0, 0, 0, 0, time.UTC), Grund: "Brückentag"},
	})
	roh, err := json.Marshal(ist)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(roh), `[{"datum":"2026-06-04","grund":"Fronleichnam"},{"datum":"2026-06-05","grund":"Brückentag"}]`; got != want {
		t.Errorf("Ausfälle in der Antwort: %s, erwartet %s", got, want)
	}
	if roh, err = json.Marshal(lmfPlanAusfaelle(nil)); err != nil || string(roh) != "[]" {
		t.Errorf("ohne Ausfall: %s (%v), erwartet []", roh, err)
	}
}
