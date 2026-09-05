package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// Der LMF-Plan am echten Postgres (Migration 097): Ein Plan wird als Reihenfolge
// gespeichert, seine Zeilen tragen die gerechneten Plätze; Klassen laufen durch das
// Vokabular (Trigger); die Liste sortiert nach Zeitpunkt, „alle" hebt die Schuljahres-
// grenze auf; der Hinweis „ohne Rückgabe-Termin" kennt Schreibvarianten und mahnt
// ausgelassene Klassen nicht an; ein zweites Speichern im selben Schuljahr ersetzt, in
// einem anderen legt es einen neuen Plan an; der neueste gewinnt.
func TestLmfPlan_SpeichernListenAuslassen(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewLmfTerminRepository(pool)
	t.Cleanup(func() {
		raeumeLmfPlaene(t, pool)
		if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE nachname = 'Lmfplantest'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	if _, err := repo.NeuesterLmfPlan(ctx, LmfTerminRueckgabe); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("ohne Plan erwartet ErrNoRows, war %v", err)
	}

	// Donnerstag 11.06.2026 ab 3. Stunde, 6 je Tag — wie der echte Plan der Schule.
	// Schreibvarianten und eine Dublette in einer Zeile; eine Zeile ohne Klasse.
	st := speicherePlan(t, repo, LmfTerminRueckgabe, "2026-06-11", 3, 6,
		[]LmfPlanZeile{
			{Klassen: []string{"9h1"}}, {Klassen: []string{"9H2"}, Vermerk: "bis 11. eingesammelt"},
			{Klassen: []string{"10r1", " 10R2 ", "10r1"}}, {Klassen: []string{"10R3"}},
			{Klassen: []string{"8H1"}}, {Vermerk: "Bücher setzen"},
		}, []string{"q1", "12T1"})
	if len(st.Zeilen) != 6 {
		t.Fatalf("%d Zeilen gespeichert", len(st.Zeilen))
	}
	if z := st.Zeilen[2]; len(z.Klassen) != 2 || z.Datum != "2026-06-11" || z.Stunde != 5 {
		t.Errorf("Zeile 3: Klassen dedupliziert/kanonisiert und Platz Do 5. Std. erwartet, war %+v", z)
	}
	if z := st.Zeilen[4]; z.Datum != "2026-06-12" || z.Stunde != 1 {
		t.Errorf("Zeile 5 muss am Freitag in der 1. Stunde liegen, war %+v", z)
	}
	if len(st.Ausgelassen) != 2 {
		t.Errorf("Auslassungen: %v", st.Ausgelassen)
	}

	// Liste ab Schuljahresbeginn: alle sechs, sortiert nach Platz; das Vokabular hat
	// die Schreibweise vereinheitlicht („9h1" → registrierte Form).
	ab := time.Date(2025, time.August, 1, 0, 0, 0, 0, schulzeit.Zone())
	liste, err := repo.ListLmfTermine(ctx, ab)
	if err != nil {
		t.Fatal(err)
	}
	if len(liste) != 6 || liste[0].Stunde != 3 || liste[5].Vermerk != "Bücher setzen" || len(liste[5].Klassen) != 0 {
		t.Fatalf("Liste: %+v", liste)
	}
	if klassenNorm(liste[0].Klassen[0]) != "9h1" {
		t.Errorf("Klasse der ersten Zeile: %v", liste[0].Klassen)
	}
	spaeter, err := repo.ListLmfTermine(ctx, time.Date(2026, time.August, 1, 0, 0, 0, 0, schulzeit.Zone()))
	if err != nil || len(spaeter) != 0 {
		t.Errorf("ab August 2026 darf nichts mehr zu sehen sein: %d (%v)", len(spaeter), err)
	}
	alle, err := repo.ListLmfTermine(ctx, time.Time{})
	if err != nil || len(alle) != 6 {
		t.Errorf("alle hebt die Grenze auf: %d (%v)", len(alle), err)
	}

	// Hinweis „ohne Rückgabe-Termin": 9H1 hat einen (andere Schreibweise), 8G1 nicht,
	// Q1 ist ausgelassen und wird nicht angemahnt, 12T1 ebenso.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES
		('LMFP-1', 'Neun', 'Lmfplantest', '09H1', 2027),
		('LMFP-2', 'Acht', 'Lmfplantest', '8G1', 2028),
		('LMFP-3', 'Zwoelf', 'Lmfplantest', '12T1', 2028),
		('LMFP-4', 'Elf', 'Lmfplantest', '11G1', 2028)`); err != nil {
		t.Fatal(err)
	}
	ohne, err := repo.KlassenOhneRueckgabeTermin(ctx, ab)
	if err != nil {
		t.Fatal(err)
	}
	hat := map[string]bool{}
	for _, k := range ohne {
		hat[klassenNorm(k)] = true
	}
	if !hat["8g1"] || !hat["11g1"] || hat["9h1"] || hat["12t1"] {
		t.Errorf("ohne Rückgabe-Termin: %v — erwartet 8G1 und 11G1 drin, 9H1 und 12T1 (ausgelassen) draußen", ohne)
	}

	// Vorschlags-Reihenfolge: Abschlussklasse zuerst, dann Jahrgang absteigend; ab 11 Oberstufe.
	reihe, err := repo.KlassenMitSchuelern(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(reihe) != 4 || klassenNorm(reihe[0].Name) != "9h1" || !reihe[0].Abschluss ||
		klassenNorm(reihe[1].Name) != "12t1" || !reihe[1].Oberstufe ||
		klassenNorm(reihe[2].Name) != "11g1" || !reihe[2].Oberstufe ||
		klassenNorm(reihe[3].Name) != "8g1" || reihe[3].Oberstufe {
		t.Errorf("Reihenfolge: %+v", reihe)
	}

	// Zweites Speichern im selben Schuljahr ERSETZT (eine Zeile weniger, andere Startstunde).
	st2 := speicherePlan(t, repo, LmfTerminRueckgabe, "2026-06-15", 1, 6,
		[]LmfPlanZeile{{Klassen: []string{"9H1"}}}, nil)
	if st2.Plan.ID != st.Plan.ID {
		t.Errorf("gleiches Schuljahr muss denselben Plan umschreiben: %s ≠ %s", st2.Plan.ID, st.Plan.ID)
	}
	neuester, err := repo.NeuesterLmfPlan(ctx, LmfTerminRueckgabe)
	if err != nil || len(neuester.Zeilen) != 1 || neuester.Plan.Startstunde != 1 || len(neuester.Ausgelassen) != 0 {
		t.Errorf("umgeschriebener Plan: %+v (%v)", neuester, err)
	}
	// Anderes Schuljahr legt einen NEUEN Plan an; der neueste gewinnt.
	st3 := speicherePlan(t, repo, LmfTerminRueckgabe, "2027-06-14", 1, 6,
		[]LmfPlanZeile{{Klassen: []string{"9H1"}}, {Klassen: []string{"9H2"}}}, nil)
	if st3.Plan.ID == st.Plan.ID {
		t.Error("neues Schuljahr muss einen neuen Plan anlegen")
	}
	if neuester, err = repo.NeuesterLmfPlan(ctx, LmfTerminRueckgabe); err != nil || neuester.Plan.ID != st3.Plan.ID {
		t.Errorf("neuester Plan: %+v (%v)", neuester.Plan, err)
	}
	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM lmf_plaene WHERE art = 'rueckgabe'`).Scan(&anzahl); err != nil || anzahl != 2 {
		t.Errorf("zwei Rückgabe-Pläne erwartet, %d (%v)", anzahl, err)
	}

	// Löschen räumt Zeilen und Klassen mit (CASCADE) und meldet, ob es etwas gab.
	weg, err := repo.DeleteLmfPlan(ctx, st3.Plan.ID)
	if err != nil || !weg {
		t.Fatalf("Löschen: weg=%v err=%v", weg, err)
	}
	if weg, err = repo.DeleteLmfPlan(ctx, st3.Plan.ID); err != nil || weg {
		t.Errorf("zweites Löschen: weg=%v err=%v", weg, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM lmf_termine WHERE plan_id = $1`, st3.Plan.ID).Scan(&anzahl); err != nil || anzahl != 0 {
		t.Errorf("Zeilen des gelöschten Plans: %d (%v)", anzahl, err)
	}
}

// Ferien und Schließzeiten, die den Zeitraum berühren.
func TestLmfPlan_FreieTage(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewLmfTerminRepository(pool)
	if _, err := pool.Exec(ctx, `INSERT INTO ferien_schliesszeiten (bezeichnung, start_datum, end_datum) VALUES
		('PGTEST Pfingsten', '2026-05-26', '2026-05-29'), ('PGTEST Sommer', '2026-06-29', '2026-08-07')`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM ferien_schliesszeiten WHERE bezeichnung LIKE 'PGTEST%'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	von := time.Date(2026, time.June, 11, 0, 0, 0, 0, schulzeit.Zone())
	frei, err := repo.FreieTage(ctx, von, von.AddDate(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(frei) != 1 || frei[0].Von.Format("2006-01-02") != "2026-06-29" {
		t.Errorf("nur die Sommerferien berühren Juni/Juli: %+v", frei)
	}
}

// klassenNorm spiegelt klassen_normkey für den Vergleich im Test.
func klassenNorm(k string) string {
	var b []byte
	for i := 0; i < len(k); i++ {
		c := k[i]
		if c == ' ' {
			continue
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b = append(b, c)
	}
	for len(b) > 1 && b[0] == '0' && b[1] >= '0' && b[1] <= '9' {
		b = b[1:]
	}
	return string(b)
}
