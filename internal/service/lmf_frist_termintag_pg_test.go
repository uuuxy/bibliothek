package service

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Frist eines Schulbuchs um den Büchertausch, am echten Postgres und mit fester Uhr: Vor
// dem Tausch ist der Termin der Klasse die Frist. Ab dem ersten Tag des Tauschs gibt jedes Kind
// ein neu ausgegebenes Schulbuch erst im nächsten Schuljahr zurück, auch aus einer Klasse, die
// nicht im Plan steht, und auch vor dem eigenen Termin. Wäre es der Termin selbst oder der
// 31.07. des laufenden Schuljahres, ein Tag in den Ferien, stünde das Kind nach den Ferien
// überfällig da und wäre nach 14 Tagen gesperrt.
func TestLMFFrist_AbBeginnDesBuechertauschsGiltDasFolgendeSchuljahr(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	rueckgabePlanJuni2027(t, pool)

	svc := &AusleihDienst{pool: pool}
	schulbuch := &repository.Exemplar{Titel: "Mathe 9", IstLernmittel: true, Medientyp: "Buch"}
	tag := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatal(err)
		}
		return x.Add(10 * time.Hour)
	}
	laufend := schulzeit.TagesEnde(tag("2027-07-31"))
	folgend := schulzeit.TagesEnde(tag("2028-07-31"))
	faelle := []struct {
		name   string
		klasse string
		heute  time.Time
		want   time.Time
	}{
		{"Freitag vor dem Tausch: der Termin ist die Frist", "9H2", tag("2027-06-25"), schulzeit.TagesEnde(tag("2027-06-29"))},
		{"Freitag vor dem Tausch, Klasse nicht im Plan: Stichtag des laufenden Schuljahres", "5F1", tag("2027-06-25"), laufend},
		{"erster Tag des Tauschs, Termin der Klasse steht noch an: folgendes Schuljahr", "9H2", tag("2027-06-28"), folgend},
		{"erster Tag des Tauschs, Klasse nicht im Plan: folgendes Schuljahr", "5F1", tag("2027-06-28"), folgend},
		{"am Termintag: folgendes Schuljahr", "9H2", tag("2027-06-29"), folgend},
		{"Tag danach: folgendes Schuljahr", "9H2", tag("2027-06-30"), folgend},
		// 9H1 steht zweimal im Plan (28.06. und, hinter dem Wochenende, 05.07.).
		{"Nachzügler-Termin steht noch an: folgendes Schuljahr", "9H1", tag("2027-06-28"), folgend},
		{"nach der letzten Zeile, vor dem Stichtag: folgendes Schuljahr", "5F1", tag("2027-07-20"), folgend},
		{"nächstes Schuljahr ohne neuen Plan: Stichtag des laufenden", "9H2", tag("2027-09-06"), folgend},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			svc.jetzt = func() time.Time { return fall.heute }
			got, err := svc.bestimmeFrist(ctx, schulbuch, fall.klasse)
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

// rueckgabePlanJuni2027 veröffentlicht einen Rückgabe-Plan ab Montag, 28.06.2027, eine
// Zeile je Tag: 9H2 ist am Dienstag dran, 9H1 zweimal (28.06. und als Nachzügler).
func rueckgabePlanJuni2027(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	repo := repository.NewLMFAbfragen(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	ersterTag := time.Date(2027, time.June, 28, 0, 0, 0, 0, schulzeit.Zone())
	zeilen := []repository.LMFPlanZeile{{Klassen: []string{"9H1"}}, {Klassen: []string{"9H2"}}, {Klassen: []string{"10R1"}},
		{Klassen: []string{"10R2"}}, {Klassen: []string{"10R3"}}, {Klassen: []string{"9H1"}, Vermerk: "Nachzügler"}}
	plaetze := lmfplan.VerteileMit(lmfplan.Rahmen{ErsterTag: ersterTag, Startstunde: 1, StundenJeTag: 1},
		make([]*lmfplan.Platz, len(zeilen)), lmfplan.Schultage(nil))
	plan := repository.LMFPlan{Art: repository.LMFTerminRueckgabe, ErsterTag: "2027-06-28", Startstunde: 1, StundenJeTag: 1,
		LetzterTag: plaetze[len(plaetze)-1].Datum.Format("2006-01-02"), LetzteStunde: plaetze[len(plaetze)-1].Stunde}
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
	if _, err := repo.VeroeffentlichePlan(ctx, pool, stand.Plan.ID, time.Now()); err != nil {
		t.Fatalf("Plan veröffentlichen: %v", err)
	}

}

// Mehrjahresband (Antwort der Schule): Ein Titel mit Spanne bis Jahrgang 11 und gesetztem
// Schalter bekommt bei einem Kind der 9 den Stichtag zwei Schuljahre später — der
// Rückgabetermin der Klasse geht ihn nichts an. Hat der Büchertausch begonnen, rechnet die Frist
// vom folgenden Schuljahr aus, und eines der Jahre steckt in diesem Sprung: 31.07.2029, nicht
// 2030. Rot gesehen am Rückbau von AddDate und -1.
func TestLMFFrist_MehrjahresbandRechnetUeberDenStichtagHinaus(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	rueckgabePlanJuni2027(t, pool)

	svc := &AusleihDienst{pool: pool}
	band := &repository.Exemplar{Titel: "Mathe 9-11", IstLernmittel: true, Medientyp: "Buch", Mehrjahresband: true, JahrgangBis: 11}
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
		{"vor dem Tausch: der Termin zählt nicht, Stichtag plus zwei Jahre", "9H2", tag("2027-06-25"), schulzeit.TagesEnde(tag("2029-07-31"))},
		{"erster Tag des Tauschs: folgendes Schuljahr plus das verbleibende Jahr", "9H2", tag("2027-06-28"), schulzeit.TagesEnde(tag("2029-07-31"))},
		{"am Termintag: dasselbe", "9H2", tag("2027-06-29"), schulzeit.TagesEnde(tag("2029-07-31"))},
		{"Tag danach: dasselbe", "9H2", tag("2027-06-30"), schulzeit.TagesEnde(tag("2029-07-31"))},
		{"Kind der 11 vor dem Tausch: der letzte Jahrgang, ein Schuljahr wie bisher", "ET", tag("2027-06-25"), schulzeit.TagesEnde(tag("2027-07-31"))},
		{"Kind der 11 im Tausch: das folgende Schuljahr", "ET", tag("2027-06-28"), schulzeit.TagesEnde(tag("2028-07-31"))},
		{"Kind über der Spanne vor dem Tausch: ein Schuljahr", "12T", tag("2027-06-25"), schulzeit.TagesEnde(tag("2027-07-31"))},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			svc.jetzt = func() time.Time { return fall.heute }
			got, err := svc.bestimmeFrist(ctx, band, fall.klasse)
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

// Der Rückgabetermin der Klasse und die Jahre des Mehrjahresbands hängen an zwei Angaben des
// Titels: Ohne das Kästchen „Mehrjahresband" ist ein Schulbuch mit Spanne bis Jahrgang 11 ein
// gewöhnliches Schulbuch und folgt dem Termin der Klasse. Ein Buch der Bücherei geht der
// Termin nichts an, es bekommt die Leihfrist für Bücher.
func TestLMFFrist_OhneKaestchenUndBuechereiBuch(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	rueckgabePlanJuni2027(t, pool)

	// Freitag vor dem Büchertausch, der am Montag beginnt; der Termin der 9H2 ist der Dienstag.
	heute := time.Date(2027, time.June, 25, 10, 0, 0, 0, schulzeit.Zone())
	termin := schulzeit.TagesEnde(time.Date(2027, time.June, 29, 10, 0, 0, 0, schulzeit.Zone()))
	svc := &AusleihDienst{pool: pool, jetzt: func() time.Time { return heute }}

	ohneKaestchen := &repository.Exemplar{Titel: "Mathe 9-11", IstLernmittel: true, Medientyp: "Buch", JahrgangBis: 11}
	got, err := svc.bestimmeFrist(ctx, ohneKaestchen, "9H2")
	if err != nil {
		t.Fatalf("Frist: %v", err)
	}
	if !got.Equal(termin) {
		t.Errorf("Schulbuch mit Spanne, ohne Kästchen: Frist %v, erwartet den Termin der Klasse %v",
			got.In(schulzeit.Zone()), termin.In(schulzeit.Zone()))
	}

	buecherei := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch"}
	got, err = svc.bestimmeFrist(ctx, buecherei, "9H2")
	if err != nil {
		t.Fatalf("Frist: %v", err)
	}
	if !got.After(termin.AddDate(0, 0, 7)) {
		t.Errorf("Buch der Bücherei: Frist %v — erwartet die Leihfrist für Bücher, nicht den Termin der Klasse %v",
			got.In(schulzeit.Zone()), termin.In(schulzeit.Zone()))
	}
}
