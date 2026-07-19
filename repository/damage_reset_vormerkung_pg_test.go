package repository

import (
	"context"
	"testing"
)

// TestReportDamage_ResetsAbholbereiteVormerkung sichert #3 ab: Wird ein Exemplar, das bei der
// Rückgabe gerade einem wartenden Schüler zugewiesen wurde (Vormerkung 'abholbereit'), als
// beschädigt ausgesondert, darf keine Geister-Zuteilung auf ein defektes Buch zurückbleiben.
// Die Vormerkung muss auf 'wartend' zurückfallen und die Exemplar-Bindung gelöst werden.
func TestReportDamage_ResetsAbholbereiteVormerkung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	_, ex := seedSignaturMitExemplaren(t, pool, "DmgVorm", 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	schueler := seedSchueler(t, pool, "DV-1", "Mia", "7a")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, ex[0], schueler, bearbeiter)
	returnLoan(t, pool, loan)

	// Vormerkung, die genau dieses Exemplar als abholbereit bereitstellt.
	var vID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		 VALUES ($1, $2, 'abholbereit', $3, now() + interval '3 days') RETURNING id`,
		titelID, schueler, ex[0]).Scan(&vID); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}

	repo := NewDamageRepository(pool)
	if _, err := repo.ReportDamage(ctx, ex[0], loan, schueler, bearbeiter, "Wasserschaden", 10.0); err != nil {
		t.Fatalf("ReportDamage: %v", err)
	}

	var status string
	var bereitgestellt *string
	if err := pool.QueryRow(ctx,
		`SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen WHERE id = $1`, vID).
		Scan(&status, &bereitgestellt); err != nil {
		t.Fatal(err)
	}
	if status != "wartend" {
		t.Errorf("Vormerkung-Status = %q, erwartet 'wartend' (Schüler zurück in die Warteschlange)", status)
	}
	if bereitgestellt != nil {
		t.Errorf("bereitgestellt_exemplar_id nicht gelöst: %v (zeigt weiter auf defektes Exemplar)", *bereitgestellt)
	}
}
