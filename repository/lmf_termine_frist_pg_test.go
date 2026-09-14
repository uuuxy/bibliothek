package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Lookup, den der Ausleihdienst beim Ausleihen eines Schulbuchs macht: Wo steht die
// Klasse im Rückgabe-Plan, vom Tag aus gesehen? Schreibvariante egal, Ausgabe-Zeilen zählen
// nicht, Entwürfe nicht. Die Termine kommen aus Plänen (Migration 097): ein vergangener
// Rückgabe-Plan, ein künftiger mit 9H1 an zwei Tagen, ein Ausgabe-Plan.
func TestRueckgabeTerminLage(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewLmfTerminRepository(pool)
	t.Cleanup(func() { raeumeLmfPlaene(t, pool) })

	veroeffentliche(t, repo, speicherePlan(t, repo, LmfTerminRueckgabe, "2026-06-01", 1, 6,
		[]LmfPlanZeile{{Klassen: []string{"9H1"}, Vermerk: "vergangen"}}, nil))
	// 9H1 am 28.06. (Zeile 1) und noch einmal am 05.07. (Zeile 6, hinter dem Wochenende);
	// 9H2 nur am 29.06.
	entwurf := speicherePlan(t, repo, LmfTerminRueckgabe, "2027-06-28", 1, 1,
		[]LmfPlanZeile{{Klassen: []string{"9H1"}}, {Klassen: []string{"9H2"}}, {Klassen: []string{"10R1"}},
			{Klassen: []string{"10R2"}}, {Klassen: []string{"10R3"}}, {Klassen: []string{"9H1"}, Vermerk: "zweiter Termin"}}, nil)
	veroeffentliche(t, repo, speicherePlan(t, repo, LmfTerminAusgabe, "2027-08-10", 2, 6,
		[]LmfPlanZeile{{Klassen: []string{"7G1"}, Vermerk: "neu"}}, nil))
	heute := time.Date(2026, time.September, 5, 12, 0, 0, 0, schulzeit.Zone())
	lageVon := func(klasse string, tag time.Time) LmfTerminLage {
		t.Helper()
		lage, err := repo.RueckgabeTerminLage(ctx, klasse, tag)
		if err != nil {
			t.Fatalf("Lage %s am %s: %v", klasse, tag.Format("2006-01-02"), err)
		}
		return lage
	}

	// Solange der Plan 2027 Entwurf ist (Migration 100), kennt der Ausleihdienst nur den
	// vergangenen Termin — also keinen: Ein Entwurf setzt still keine Frist. Und der
	// Termin vom Juni 2026 liegt im vorigen Schuljahr, zählt am 05.09.2026 also nicht als
	// „vergangen".
	if lage := lageVon("09h1", heute); lage.Bevorstehend || lage.Vergangen {
		t.Errorf("ein Entwurf darf keine Frist liefern, ein alter Termin nicht nachwirken: %+v", lage)
	}
	veroeffentliche(t, repo, entwurf)

	lage := lageVon("09h1", heute)
	if !lage.Bevorstehend || lage.Naechster.Format("2006-01-02") != "2027-06-28" || lage.Vergangen {
		t.Errorf("9H1 am 05.09.2026: %+v, erwartet nächster 28.06.2027 (nicht der spätere, nicht der vergangene)", lage)
	}
	if lage := lageVon("7G1", heute); lage.Bevorstehend || lage.Vergangen {
		t.Errorf("eine Ausgabe-Zeile darf keine Frist liefern: %+v", lage)
	}
	if lage := lageVon("5F1", heute); lage.Bevorstehend || lage.Vergangen {
		t.Errorf("Klasse ohne Termin liefert einen: %+v", lage)
	}

	// Die Matrix um den Termin (Bugklasse „Frist am Tag des Ereignisses"): 9H2 hat genau
	// einen Termin, den 29.06.2027. Am Tag davor steht er bevor; am Termintag und danach
	// ist er vergangen und kein weiterer steht an; im nächsten Schuljahr ist er weder das
	// eine noch das andere.
	tag := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatal(err)
		}
		return x.Add(10 * time.Hour)
	}
	if lage := lageVon("9H2", tag("2027-06-28")); !lage.Bevorstehend || lage.Naechster.Format("2006-01-02") != "2027-06-29" || lage.Vergangen {
		t.Errorf("9H2 am Tag davor: %+v, erwartet nächster 29.06.2027, nicht vergangen", lage)
	}
	if lage := lageVon("9H2", tag("2027-06-29")); lage.Bevorstehend || !lage.Vergangen {
		t.Errorf("9H2 am Termintag: %+v, erwartet vergangen und nichts bevorstehend", lage)
	}
	if lage := lageVon("9H2", tag("2027-06-30")); lage.Bevorstehend || !lage.Vergangen {
		t.Errorf("9H2 am Tag danach: %+v, erwartet vergangen und nichts bevorstehend", lage)
	}
	if lage := lageVon("9H2", tag("2027-09-06")); lage.Bevorstehend || lage.Vergangen {
		t.Errorf("9H2 im nächsten Schuljahr: %+v, erwartet weder bevorstehend noch vergangen", lage)
	}
	// 9H1 steht zweimal im Plan: Am ersten Termintag steht der zweite noch bevor.
	if lage := lageVon("9H1", tag("2027-06-28")); !lage.Bevorstehend || lage.Naechster.Format("2006-01-02") != "2027-07-05" || !lage.Vergangen {
		t.Errorf("9H1 am ersten Termintag: %+v, erwartet nächster 05.07.2027 und vergangen", lage)
	}
}

