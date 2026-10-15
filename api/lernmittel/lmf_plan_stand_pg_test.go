package lernmittel

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/schulzeit"
)

// Der Stand für den Planer über die Tür, an der Datenbank: Ein laufender Plan ist nicht vorbei
// und bekommt keinen Vorschlag; die Liste „bleiben draußen" nennt je Art die Klassen, die ihre
// Regel auslässt; und die Vorschau nennt die Ausfälle zwischen erstem Tag und letztem Platz.
func TestLMFPlan_StandFuerDenPlaner(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	raeume := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM lmf_plaene`); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)
	for i, klasse := range []string{"5F1", "6F1", "7G2", "9H1", "Q1", "12T1"} {
		testdaten.LegeSchuelerAn(t, pool, "ST-"+string(rune('1'+i)), "Kind", klasse)
	}
	// Die Uhr steht auf dem 06.09.2026; die nächsten Sommerferien beginnen am 28.06.2027.
	srv.Uhr = func() time.Time { return time.Date(2026, 9, 6, 10, 0, 0, 0, schulzeit.Zone()) }
	stand := func(t *testing.T, art string) lmfPlanStandAntwort {
		t.Helper()
		rec := lmfPlanAufruf(t, srv, http.MethodGet, art, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s lesen: %d %s", art, rec.Code, rec.Body.String())
		}
		var a lmfPlanStandAntwort
		if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
			t.Fatal(err)
		}
		return a
	}

	t.Run("bleiben draußen je Art", func(t *testing.T) {
		rueck, aus := stand(t, "rueckgabe"), stand(t, "ausgabe")
		if len(rueck.Klassen) != 6 || len(aus.Klassen) != 6 {
			t.Fatalf("Klassen mit Schülern: %v und %v, erwartet je sechs", rueck.Klassen, aus.Klassen)
		}
		// Der Büchertausch lässt die Oberstufe aus.
		if got := slices.Sorted(slices.Values(rueck.AusgelassenRegel)); !slices.Equal(got, []string{"12T1", "Q1"}) {
			t.Errorf("Rückgabe, bleiben draußen: %v, erwartet die Oberstufe (12T1, Q1)", got)
		}
		// Die Ausgabe lässt alles aus außer den Eingangsjahrgängen 5 und 7 (Vorgabe der
		// Einstellung).
		drin := []string{}
		for _, k := range aus.Klassen {
			if !slices.Contains(aus.AusgelassenRegel, k) {
				drin = append(drin, k)
			}
		}
		if len(aus.AusgelassenRegel) != 4 || len(drin) != 2 {
			t.Errorf("Ausgabe, bleiben draußen: %v, im Plan %v; erwartet vier draußen und die zwei Klassen der Jahrgänge 5 und 7 im Plan", aus.AusgelassenRegel, drin)
		}
	})

	t.Run("laufender Plan ist nicht vorbei und bekommt keinen Vorschlag", func(t *testing.T) {
		rec := lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
			`{"letzter_tag":"2027-06-24","letzte_stunde":4,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]},{"klassen":["7G2"]}]}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("Plan speichern: %d %s", rec.Code, rec.Body.String())
		}
		a := stand(t, "rueckgabe")
		if a.Plan == nil || a.Vorbei || a.Vorschlag != nil || len(a.Zeilen) != 2 {
			t.Errorf("laufender Plan: plan=%v vorbei=%v vorschlag=%+v zeilen=%d; erwartet einen Plan mit zwei Zeilen, nicht vorbei, ohne Vorschlag",
				a.Plan != nil, a.Vorbei, a.Vorschlag, len(a.Zeilen))
		}
		// Am Tag nach dem letzten Platz ist er vorbei, und der Vorschlag kommt aus ihm.
		srv.Uhr = func() time.Time { return time.Date(2027, 6, 25, 0, 30, 0, 0, schulzeit.Zone()) }
		t.Cleanup(func() { srv.Uhr = func() time.Time { return time.Date(2026, 9, 6, 10, 0, 0, 0, schulzeit.Zone()) } })
		a = stand(t, "rueckgabe")
		if !a.Vorbei || a.Vorschlag == nil || a.Vorschlag.Quelle != "vorjahr" {
			t.Errorf("am Tag nach dem letzten Platz: vorbei=%v vorschlag=%+v; erwartet vorbei mit dem Vorschlag aus dem Vorjahr", a.Vorbei, a.Vorschlag)
		}
	})

	t.Run("Vorschau nennt die Ausfälle und hält den festen Platz", func(t *testing.T) {
		// Der 16.08.2027 ist ein Montag; der 17.08. ist im Plan frei, die zweite Zeile liegt fest.
		rec := lmfPlanAufruf(t, srv, http.MethodPut, "ausgabe", `{"vorschau":true,"erster_tag":"2027-08-16","startstunde":1,"stunden_je_tag":1,
			"freie_tage":[{"datum":"2027-08-17","grund":"Pädagogischer Tag"}],
			"zeilen":[{"klassen":["5F1"]},{"klassen":["7G2"],"fest":{"datum":"2027-08-20","stunde":3}},{"klassen":["6F1"]}]}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("Vorschau: %d %s", rec.Code, rec.Body.String())
		}
		var a lmfPlanSpeicherAntwort
		if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
			t.Fatal(err)
		}
		if !a.Vorschau || len(a.Zeilen) != 3 {
			t.Fatalf("Vorschau: %+v", a)
		}
		var plaetze []string
		for _, z := range a.Zeilen {
			plaetze = append(plaetze, z.Datum+"/"+string(rune('0'+z.Stunde)))
		}
		if !slices.Equal(plaetze, []string{"2027-08-16/1", "2027-08-20/3", "2027-08-18/1"}) {
			t.Errorf("Plätze der Vorschau: %v, erwartet den freien Tag übersprungen und die zweite Zeile auf ihrem festen Platz", plaetze)
		}
		if !slices.ContainsFunc(a.Ausfaelle, func(x lmfPlanAusfall) bool { return x.Datum == "2027-08-17" && x.Grund == "Pädagogischer Tag" }) {
			t.Errorf("Ausfälle der Vorschau: %+v, erwartet den pädagogischen Tag am 17.08.2027", a.Ausfaelle)
		}
	})
}
