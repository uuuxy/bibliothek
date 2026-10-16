package service

import (
	"strings"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
)

// Die eine Regel, welche Klasse nicht in den Plan einer Art gehört. Der Vorschlag und die Liste
// „bleiben draußen" des Planers (ausgelassen_regel) fragen dieselbe.
func TestIstAusgelassenImLMFPlan(t *testing.T) {
	eingang := []int{5, 7}
	f := func(name string, jg int, ober bool) repository.KlasseImPlan {
		return repository.KlasseImPlan{Name: name, Jahrgang: jg, Oberstufe: ober}
	}
	faelle := []struct {
		art  string
		k    repository.KlasseImPlan
		soll bool
	}{
		{repository.LMFTerminRueckgabe, f("05F1", 5, false), false},
		{repository.LMFTerminRueckgabe, f("10R1", 10, false), false},
		{repository.LMFTerminRueckgabe, f("12T1", 12, true), true},
		{repository.LMFTerminAusgabe, f("05F1", 5, false), false},
		{repository.LMFTerminAusgabe, f("07G1", 7, false), false},
		{repository.LMFTerminAusgabe, f("06F1", 6, false), true},
		{repository.LMFTerminAusgabe, f("12T1", 12, true), true},
	}
	for _, c := range faelle {
		if ist := IstAusgelassenImLMFPlan(c.art, eingang, c.k); ist != c.soll {
			t.Errorf("%s %s: ausgelassen=%v, erwartet %v", c.art, c.k.Name, ist, c.soll)
		}
	}
}

// Die Reihenfolge aus der Regel endet wie der Plan der Schule: nach den Klassen „Nachzügler" und
// „Aufräumen" (Zeilen ohne Klasse); beim Ausgabe-Plan stehen davor nur die Eingangsjahrgänge.
// Das Vorjahr bringt seine eigenen Zeilen mit, ohne Zusatz.
func TestLMFPlanReihenfolgeVorgabe_RegelEndetMitNachzueglerUndAufraeumen(t *testing.T) {
	klassen := []repository.KlasseImPlan{
		{Name: "10R1", Jahrgang: 10, Abschluss: true},
		{Name: "07G1", Jahrgang: 7},
		{Name: "05F1", Jahrgang: 5},
		{Name: "12T1", Jahrgang: 12, Oberstufe: true},
	}
	v := LMFPlanReihenfolgeVorgabe(repository.LMFTerminAusgabe, []int{5, 7}, false, repository.LMFPlanStand{}, klassen)
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
	st := repository.LMFPlanStand{Zeilen: []repository.LMFPlanZeile{{Klassen: []string{"07G1"}}}}
	if v := LMFPlanReihenfolgeVorgabe(repository.LMFTerminAusgabe, []int{5, 7}, true, st, klassen); len(v.Zeilen) != 2 {
		t.Errorf("Vorjahr-Vorschlag: %d Zeilen, erwartet 2 (07G1 + neue 05F1)", len(v.Zeilen))
	}
}

// Aus dem Vorjahr kommen Klassen und Vermerk einer Zeile, nicht ihr Platz: Datum, Stunde und
// das Merkmal „fest" gehören dem alten Plan. Klassen, die der alte Plan schon nennt, in einer
// Zeile oder unter den ausgelassenen, kommen nicht ein zweites Mal dazu, auch nicht in anderer
// Schreibweise; neue Klassen stehen hinten oder nach der Regel draußen.
func TestLMFPlanReihenfolgeVorgabe_VorjahrBringtKlassenUndVermerkOhnePlatz(t *testing.T) {
	st := repository.LMFPlanStand{
		Zeilen: []repository.LMFPlanZeile{
			{Position: 1, Datum: "2026-06-24", Stunde: 3, Fest: true, Klassen: []string{"10R1", "09H1"}, Vermerk: "nur Rückgabe"},
			{Position: 2, Datum: "2026-06-25", Stunde: 4, Klassen: []string{}, Vermerk: "Aufräumen"},
		},
		Ausgelassen: []string{"08R1"},
	}
	klassen := []repository.KlasseImPlan{
		{Name: "10r1 ", Jahrgang: 10}, // steht im Vorjahr als 10R1
		{Name: "08R1", Jahrgang: 8},   // stand im Vorjahr draußen
		{Name: "07G1", Jahrgang: 7},   // neu
		{Name: "12T1", Jahrgang: 12, Oberstufe: true},
	}
	ist := LMFPlanReihenfolgeVorgabe(repository.LMFTerminRueckgabe, []int{5, 7}, true, st, klassen)
	will := LMFPlanReihenfolge{
		Quelle: lmfPlanQuelleVorjahr,
		Zeilen: []repository.LMFPlanZeile{
			{Klassen: []string{"10R1", "09H1"}, Vermerk: "nur Rückgabe"},
			{Klassen: []string{}, Vermerk: "Aufräumen"},
			{Klassen: []string{"07G1"}},
		},
		Ausgelassen: []string{"08R1", "12T1"},
	}
	if !cmp.Equal(ist, will) {
		t.Errorf("Reihenfolge aus dem Vorjahr =\n%+v\nerwartet\n%+v", ist, will)
	}
	// Ohne Vorjahr gilt die Regel, und was im Stand steht, zählt nicht.
	regel := LMFPlanReihenfolgeVorgabe(repository.LMFTerminRueckgabe, []int{5, 7}, false, st, klassen)
	if regel.Quelle != lmfPlanQuelleRegel || len(regel.Zeilen) != 5 || !cmp.Equal(regel.Ausgelassen, []string{"12T1"}) {
		t.Errorf("Reihenfolge aus der Regel: %+v, erwartet drei Klassen, Nachzügler und Aufräumen und die Oberstufe draußen", regel)
	}
	// Leere Listen statt nil: Der Planer bekommt `[]`.
	leer := LMFPlanReihenfolgeVorgabe(repository.LMFTerminAusgabe, nil, true, repository.LMFPlanStand{}, nil)
	if leer.Zeilen == nil || leer.Ausgelassen == nil {
		t.Errorf("leere Reihenfolge trägt nil: %+v", leer)
	}
}

