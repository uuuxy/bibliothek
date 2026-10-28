package repository

import (
	"context"
	"testing"
)

// Wird ein Exemplar, das bei der Rückgabe gerade einem wartenden Schüler zugewiesen wurde
// (Vormerkung abholbereit), als beschädigt ausgesondert, bleibt keine Zuteilung auf ein Buch
// zurück, das es nicht mehr gibt: Die Vormerkung fällt auf wartend zurück und verliert das
// Exemplar.
func TestSchadenMelden_SetztAbholbereiteVormerkungZurueck(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "DmgVorm", 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	schueler := legeSchuelerAn(t, pool, "DV-1", "Mia", "7a")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, ex[0], schueler, bearbeiter)
	bucheRueckgabe(t, pool, loan)

	// Vormerkung, die genau dieses Exemplar als abholbereit bereitstellt.
	var vID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		 VALUES ($1, $2, 'abholbereit', $3, now() + interval '3 days') RETURNING id`,
		titelID, schueler, ex[0]).Scan(&vID); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}

	repo := NewSchadensAbfragen(pool)
	if _, err := repo.Melde(ctx, loan, bearbeiter, "Wasserschaden", SchadensArtBeschaedigt, 1000); err != nil {
		t.Fatalf("Melde: %v", err)
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