// speicherePlan verteilt die Zeilen wie der Handler (Mo–Fr, ohne Ferien) und speichert.
func speicherePlan(t *testing.T, repo *LmfTerminRepository, art, ersterTag string, startstunde, stundenJeTag int, zeilen []LmfPlanZeile, ausgelassen []string) LmfPlanStand {
	t.Helper()
	tag, err := time.ParseInLocation("2006-01-02", ersterTag, schulzeit.Zone())
	if err != nil {
		t.Fatal(err)
	}
	plaetze := lmfplan.VerteileMit(lmfplan.Rahmen{ErsterTag: tag, Startstunde: startstunde, StundenJeTag: stundenJeTag},
		make([]*lmfplan.Platz, len(zeilen)), lmfplan.Schultage(nil))
	plan := LmfPlan{Art: art, ErsterTag: ersterTag, Startstunde: startstunde, StundenJeTag: stundenJeTag}
	// Der Rückgabe-Plan trägt sein Ende (Migration 101): hier der Platz der letzten Zeile
	// — vom Ende her gerechnet ergäbe das dieselben Plätze.
	if art == LmfTerminRueckgabe {
		plan.LetzterTag, plan.LetzteStunde = ersterTag, startstunde
		if n := len(plaetze); n > 0 {
			plan.LetzterTag, plan.LetzteStunde = plaetze[n-1].Datum.Format("2006-01-02"), plaetze[n-1].Stunde
		}
	}
	st, err := speichereLmfPlanImTest(context.Background(), repo, plan, zeilen, plaetze, ausgelassen)
	if err != nil {
		t.Fatalf("Plan %s ab %s speichern: %v", art, ersterTag, err)
	}
	return st
}

// veroeffentliche stempelt einen gespeicherten Plan (Migration 100) — wie POST
// …/veroeffentlichen, ohne Frist-Kopplung (die liegt in api/).
func veroeffentliche(t *testing.T, repo *LmfTerminRepository, st LmfPlanStand) LmfPlanStand {
	t.Helper()
	st, err := repo.VeroeffentlicheLmfPlanIn(context.Background(), repo.db, st.Plan.ID, time.Now())
	if err != nil {
		t.Fatalf("Plan %s veröffentlichen: %v", st.Plan.ID, err)
	}
	return st
}

// raeumeLmfPlaene löscht alle Pläne (CASCADE nimmt Zeilen, Klassen und Auslassungen mit).
func raeumeLmfPlaene(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM lmf_plaene`); err != nil {
		t.Logf("Aufräumen: %v", err)
	}
}
