package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Das Portal zeigt nur den Plan, der als nächster dran ist: bis einschließlich zum letzten Tag
// des Büchertauschs diesen, ab dem Tag danach die Ausgabe nach den Sommerferien. Die Ausgabe
// des Vorjahres und ein Entwurf stehen nie dort, auch nicht, wenn sonst nichts ansteht.
func TestLMFPlan_NaechsterPlan_WechseltAmTagNachDemLetztenTermin(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewLMFAbfragen(pool)
	raeumeLMFPlaene(t, pool)
	t.Cleanup(func() { raeumeLMFPlaene(t, pool) })

	// Ausgabe des Vorjahres (Mo 10.08.2026), Büchertausch Mo 21. und Di 22.06.2027, Ausgabe danach
	// (Mo 09.08.2027), alle veröffentlicht; dazu der Entwurf des nächsten Büchertauschs.
	veroeffentliche(t, repo, speicherePlan(t, repo, LMFPlan{Art: LMFTerminAusgabe, ErsterTag: "2026-08-10", Startstunde: 1, StundenJeTag: 6},
		[]LMFPlanZeile{{Klassen: []string{"5A"}}}, nil))
	veroeffentliche(t, repo, speicherePlan(t, repo, LMFPlan{Art: LMFTerminRueckgabe, ErsterTag: "2027-06-21", Startstunde: 1, StundenJeTag: 2},
		[]LMFPlanZeile{{Klassen: []string{"9A"}}, {Klassen: []string{"9B"}}, {Klassen: []string{"8A"}}}, nil))
	veroeffentliche(t, repo, speicherePlan(t, repo, LMFPlan{Art: LMFTerminAusgabe, ErsterTag: "2027-08-09", Startstunde: 1, StundenJeTag: 6},
		[]LMFPlanZeile{{Klassen: []string{"5A"}}, {Klassen: []string{"5B"}}}, nil))
	speicherePlan(t, repo, LMFPlan{Art: LMFTerminRueckgabe, ErsterTag: "2028-06-19", Startstunde: 1, StundenJeTag: 2},
		[]LMFPlanZeile{{Klassen: []string{"9A"}}}, nil)

	berlin := func(tag string, stunde, minute int) time.Time {
		d, err := time.ParseInLocation("2006-01-02", tag, schulzeit.Zone())
		if err != nil {
			t.Fatal(err)
		}
		return d.Add(time.Duration(stunde)*time.Hour + time.Duration(minute)*time.Minute)
	}
	faelle := []struct {
		name      string
		jetzt     time.Time
		art       string
		anzahl    int
		ersterTag string
	}{
		{"im Juni vor dem Büchertausch", berlin("2027-06-01", 9, 0), LMFTerminRueckgabe, 3, "2027-06-21"},
		{"am letzten Tag des Büchertauschs, spätabends", berlin("2027-06-22", 23, 30), LMFTerminRueckgabe, 3, "2027-06-21"},
		{"kurz nach Mitternacht danach", berlin("2027-06-23", 0, 30), LMFTerminAusgabe, 2, "2027-08-09"},
		{"am Tag der Ausgabe", berlin("2027-08-09", 15, 0), LMFTerminAusgabe, 2, "2027-08-09"},
		{"nach der Ausgabe, nur ein Entwurf steht an", berlin("2027-08-10", 8, 0), "", 0, ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			termine, err := repo.LadeNaechstenPlan(ctx, f.jetzt.UTC())
			if err != nil {
				t.Fatal(err)
			}
			if len(termine) != f.anzahl {
				t.Fatalf("%d Termine, erwartet %d: %+v", len(termine), f.anzahl, termine)
			}
			for _, termin := range termine {
				if termin.Art != f.art {
					t.Errorf("Termin %s der Art %q, erwartet nur %q", termin.Datum, termin.Art, f.art)
				}
			}
			if f.anzahl > 0 && termine[0].Datum != f.ersterTag {
				t.Errorf("Der Plan beginnt am %s, erwartet %s", termine[0].Datum, f.ersterTag)
			}
		})
	}
}
