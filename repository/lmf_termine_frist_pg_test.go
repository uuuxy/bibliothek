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
// Klasse im Rückgabe-Plan, und hat der Büchertausch schon begonnen, vom Tag aus gesehen?
// Schreibvariante egal, Ausgabe-Zeilen zählen nicht, Entwürfe nicht. Die Termine kommen aus
// Plänen: ein vergangener Rückgabe-Plan, ein künftiger mit 9H1 an zwei Tagen, ein Ausgabe-Plan.
func TestRueckgabeTerminLage(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewLMFAbfragen(pool)
	t.Cleanup(func() { raeumeLMFPlaene(t, pool) })

	veroeffentliche(t, repo, speicherePlan(t, repo, LMFPlan{Art: LMFTerminRueckgabe, ErsterTag: "2026-06-01", Startstunde: 1, StundenJeTag: 6},
		[]LMFPlanZeile{{Klassen: []string{"9H1"}, Vermerk: "vergangen"}}, nil))
	// 9H1 am 28.06. (Zeile 1) und noch einmal am 05.07. (Zeile 6, hinter dem Wochenende);
	// 9H2 nur am 29.06.
	entwurf := speicherePlan(t, repo, LMFPlan{Art: LMFTerminRueckgabe, ErsterTag: "2027-06-28", Startstunde: 1, StundenJeTag: 1},
		[]LMFPlanZeile{{Klassen: []string{"9H1"}}, {Klassen: []string{"9H2"}}, {Klassen: []string{"10R1"}},
			{Klassen: []string{"10R2"}}, {Klassen: []string{"10R3"}}, {Klassen: []string{"9H1"}, Vermerk: "zweiter Termin"}}, nil)
	veroeffentliche(t, repo, speicherePlan(t, repo, LMFPlan{Art: LMFTerminAusgabe, ErsterTag: "2027-08-10", Startstunde: 2, StundenJeTag: 6},
		[]LMFPlanZeile{{Klassen: []string{"7G1"}, Vermerk: "neu"}}, nil))
	heute := time.Date(2026, time.September, 5, 12, 0, 0, 0, schulzeit.Zone())
	lageVon := func(klasse string, tag time.Time) LMFTerminLage {
		t.Helper()
		lage, err := repo.LadeRueckgabeTerminLage(ctx, klasse, tag)
		if err != nil {
			t.Fatalf("Lage %s am %s: %v", klasse, tag.Format("2006-01-02"), err)
		}
		return lage
	}

	tag := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatal(err)
		}
		return x.Add(10 * time.Hour)
	}

	// Solange der Plan 2027 Entwurf ist, kennt der Ausleihdienst nur den
	// vergangenen Termin — also keinen: Ein Entwurf setzt still keine Frist. Und der
	// Termin vom Juni 2026 liegt im vorigen Schuljahr, zählt am 05.09.2026 also nicht als
	// begonnener Büchertausch. Auch am ersten Tag des Entwurfs hat kein Tausch begonnen.
	if lage := lageVon("09h1", heute); lage.Bevorstehend || lage.TauschBegonnen {
		t.Errorf("ein Entwurf darf keine Frist liefern, ein alter Termin nicht nachwirken: %+v", lage)
	}
	if lage := lageVon("5F1", tag("2027-06-28")); lage.TauschBegonnen {
		t.Errorf("ein Entwurf beginnt keinen Büchertausch: %+v", lage)
	}
	veroeffentliche(t, repo, entwurf)

	lage := lageVon("09h1", heute)
	if !lage.Bevorstehend || lage.Naechster.Format("2006-01-02") != "2027-06-28" || lage.TauschBegonnen {
		t.Errorf("9H1 am 05.09.2026: %+v, erwartet nächster 28.06.2027 (nicht der spätere, nicht der vergangene)", lage)
	}
	if lage := lageVon("7G1", heute); lage.Bevorstehend || lage.TauschBegonnen {
		t.Errorf("eine Ausgabe-Zeile darf keine Frist liefern: %+v", lage)
	}
	if lage := lageVon("5F1", heute); lage.Bevorstehend || lage.TauschBegonnen {
		t.Errorf("Klasse ohne Termin liefert einen: %+v", lage)
	}

	// Die Matrix um den Beginn des Tauschs und den Termin (Bugklasse „Frist am Tag des
	// Ereignisses"): Der Plan beginnt Montag, 28.06.2027, mit 9H1; 9H2 hat genau einen Termin,
	// den 29.06.2027. Am Freitag davor hat der Tausch nicht begonnen; ab dem 28.06. hat er es für
	// jede Klasse, auch für 9H2 vor ihrem Termin und für 5F1, die nicht im Plan steht; im
	// nächsten Schuljahr nicht mehr.
	if lage := lageVon("9H2", tag("2027-06-25")); !lage.Bevorstehend || lage.Naechster.Format("2006-01-02") != "2027-06-29" || lage.TauschBegonnen {
		t.Errorf("9H2 am Freitag vor dem Tausch: %+v, erwartet nächster 29.06.2027, Tausch nicht begonnen", lage)
	}
	if lage := lageVon("5F1", tag("2027-06-25")); lage.Bevorstehend || lage.TauschBegonnen {
		t.Errorf("5F1 am Freitag vor dem Tausch: %+v, erwartet nichts", lage)
	}
	if lage := lageVon("9H2", tag("2027-06-28")); !lage.Bevorstehend || lage.Naechster.Format("2006-01-02") != "2027-06-29" || !lage.TauschBegonnen {
		t.Errorf("9H2 am ersten Tag des Tauschs: %+v, erwartet nächster 29.06.2027 und Tausch begonnen", lage)
	}
	if lage := lageVon("5F1", tag("2027-06-28")); lage.Bevorstehend || !lage.TauschBegonnen {
		t.Errorf("5F1 (nicht im Plan) am ersten Tag des Tauschs: %+v, erwartet Tausch begonnen", lage)
	}
	if lage := lageVon("9H2", tag("2027-06-29")); lage.Bevorstehend || !lage.TauschBegonnen {
		t.Errorf("9H2 am Termintag: %+v, erwartet Tausch begonnen und nichts bevorstehend", lage)
	}
	if lage := lageVon("9H2", tag("2027-06-30")); lage.Bevorstehend || !lage.TauschBegonnen {
		t.Errorf("9H2 am Tag danach: %+v, erwartet Tausch begonnen und nichts bevorstehend", lage)
	}
	if lage := lageVon("9H2", tag("2027-09-06")); lage.Bevorstehend || lage.TauschBegonnen {
		t.Errorf("9H2 im nächsten Schuljahr: %+v, erwartet weder bevorstehend noch Tausch begonnen", lage)
	}
	// 9H1 steht zweimal im Plan: Am ersten Termintag steht der zweite noch bevor.
	if lage := lageVon("9H1", tag("2027-06-28")); !lage.Bevorstehend || lage.Naechster.Format("2006-01-02") != "2027-07-05" || !lage.TauschBegonnen {
		t.Errorf("9H1 am ersten Termintag: %+v, erwartet nächster 05.07.2027 und Tausch begonnen", lage)
	}
}

