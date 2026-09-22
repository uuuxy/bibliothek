package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Frist eines Schulbuchs um den Rückgabetermin der Klasse, am echten Postgres und mit
// fester Uhr (docs/OFFEN.md 1.4, Entscheidung 13.09.2026): Am Tag davor ist der Termin die
// Frist. Am Termintag und danach gibt es das Buch erst im nächsten Schuljahr zurück — die
// Frist ist dessen Stichtag. Bis zum 14.09.2026 war sie am Termintag der Termin selbst
// (heute 23:59) und danach der 31.07. des laufenden Schuljahres, ein Tag in den Ferien:
// Nach den Ferien wäre die ganze Klasse überfällig und nach 14 Tagen gesperrt gewesen.
func TestLmfFrist_AmOderNachDemRueckgabeterminGiltDasFolgendeSchuljahr(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	rueckgabePlanJuni2027(t, pool)

	svc := &defaultLoanService{pool: pool}
	schulbuch := &repository.BookCopy{Titel: "Mathe 9", IstLernmittel: true, Medientyp: "Buch"}
	tag := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatal(err)
		}
		return x.Add(10 * time.Hour)
	}
	faelle := []struct {
		name  string
		heute time.Time
		want  time.Time
	}{
		{"Tag davor: der Termin ist die Frist", tag("2027-06-28"), TagesEndeInSchulzeitzone(tag("2027-06-29"))},
		{"am Termintag: Stichtag des folgenden Schuljahres", tag("2027-06-29"), TagesEndeInSchulzeitzone(tag("2028-07-31"))},
		{"Tag danach: Stichtag des folgenden Schuljahres", tag("2027-06-30"), TagesEndeInSchulzeitzone(tag("2028-07-31"))},
		{"nächstes Schuljahr ohne neuen Plan: Stichtag des laufenden", tag("2027-09-06"), TagesEndeInSchulzeitzone(tag("2028-07-31"))},
	}
	// 9H1 steht zweimal im Plan (28.06. und, hinter dem Wochenende, 05.07.): Am ersten
	// Termintag zählt nicht der Nachzügler-Termin, sondern das folgende Schuljahr.
	faelle = append(faelle, struct {
		name  string
		heute time.Time
		want  time.Time
	}{"Nachzügler-Termin steht noch an: trotzdem das folgende Schuljahr", tag("2027-06-28"), TagesEndeInSchulzeitzone(tag("2028-07-31"))})
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			svc.jetzt = func() time.Time { return fall.heute }
			klasse := "9H2"
			if strings.HasPrefix(fall.name, "Nachzügler") {
				klasse = "9H1"
			}
			got, err := svc.resolveCheckoutDueDate(ctx, schulbuch, klasse)
			if err != nil {
				t.Fatalf("Frist: %v", err)
			}
			if !got.Equal(fall.want) {
				t.Errorf("heute %s: Frist %v, erwartet %v", fall.heute.Format("2006-01-02"),
					got.In(schulzeit.Zone()), fall.want.In(schulzeit.Zone()))
			}
		})
	}
}

// rueckgabePlanJuni2027 veröffentlicht einen Rückgabe-Plan ab Montag, 28.06.2027, eine
// Zeile je Tag: 9H2 ist am Dienstag dran, 9H1 zweimal (28.06. und als Nachzügler).
func rueckgabePlanJuni2027(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	repo := repository.NewLmfTerminRepository(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	ersterTag := time.Date(2027, time.June, 28, 0, 0, 0, 0, schulzeit.Zone())
	zeilen := []repository.LmfPlanZeile{{Klassen: []string{"9H1"}}, {Klassen: []string{"9H2"}}, {Klassen: []string{"10R1"}},
		{Klassen: []string{"10R2"}}, {Klassen: []string{"10R3"}}, {Klassen: []string{"9H1"}, Vermerk: "Nachzügler"}}
	plaetze := lmfplan.VerteileMit(lmfplan.Rahmen{ErsterTag: ersterTag, Startstunde: 1, StundenJeTag: 1},
		make([]*lmfplan.Platz, len(zeilen)), lmfplan.Schultage(nil))
	plan := repository.LmfPlan{Art: repository.LmfTerminRueckgabe, ErsterTag: "2027-06-28", Startstunde: 1, StundenJeTag: 1,
		LetzterTag: plaetze[len(plaetze)-1].Datum.Format("2006-01-02"), LetzteStunde: plaetze[len(plaetze)-1].Stunde}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stand, err := repo.SaveLmfPlanIn(ctx, tx, plan, zeilen, plaetze, nil)
	if err != nil {
		t.Fatalf("Plan speichern: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.VeroeffentlicheLmfPlanIn(ctx, pool, stand.Plan.ID, time.Now()); err != nil {
		t.Fatalf("Plan veröffentlichen: %v", err)
	}

}

// Mehrjahresband (Antwort der Schule vom 22.09.2026, docs/OFFEN.md 9.6): Ein Titel, der
// „bis Jahrgang 11" beim Kind bleibt, bekommt bei einem Kind der 9 den Stichtag zwei
// Schuljahre später — der Rückgabetermin der Klasse geht ihn nichts an. Ist der Termin
// schon vorbei, rechnet die Frist vom folgenden Schuljahr aus, und eines der Jahre steckt
// in diesem Sprung: 31.07.2029, nicht 2030. Rot gesehen am Rückbau von AddDate und -1.
func TestLmfFrist_MehrjahresbandRechnetUeberDenStichtagHinaus(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	rueckgabePlanJuni2027(t, pool)

	svc := &defaultLoanService{pool: pool}
	band := &repository.BookCopy{Titel: "Mathe 9-11", IstLernmittel: true, Medientyp: "Buch", ZielJahrgang: 11}
	tag := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatal(err)
		}
		return x.Add(10 * time.Hour)
	}
	faelle := []struct {
		name   string
		klasse string
		heute  time.Time
		want   time.Time
	}{
		{"Tag vor dem Termin: der Termin zählt nicht, Stichtag plus zwei Jahre", "9H2", tag("2027-06-28"), TagesEndeInSchulzeitzone(tag("2029-07-31"))},
		{"am Termintag: folgendes Schuljahr plus das verbleibende Jahr", "9H2", tag("2027-06-29"), TagesEndeInSchulzeitzone(tag("2029-07-31"))},
		{"Tag danach: dasselbe", "9H2", tag("2027-06-30"), TagesEndeInSchulzeitzone(tag("2029-07-31"))},
		{"Kind der 11: der letzte Jahrgang, ein Schuljahr wie bisher", "ET", tag("2027-06-28"), TagesEndeInSchulzeitzone(tag("2027-07-31"))},
		{"Kind über dem Zieljahrgang: ein Schuljahr", "12T", tag("2027-06-28"), TagesEndeInSchulzeitzone(tag("2027-07-31"))},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			svc.jetzt = func() time.Time { return fall.heute }
			got, err := svc.resolveCheckoutDueDate(ctx, band, fall.klasse)
			if err != nil {
				t.Fatalf("Frist: %v", err)
			}
			if !got.Equal(fall.want) {
				t.Errorf("%s am %s: Frist %v, erwartet %v", fall.klasse, fall.heute.Format("2006-01-02"),
					got.In(schulzeit.Zone()), fall.want.In(schulzeit.Zone()))
			}
		})
	}
}