// Ein Plan ist vorbei, wenn sein letzter Platz vor dem heutigen Kalendertag der Schule liegt.
// Am Tag des letzten Platzes läuft er noch, bis Mitternacht in der Zone der Schule.
func TestIstLMFPlanVorbei(t *testing.T) {
	in := func(j int, m time.Month, d, h, minute int) time.Time {
		return time.Date(j, m, d, h, minute, 0, 0, schulzeit.Zone())
	}
	plan := repository.LMFPlanStand{
		Plan: repository.LMFPlan{ErsterTag: "2026-06-22"},
		// Der späteste Tag steht nicht in der letzten Zeile.
		Zeilen: []repository.LMFPlanZeile{{Datum: "2026-06-23"}, {Datum: "2026-06-25"}, {Datum: "2026-06-24"}},
	}
	ohneZeilen := repository.LMFPlanStand{Plan: repository.LMFPlan{ErsterTag: "2026-06-22"}}
	for _, f := range []struct {
		name  string
		st    repository.LMFPlanStand
		jetzt time.Time
		want  bool
	}{
		{"am letzten Tag kurz vor Mitternacht", plan, in(2026, time.June, 25, 23, 59), false},
		{"am Tag danach kurz nach Mitternacht", plan, in(2026, time.June, 26, 0, 1), true},
		// 22:01 UTC am 25. ist in der Schule schon der 26.
		{"Tag danach, als UTC gereicht", plan, time.Date(2026, time.June, 25, 22, 1, 0, 0, time.UTC), true},
		{"letzter Tag, als UTC gereicht", plan, time.Date(2026, time.June, 25, 21, 59, 0, 0, time.UTC), false},
		{"vor dem Plan", plan, in(2026, time.June, 1, 12, 0), false},
		{"ohne Zeilen zählt der erste Tag", ohneZeilen, in(2026, time.June, 22, 12, 0), false},
		{"ohne Zeilen, am Tag nach dem ersten", ohneZeilen, in(2026, time.June, 23, 0, 1), true},
	} {
		if ist := IstLMFPlanVorbei(f.st, f.jetzt); ist != f.want {
			t.Errorf("%s: vorbei=%v, erwartet %v", f.name, ist, f.want)
		}
	}
}

// Der Vermerk „nur Rückgabe" wird vorbelegt, wenn jede Klasse der Zeile nur abgibt. Ein
// vorhandener Vermerk bleibt dahinter stehen; sagt er es schon, bleibt die Zeile, wie sie ist.
func TestBelegeVermerkNurRueckgabeVor(t *testing.T) {
	zeilen := []repository.LMFPlanZeile{
		{Klassen: []string{"10R1", "09H1"}},                                // alle geben nur ab
		{Klassen: []string{"10R1", "07G1"}},                                // eine tauscht
		{Klassen: []string{"09H1"}, Vermerk: "Ausflug"},                    // Vermerk bleibt dahinter
		{Klassen: []string{"09H1"}, Vermerk: "NUR RÜCKGABE, dann Zeugnis"}, // sagt es schon
		{Klassen: []string{}, Vermerk: "Aufräumen"},                        // ohne Klasse
		{Klassen: []string{" 10r1"}},                                       // andere Schreibweise
		{Klassen: []string{"07G1"}},                                        // tauscht
	}
	BelegeVermerkNurRueckgabeVor(zeilen, []string{"10R1", "09h1 "})
	will := []string{"nur Rückgabe", "", "nur Rückgabe · Ausflug", "NUR RÜCKGABE, dann Zeugnis", "Aufräumen", "nur Rückgabe", ""}
	for i, z := range zeilen {
		if z.Vermerk != will[i] {
			t.Errorf("Zeile %d (%v): Vermerk %q, erwartet %q", i+1, z.Klassen, z.Vermerk, will[i])
		}
	}
	// Ohne Klassen, die nur abgeben, bleibt jede Zeile leer.
	ohne := []repository.LMFPlanZeile{{Klassen: []string{"10R1"}}}
	BelegeVermerkNurRueckgabeVor(ohne, nil)
	if ohne[0].Vermerk != "" {
		t.Errorf("ohne Klassen, die nur abgeben: Vermerk %q", ohne[0].Vermerk)
	}
}