// speicherePlan verteilt die Zeilen wie der Handler (Mo–Fr, ohne Ferien) und speichert.
// Vom Plan zählen Art, erster Tag, Startstunde und Stunden je Tag; das Ende des
// Rückgabe-Plans rechnet der Helfer.
func speicherePlan(t *testing.T, repo *LMFAbfragen, plan LMFPlan, zeilen []LMFPlanZeile, ausgelassen []string) LMFPlanStand {
	t.Helper()
	tag, err := time.ParseInLocation("2006-01-02", plan.ErsterTag, schulzeit.Zone())
	if err != nil {
		t.Fatal(err)
	}
	plaetze := lmfplan.VerteileMit(lmfplan.Rahmen{ErsterTag: tag, Startstunde: plan.Startstunde, StundenJeTag: plan.StundenJeTag},
		make([]*lmfplan.Platz, len(zeilen)), lmfplan.Schultage(nil))
	// Der Rückgabe-Plan trägt sein Ende: hier der Platz der letzten Zeile
	// — vom Ende her gerechnet ergäbe das dieselben Plätze.
	if plan.Art == LMFTerminRueckgabe {
		plan.LetzterTag, plan.LetzteStunde = plan.ErsterTag, plan.Startstunde
		if n := len(plaetze); n > 0 {
			plan.LetzterTag, plan.LetzteStunde = plaetze[n-1].Datum.Format("2006-01-02"), plaetze[n-1].Stunde
		}
	}
	st, err := schreibeLMFPlanImTest(context.Background(), repo, plan, zeilen, plaetze, ausgelassen)
	if err != nil {
		t.Fatalf("Plan %s ab %s speichern: %v", plan.Art, plan.ErsterTag, err)
	}
	return st
}

// veroeffentliche stempelt einen gespeicherten Plan — wie POST
// …/veroeffentlichen, ohne Frist-Kopplung (die liegt in api/).
func veroeffentliche(t *testing.T, repo *LMFAbfragen, st LMFPlanStand) LMFPlanStand {
	t.Helper()
	st, err := repo.VeroeffentlichePlan(context.Background(), repo.db, st.Plan.ID, time.Now())
	if err != nil {
		t.Fatalf("Plan %s veröffentlichen: %v", st.Plan.ID, err)
	}
	return st
}

// raeumeLMFPlaene löscht alle Pläne (CASCADE nimmt Zeilen, Klassen und Auslassungen mit).
func raeumeLMFPlaene(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM lmf_plaene`); err != nil {
		t.Logf("Aufräumen: %v", err)
	}
}
