package repository

import (
	"context"
	"testing"
)

// Der Reiter „Ehemalige / Archiv" und „Aktive Schüler" sind dieselbe Abfrage mit
// umgekehrtem Vorzeichen — hier der Beweis, dass keine Zeile in beiden oder in keiner
// Liste landet: aktiv, weggegangen, gelöscht.
func TestListEhemaligeWithStats_GegenstueckDerAktiven(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := &pgStudentRepository{db: pool}

	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, deleted_at) VALUES
		('EHEM-A', 'Aktiv',      'Ehemtest', '07G1', 2029, false, NULL),
		('EHEM-W', 'Weg',        'Ehemtest', 'ABG',  2026, true,  NULL),
		('EHEM-X', 'Weglaenger', 'Ehemtest', 'ABG',  2025, true,  NULL),
		('EHEM-G', 'Geloescht',  'Ehemtest', 'ABG',  2026, true,  now())`); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE nachname = 'Ehemtest'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	ehemalige, err := repo.ListEhemaligeWithStats(ctx, "Ehemtest")
	if err != nil {
		t.Fatalf("ListEhemaligeWithStats: %v", err)
	}
	aktive, err := repo.ListStudentsWithStats(ctx, "", "Ehemtest")
	if err != nil {
		t.Fatalf("ListStudentsWithStats: %v", err)
	}

	namen := func(l []StudentListStat) map[string]bool {
		m := map[string]bool{}
		for _, s := range l {
			m[s.Vorname] = true
		}
		return m
	}
	e, a := namen(ehemalige), namen(aktive)
	if !e["Weg"] || !e["Weglaenger"] || e["Aktiv"] || e["Geloescht"] || len(ehemalige) != 2 {
		t.Errorf("Ehemalige: %v — erwartet genau Weg und Weglaenger", e)
	}
	if !a["Aktiv"] || a["Weg"] || a["Geloescht"] || len(aktive) != 1 {
		t.Errorf("Aktive: %v — erwartet genau Aktiv", a)
	}

	// Ohne Suchbegriff: jüngster Abgang zuerst.
	alle, err := repo.ListEhemaligeWithStats(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	for i, s := range alle {
		if s.Nachname == "Ehemtest" {
			pos[s.Vorname] = i
		}
	}
	if pos["Weg"] > pos["Weglaenger"] {
		t.Errorf("Reihenfolge: Abgang 2026 muss vor 2025 stehen, war %v", pos)
	}
}
