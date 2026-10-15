package repository

import (
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
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('NB-Titel') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'NB-B-1') RETURNING id`,
		titelID).Scan(&copyID); err != nil {
		t.Fatalf("Exemplar: %v", err)
	}

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
