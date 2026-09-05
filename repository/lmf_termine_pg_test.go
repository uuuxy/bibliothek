package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// Der LMF-Plan am echten Postgres: Klassen laufen durch das Vokabular (Trigger),
// die Liste sortiert nach Zeitpunkt, „alle" hebt die Schuljahresgrenze auf, und der
// Hinweis „ohne Rückgabe-Termin" kennt Schreibvarianten.
func TestLmfTermine_SpeichernListenLoeschen(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewLmfTerminRepository(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_termine WHERE vermerk LIKE 'PGTEST%'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE nachname = 'Lmfplantest'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	// Zwei Termine, bewusst in falscher Reihenfolge und mit Schreibvariante der Klasse.
	spaet, err := repo.SaveLmfTermin(ctx, LmfTermin{Datum: "2027-07-01", Stunde: 5, Art: LmfTerminRueckgabe,
		Klassen: []string{"10r1", " 10R3 ", "10r1"}, Vermerk: "PGTEST spät"})
	if err != nil {
		t.Fatalf("Termin anlegen: %v", err)
	}
	if len(spaet.Klassen) != 2 {
		t.Errorf("Klassen dedupliziert und kanonisiert erwartet, war %v", spaet.Klassen)
	}
	frueh, err := repo.SaveLmfTermin(ctx, LmfTermin{Datum: "2027-06-28", Stunde: 3, Art: LmfTerminRueckgabe,
		Klassen: []string{"9H1"}, Vermerk: "PGTEST früh"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveLmfTermin(ctx, LmfTermin{Datum: "2027-06-28", Stunde: 5, Art: LmfTerminRueckgabe,
		Vermerk: "PGTEST Bücher setzen"}); err != nil {
		t.Fatalf("Termin ohne Klasse: %v", err)
	}

	ab := time.Date(2026, time.August, 1, 0, 0, 0, 0, schulzeit.Zone())
	liste, err := repo.ListLmfTermine(ctx, ab)
	if err != nil {
		t.Fatal(err)
	}
	var eigene []LmfTermin
	for _, e := range liste {
		if len(e.Vermerk) >= 6 && e.Vermerk[:6] == "PGTEST" {
			eigene = append(eigene, e)
		}
	}
	if len(eigene) != 3 || eigene[0].ID != frueh.ID || eigene[2].ID != spaet.ID {
		t.Fatalf("Reihenfolge nach Datum/Stunde erwartet (früh, setzen, spät), war %+v", eigene)
	}
	if len(eigene[1].Klassen) != 0 {
		t.Errorf("Bücher-setzen-Zeile hat keine Klasse, war %v", eigene[1].Klassen)
	}

	// Umschreiben ersetzt die Klassen vollständig.
	spaet.Klassen = []string{"10R2"}
	spaet.Stunde = 6
	if spaet, err = repo.SaveLmfTermin(ctx, spaet); err != nil {
		t.Fatal(err)
	}
	gelesen, err := repo.GetLmfTermin(ctx, spaet.ID)
	if err != nil || len(gelesen.Klassen) != 1 || gelesen.Stunde != 6 {
		t.Errorf("Umschreiben: %+v (%v)", gelesen, err)
	}

	// Schuljahresgrenze: ab August 2027 ist nichts davon mehr zu sehen, „alle" zeigt es.
	spaeter, err := repo.ListLmfTermine(ctx, time.Date(2027, time.August, 1, 0, 0, 0, 0, schulzeit.Zone()))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range spaeter {
		if e.ID == frueh.ID {
			t.Error("ein Termin des alten Schuljahres steht noch in der Liste des neuen")
		}
	}
	alle, err := repo.ListLmfTermine(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alle) < 3 {
		t.Errorf("alle muss die Schuljahresgrenze aufheben, waren %d", len(alle))
	}

	// Hinweis: Klasse mit Schülern, aber ohne Rückgabe-Termin — Schreibvariante zählt als
	// dieselbe Klasse. 9H1 hat einen (in eigener Schreibweise geseedet), 8G1 nicht.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES
		('LMFP-1', 'Neun', 'Lmfplantest', '9h1', 2027),
		('LMFP-2', 'Acht', 'Lmfplantest', '8G1', 2028)`); err != nil {
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
	if !hat["8g1"] || hat["9h1"] {
		t.Errorf("ohne Rückgabe-Termin: %v — erwartet 8G1 drin, 9H1 draußen", ohne)
	}

	// Löschen räumt die Klassen mit (CASCADE) und meldet, ob es etwas gab.
	weg, err := repo.DeleteLmfTermin(ctx, spaet.ID)
	if err != nil || !weg {
		t.Fatalf("Löschen: weg=%v err=%v", weg, err)
	}
	if _, err := repo.GetLmfTermin(ctx, spaet.ID); err != pgx.ErrNoRows {
		t.Errorf("gelöschter Termin noch lesbar: %v", err)
	}
	weg, err = repo.DeleteLmfTermin(ctx, spaet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if weg {
		t.Error("zweites Löschen meldet Erfolg")
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
