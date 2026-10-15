package service

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
)

// Welcher Plan beim Speichern als „vorher" zählt: der veröffentlichte Plan derselben Art im
// Schuljahr des neuen ersten Tages. Seine Zeilen bekommt die Kopplung der Fristen als „alt";
// Klassen, die im neuen Plan fehlen, kehren zum Stichtag zurück. Ein Entwurf hat keine Frist
// gesetzt, und der Plan eines anderen Schuljahres bleibt neben dem neuen stehen: Beide liefern
// keine Zeile, sonst nähme das Speichern Fristen zurück, die der neue Plan nie gesetzt hat.
func TestLadeLMFPlanZeilenVorher(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := repository.NewLMFAbfragen(pool)
	raeume := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM lmf_plaene`); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)

	const art = repository.LMFTerminRueckgabe
	lies := func(ersterTag string) []repository.LMFPlanZeile {
		t.Helper()
		zeilen, err := LadeLMFPlanZeilenVorher(ctx, repo, art, ersterTag)
		if err != nil {
			t.Fatalf("Zeilen vorher zum %s: %v", ersterTag, err)
		}
		return zeilen
	}
	if z := lies("2027-06-22"); z != nil {
		t.Errorf("ohne Plan: %+v, erwartet keine Zeile und keinen Fehler", z)
	}

	// Ein Plan im Schuljahr 2026/27 mit zwei Zeilen, zuerst als Entwurf.
	zeilen := []repository.LMFPlanZeile{{Klassen: []string{"9H1"}}, {Klassen: []string{"10R1"}, Vermerk: "nur Rückgabe"}}
	plaetze := lmfplan.VerteileMit(lmfplan.Rahmen{ErsterTag: time.Date(2027, time.June, 22, 0, 0, 0, 0, schulzeit.Zone()), Startstunde: 1, StundenJeTag: 1},
		make([]*lmfplan.Platz, len(zeilen)), lmfplan.Schultage(nil))
	plan := repository.LMFPlan{Art: art, ErsterTag: "2027-06-22", Startstunde: 1, StundenJeTag: 1, LetzterTag: "2027-06-23", LetzteStunde: 1}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stand, err := repo.SchreibePlan(ctx, tx, plan, zeilen, plaetze, nil)
	if err != nil {
		t.Fatalf("Plan speichern: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if z := lies("2027-06-21"); len(z) != 0 {
		t.Errorf("Entwurf im selben Schuljahr: %+v, erwartet keine Zeile", z)
	}

	if _, err := repo.VeroeffentlichePlan(ctx, pool, stand.Plan.ID, time.Now()); err != nil {
		t.Fatalf("Plan veröffentlichen: %v", err)
	}
	var klassen [][]string
	for _, z := range lies("2027-06-21") {
		klassen = append(klassen, z.Klassen)
	}
	// Die Datenbank führt die Klasse in ihrer Anzeigeform („09H1").
	if !cmp.Equal(klassen, [][]string{{"09H1"}, {"10R1"}}) {
		t.Errorf("veröffentlichter Plan im selben Schuljahr: Klassen %v, erwartet die zwei Zeilen des Plans", klassen)
	}
	// Der 31.07.2027 gehört noch zum Schuljahr 2026/27, der 01.08.2027 zum nächsten.
	if z := lies("2027-07-31"); len(z) != 2 {
		t.Errorf("letzter Tag desselben Schuljahres: %d Zeilen, erwartet 2", len(z))
	}
	if z := lies("2027-08-01"); len(z) != 0 {
		t.Errorf("erster Tag des nächsten Schuljahres: %+v, erwartet keine Zeile", z)
	}
	if z := lies("2028-06-20"); len(z) != 0 {
		t.Errorf("Plan des folgenden Schuljahres: %+v, erwartet keine Zeile", z)
	}

	// Eine andere Art hat keinen Plan, und ein unlesbarer Tag ist ein Fehler.
	if z, err := LadeLMFPlanZeilenVorher(ctx, repo, repository.LMFTerminAusgabe, "2027-06-21"); err != nil || z != nil {
		t.Errorf("andere Art: %+v, %v; erwartet keine Zeile und keinen Fehler", z, err)
	}
	if _, err := LadeLMFPlanZeilenVorher(ctx, repo, art, "kein Datum"); err == nil {
		t.Error("unlesbarer erster Tag: kein Fehler")
	}
}
