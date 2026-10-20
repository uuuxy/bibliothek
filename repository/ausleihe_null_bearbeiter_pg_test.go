package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"testing"
)

// bearbeiter_id kann NULL sein (ON DELETE SET NULL, und die Anonymisierung leert die Spalte).
// Als Text im Modell scheiterte leseAusleihe mit „cannot scan NULL into *string", und das
// Scannen oder Zurückgeben einer Ausleihe ohne Bearbeiter endete mit 500. BearbeiterID ist
// deshalb ein Zeiger.
func TestFindeOffene_NullBearbeiterKein500(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx,
		`TRUNCATE buecher_exemplare, buecher_titel, ausleihen, leser, benutzer, ausweisnummern_ausgeschieden
		 RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset: %v", err)
	}

	schuelerID := legeSchuelerAn(t, pool, "LOAN-NB", "Nulli", "7a")

	var titelID, copyID string
	titelID = testdaten.LegeTitelAn(t, pool, "NB-Titel", 5)
	copyID = testdaten.LegeExemplarAn(t, pool, titelID, "NB-B-1", true, "")

	// Eine Ausleihe ohne bearbeiter_id (NULL).
	if _, err := pool.Exec(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist) VALUES ($1, $2, CURRENT_DATE + 14)`,
		copyID, schuelerID); err != nil {
		t.Fatalf("Ausleihe ohne Bearbeiter: %v", err)
	}

	loan, err := NewAusleihAbfragen(pool).FindeOffene(ctx, copyID)
	if err != nil {
		t.Fatalf("AusleihAbfragen.FindeOffene darf bei NULL bearbeiter_id nicht scheitern (500-Regression): %v", err)
	}
	if loan == nil {
		t.Fatal("aktive Ausleihe unerwartet nil")
	}
	if loan.BearbeiterID != nil {
		t.Fatalf("BearbeiterID muss nil sein bei NULL-Spalte, war: %q", *loan.BearbeiterID)
	}
}
