package repository

import (
	"context"
	"testing"
)

// Regression: bearbeiter_id ist in der DB NULLBAR (ON DELETE SET NULL + der GDPR-
// Anonymisierungs-Job setzt sie auf NULL). Als nicht-nullbarer string im Loan-Modell
// scheiterte scanLoan mit "cannot scan NULL into *string" → HTTP 500 im Kiosk beim
// Scannen/Rückgeben einer Ausleihe ohne Bearbeiter (genau der Prod-500 auf /api/action).
// BearbeiterID ist jetzt *string.
func TestGetActiveLoanByCopyID_NullBearbeiterKein500(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx,
		`TRUNCATE buecher_exemplare, buecher_titel, ausleihen, schueler, benutzer RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset: %v", err)
	}

	schuelerID := seedSchueler(t, pool, "LOAN-NB", "Nulli", "7a")

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

	// Ausleihe OHNE bearbeiter_id (NULL) — exakt der Fall aus dem Prod-Log.
	if _, err := pool.Exec(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist) VALUES ($1, $2, CURRENT_DATE + 14)`,
		copyID, schuelerID); err != nil {
		t.Fatalf("Ausleihe ohne Bearbeiter: %v", err)
	}

	loan, err := NewLoanRepository(pool).GetActiveLoanByCopyID(ctx, copyID)
	if err != nil {
		t.Fatalf("GetActiveLoanByCopyID darf bei NULL bearbeiter_id nicht scheitern (500-Regression): %v", err)
	}
	if loan == nil {
		t.Fatal("aktive Ausleihe unerwartet nil")
	}
	if loan.BearbeiterID != nil {
		t.Fatalf("BearbeiterID muss nil sein bei NULL-Spalte, war: %q", *loan.BearbeiterID)
	}
}
