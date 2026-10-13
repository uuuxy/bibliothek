package service

import (
	"strings"
	"testing"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Zusicherung des Planers: je Zeile ein Platz. Die Datenbank kann sie nicht halten, sie ist eine
// Bedingung zwischen zwei Listen im Speicher. Die Tür verlangt stunden_je_tag zwischen 1 und
// 12, und beide Verteiler liefern genau dann einen Platz je Eintrag; bei 0 geben sie eine leere
// Liste zurück, und Vorschau wie Speichern griffen mit plaetze[i] hinein. Ein Indexfehler
// mitten im Speichern ist keine Meldung, die jemand lesen kann.
func TestVerteileLmfPlan_PlatzJeZeile(t *testing.T) {
	entwurf := func(stundenJeTag int) LmfPlanEntwurf {
		return LmfPlanEntwurf{
			Plan: repository.LMFPlan{
				Art: repository.LMFTerminAusgabe, ErsterTag: "2026-09-10",
				Startstunde: 1, StundenJeTag: stundenJeTag,
				FreieTage: []repository.LMFFreierTag{},
			},
			Zeilen: []repository.LMFPlanZeile{{Klassen: []string{"05A"}}, {Klassen: []string{"05B"}}},
			Fest:   []*lmfplan.Platz{nil, nil},
		}
	}

	t.Run("Verteiler liefert nichts", func(t *testing.T) {
		e := entwurf(0)
		plaetze, _, err := VerteileLmfPlan(&e)
		if err == nil {
			t.Fatalf("kein Fehler bei %d Plätzen für %d Zeilen — der nächste Zugriff wäre plaetze[i]",
				len(plaetze), len(e.Zeilen))
		}
		if !strings.Contains(err.Error(), "Zeilen") {
			t.Errorf("Fehlermeldung nennt das Missverhältnis nicht: %v", err)
		}
	})

	t.Run("Normalfall bleibt unberührt", func(t *testing.T) {
		e := entwurf(6)
		plaetze, _, err := VerteileLmfPlan(&e)
		if err != nil {
			t.Fatalf("gültiger Rahmen abgelehnt: %v", err)
		}
		if len(plaetze) != len(e.Zeilen) {
			t.Errorf("%d Plätze für %d Zeilen", len(plaetze), len(e.Zeilen))
		}
	})
}

// Ein freier Tag des Plans wird übersprungen und mit seinem Grund als Ausfall genannt; ohne
// Grund heißt er „freier Tag". Der 10.09.2026 ist ein Donnerstag, bei einer Stunde je Tag
// liegt jede Zeile auf einem eigenen Schultag.
func TestVerteileLmfPlan_FreieTageWerdenUebersprungenUndGenannt(t *testing.T) {
	entwurf := func(frei []repository.LMFFreierTag) LmfPlanEntwurf {
		return LmfPlanEntwurf{
			Plan: repository.LMFPlan{
				Art: repository.LMFTerminAusgabe, ErsterTag: "2026-09-10",
				Startstunde: 1, StundenJeTag: 1, FreieTage: frei,
			},
			Zeilen: []repository.LMFPlanZeile{{Klassen: []string{"05A"}}, {Klassen: []string{"05B"}}, {Klassen: []string{"05C"}}},
			Fest:   []*lmfplan.Platz{nil, nil, nil},
		}
	}

	e := entwurf([]repository.LMFFreierTag{{Datum: "2026-09-11", Grund: "Pädagogischer Tag"}, {Datum: "2026-09-14"}})
	plaetze, ausfaelle, err := VerteileLmfPlan(&e)
	if err != nil {
		t.Fatalf("Plan mit freien Tagen abgelehnt: %v", err)
	}
	var tage []string
	for _, p := range plaetze {
		tage = append(tage, p.Datum.Format("2006-01-02"))
	}
	if got := strings.Join(tage, ","); got != "2026-09-10,2026-09-15,2026-09-16" {
		t.Errorf("Plätze %s, erwartet 2026-09-10,2026-09-15,2026-09-16", got)
	}
	var genannt []string
	for _, a := range ausfaelle {
		genannt = append(genannt, a.Datum.Format(time.DateOnly)+" "+a.Grund)
	}
	if got := strings.Join(genannt, ","); got != "2026-09-11 Pädagogischer Tag,2026-09-14 freier Tag" {
		t.Errorf("Ausfälle %q, erwartet den pädagogischen Tag und den freien Tag ohne Grund", got)
	}

	e = entwurf([]repository.LMFFreierTag{{Datum: "kein Datum"}})
	if _, _, err := VerteileLmfPlan(&e); err == nil {
		t.Error("ein freier Tag ohne lesbares Datum muss die Verteilung abbrechen")
	}
}

// Der Rückgabe-Plan fließt von seinem Ende rückwärts, und sein Beginn ist danach der früheste
// Platz; der Ausgabe-Plan fließt von seinem Beginn vorwärts und behält ihn. Der 25.06.2026 ist
// ein Donnerstag, bei zwei Stunden je Tag liegen drei Zeilen auf zwei Tagen.
func TestVerteileLmfPlan_RueckgabeFliesstRueckwaertsUndTraegtDenBeginnEin(t *testing.T) {
	zeilen := []repository.LMFPlanZeile{{Klassen: []string{"10R1"}}, {Klassen: []string{"09H1"}}, {Klassen: []string{"08R1"}}}
	plaetzeAls := func(plaetze []lmfplan.Platz) string {
		var s []string
		for _, p := range plaetze {
			s = append(s, p.Datum.Format(time.DateOnly)+"/"+string(rune('0'+p.Stunde)))
		}
		return strings.Join(s, ",")
	}

	rueck := LmfPlanEntwurf{
		Plan: repository.LMFPlan{
			Art: repository.LMFTerminRueckgabe, LetzterTag: "2026-06-25", LetzteStunde: 2, StundenJeTag: 2,
			// Der Beginn der Anfrage zählt nicht, die Verteilung rechnet ihn.
			ErsterTag: "2026-06-25", Startstunde: 2, FreieTage: []repository.LMFFreierTag{},
		},
		Zeilen: zeilen, Fest: []*lmfplan.Platz{nil, nil, nil},
	}
	plaetze, _, err := VerteileLmfPlan(&rueck)
	if err != nil {
		t.Fatalf("Rückgabe-Plan: %v", err)
	}
	if got := plaetzeAls(plaetze); got != "2026-06-24/2,2026-06-25/1,2026-06-25/2" {
		t.Errorf("Rückgabe-Plan: Plätze %s, erwartet die letzte Zeile auf dem Ende und die übrigen davor", got)
	}
	if rueck.Plan.ErsterTag != "2026-06-24" || rueck.Plan.Startstunde != 2 {
		t.Errorf("Rückgabe-Plan: Beginn %s Stunde %d, erwartet den frühesten Platz (24.06., 2. Stunde)", rueck.Plan.ErsterTag, rueck.Plan.Startstunde)
	}

	aus := LmfPlanEntwurf{
		Plan: repository.LMFPlan{
			Art: repository.LMFTerminAusgabe, ErsterTag: "2026-06-25", Startstunde: 2, StundenJeTag: 2,
			// Ein Ende in der Anfrage zählt beim Ausgabe-Plan nicht.
			LetzterTag: "2026-06-30", LetzteStunde: 1, FreieTage: []repository.LMFFreierTag{},
		},
		Zeilen: zeilen, Fest: []*lmfplan.Platz{nil, nil, nil},
	}
	plaetze, ausfaelle, err := VerteileLmfPlan(&aus)
	if err != nil {
		t.Fatalf("Ausgabe-Plan: %v", err)
	}
	if got := plaetzeAls(plaetze); got != "2026-06-25/2,2026-06-26/1,2026-06-26/2" {
		t.Errorf("Ausgabe-Plan: Plätze %s, erwartet die erste Zeile auf dem Beginn und die übrigen dahinter", got)
	}
	if aus.Plan.ErsterTag != "2026-06-25" || aus.Plan.Startstunde != 2 {
		t.Errorf("Ausgabe-Plan: Beginn %s Stunde %d, erwartet unverändert", aus.Plan.ErsterTag, aus.Plan.Startstunde)
	}
	if ausfaelle == nil {
		t.Error("ohne Ausfall kommt nil statt einer leeren Liste zurück")
	}
}

// Ein Rahmen ohne lesbaren Anker bricht die Verteilung ab, statt am Kalendertag null zu rechnen.
func TestVerteileLmfPlan_AnkerOhneDatumIstEinFehler(t *testing.T) {
	for _, plan := range []repository.LMFPlan{
		{Art: repository.LMFTerminRueckgabe, LetzterTag: "", LetzteStunde: 1, StundenJeTag: 6, ErsterTag: "2026-06-25"},
		{Art: repository.LMFTerminAusgabe, ErsterTag: "kein Datum", Startstunde: 1, StundenJeTag: 6},
	} {
		e := LmfPlanEntwurf{Plan: plan, Zeilen: []repository.LMFPlanZeile{{Klassen: []string{"05F1"}}}, Fest: []*lmfplan.Platz{nil}}
		if _, _, err := VerteileLmfPlan(&e); err == nil {
			t.Errorf("%s-Plan ohne lesbaren Anker: kein Fehler", plan.Art)
		}
	}
}

// Freie Tage und feste Plätze des Plans gelten in beiden Richtungen: Der Rückgabe-Plan
// überspringt einen freien Tag auch beim Rückwärtsfließen, und eine Zeile mit festem Platz
// behält ihn, die übrigen fließen um ihn herum. Die Woche vom 22. bis 25.06.2026 hat vier
// Schultage.
func TestVerteileLmfPlan_FreieTageUndFestePlaetzeInBeidenRichtungen(t *testing.T) {
	zeilen := []repository.LMFPlanZeile{{Klassen: []string{"10R1"}}, {Klassen: []string{"09H1"}}, {Klassen: []string{"08R1"}}}
	platz := func(tag, stunde int) *lmfplan.Platz {
		return &lmfplan.Platz{Datum: time.Date(2026, time.June, tag, 0, 0, 0, 0, schulzeit.Zone()), Stunde: stunde}
	}
	plaetzeAls := func(plaetze []lmfplan.Platz) string {
		var s []string
		for _, p := range plaetze {
			s = append(s, p.Datum.Format(time.DateOnly)+"/"+string(rune('0'+p.Stunde)))
		}
		return strings.Join(s, ",")
	}
	rueckgabe := func(stundenJeTag int, frei []repository.LMFFreierTag, fest []*lmfplan.Platz) LmfPlanEntwurf {
		return LmfPlanEntwurf{
			Plan: repository.LMFPlan{
				Art: repository.LMFTerminRueckgabe, LetzterTag: "2026-06-25", LetzteStunde: stundenJeTag, StundenJeTag: stundenJeTag,
				ErsterTag: "2026-06-25", Startstunde: stundenJeTag, FreieTage: frei,
			},
			Zeilen: zeilen, Fest: fest,
		}
	}

	e := rueckgabe(1, []repository.LMFFreierTag{{Datum: "2026-06-24", Grund: "Studientag"}}, []*lmfplan.Platz{nil, nil, nil})
	plaetze, ausfaelle, err := VerteileLmfPlan(&e)
	if err != nil {
		t.Fatalf("Rückgabe-Plan mit freiem Tag: %v", err)
	}
	if got := plaetzeAls(plaetze); got != "2026-06-22/1,2026-06-23/1,2026-06-25/1" {
		t.Errorf("Rückgabe-Plan mit freiem Tag: Plätze %s, erwartet den 24.06. übersprungen", got)
	}
	if len(ausfaelle) != 1 || ausfaelle[0].Datum.Format(time.DateOnly) != "2026-06-24" || ausfaelle[0].Grund != "Studientag" {
		t.Errorf("Rückgabe-Plan mit freiem Tag: Ausfälle %+v, erwartet den Studientag am 24.06.", ausfaelle)
	}

	e = rueckgabe(2, []repository.LMFFreierTag{}, []*lmfplan.Platz{platz(22, 1), nil, nil})
	if plaetze, _, err = VerteileLmfPlan(&e); err != nil {
		t.Fatalf("Rückgabe-Plan mit festem Platz: %v", err)
	}
	if got := plaetzeAls(plaetze); got != "2026-06-22/1,2026-06-25/1,2026-06-25/2" {
		t.Errorf("Rückgabe-Plan mit festem Platz: Plätze %s, erwartet die erste Zeile am 22.06. in der 1. Stunde", got)
	}

	aus := LmfPlanEntwurf{
		Plan: repository.LMFPlan{
			Art: repository.LMFTerminAusgabe, ErsterTag: "2026-06-25", Startstunde: 1, StundenJeTag: 2,
			FreieTage: []repository.LMFFreierTag{},
		},
		Zeilen: zeilen, Fest: []*lmfplan.Platz{nil, platz(26, 2), nil},
	}
	if plaetze, _, err = VerteileLmfPlan(&aus); err != nil {
		t.Fatalf("Ausgabe-Plan mit festem Platz: %v", err)
	}
	if got := plaetzeAls(plaetze); got != "2026-06-25/1,2026-06-26/2,2026-06-25/2" {
		t.Errorf("Ausgabe-Plan mit festem Platz: Plätze %s, erwartet die zweite Zeile am 26.06. in der 2. Stunde", got)
	}
}
