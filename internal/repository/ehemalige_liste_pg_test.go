package repository

import (
	"context"
	"testing"
)

// Der Reiter „Ehemalige" und „Aktive Schüler" sind dieselbe Abfrage mit
// umgekehrtem Vorzeichen — hier der Beweis, dass keine Zeile in beiden oder in keiner
// Liste landet: aktiv, weggegangen, gelöscht. Eine anonymisierte Zeile steht in keiner der
// beiden: Sie ist keine Person mehr und wartet nur auf die endgültige Löschung.
func TestLadeEhemaligenListe_GegenstueckDerAktiven(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := &LeserAbfragen{db: pool}

	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, deleted_at) VALUES
		('EHEM-A', 'Aktiv',      'Ehemtest', '07G1', 2029, false, NULL),
		('EHEM-W', 'Weg',        'Ehemtest', 'ABG',  2026, true,  NULL),
		('EHEM-X', 'Weglaenger', 'Ehemtest', 'ABG',  2025, true,  NULL),
		('EHEM-G', 'Geloescht',  'Ehemtest', 'ABG',  2026, true,  now())`); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	// Der Nachname bleibt Ehemtest, damit die Suche die Zeile träfe: Ausgeschlossen wird sie
	// allein wegen anonymized_at.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger,
		                      ist_gesperrt, block_reason, anonymized_at)
		VALUES ('ANON-EHEM-N', 'Anonymisiert', 'Ehemtest', '', 2026, true, true, 'Anonymisiert (DSGVO)', now())`); err != nil {
		t.Fatalf("Seed anonymisiert: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE nachname = 'Ehemtest'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	ehemalige, err := repo.LadeEhemaligenListe(ctx, "Ehemtest", LeserSortierung{})
	if err != nil {
		t.Fatalf("LadeEhemaligenListe: %v", err)
	}
	aktive, err := repo.LadeSchuelerListe(ctx, nil, "Ehemtest", LeserSortierung{})
	if err != nil {
		t.Fatalf("LadeSchuelerListe: %v", err)
	}

	namen := func(l []LeserListenZeile) map[string]bool {
		m := map[string]bool{}
		for _, s := range l {
			m[s.Vorname] = true
		}
		return m
	}
	e, a := namen(ehemalige), namen(aktive)
	if !e["Weg"] || !e["Weglaenger"] || e["Aktiv"] || e["Geloescht"] || e["Anonymisiert"] || len(ehemalige) != 2 {
		t.Errorf("Ehemalige: %v — erwartet genau Weg und Weglaenger", e)
	}
	if !a["Aktiv"] || a["Weg"] || a["Geloescht"] || a["Anonymisiert"] || len(aktive) != 1 {
		t.Errorf("Aktive: %v — erwartet genau Aktiv", a)
	}

	// Ohne Suchbegriff: jüngster Abgang zuerst, die anonymisierte Zeile auch hier nicht.
	alle, err := repo.LadeEhemaligenListe(ctx, "", LeserSortierung{})
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	for i, s := range alle {
		if s.Nachname == "Ehemtest" {
			pos[s.Vorname] = i
		}
	}
	if _, drin := pos["Anonymisiert"]; drin {
		t.Error("Ehemalige ohne Suchbegriff: die anonymisierte Zeile steht in der Liste")
	}
	if pos["Weg"] > pos["Weglaenger"] {
		t.Errorf("Reihenfolge: Abgang 2026 muss vor 2025 stehen, war %v", pos)
	}
}
