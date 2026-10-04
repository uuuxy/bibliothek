package repository

import (
	"context"
	"testing"
)

// TestMahnbriefe_ZurueckgegebeneRausfiltern: Gibt ein Schüler sein Buch zwischen dem Laden
// der Mahnliste und dem Druck zurück, steht die Ausleihe nicht mehr auf dem Mahnbrief (und
// ihre Mahnstufe steigt nicht). Geprüft wird die Abfrage, die der Druck aus der Auswahl in
// seiner Transaktion ruft (api/mahnwesen_bulk.go).
func TestMahnbriefe_ZurueckgegebeneRausfiltern(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "MahnTest", 2)
	schueler := seedSchueler(t, pool, "M-1", "Tom", "7a")
	bearbeiter := seedBearbeiter(t, pool)

	loanOffen := seedAusleihe(t, pool, ex[0], schueler, bearbeiter)
	loanZurueck := seedAusleihe(t, pool, ex[1], schueler, bearbeiter)
	// Beide Fristen sind abgelaufen: Die Abfrage nennt nur überfällige Bücher, und der
	// Unterschied der zwei Ausleihen soll allein die Rückgabe sein.
	if _, err := pool.Exec(ctx,
		`UPDATE ausleihen SET rueckgabe_frist = CURRENT_TIMESTAMP - interval '30 days' WHERE id = ANY($1)`,
		[]string{loanOffen, loanZurueck}); err != nil {
		t.Fatalf("Fristen ablaufen lassen: %v", err)
	}
	returnLoan(t, pool, loanZurueck) // dieses Buch ist schon zurück

	repo := NewMahnwesenRepository(pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Rollback nach dem Lesen ist hier der Normalfall
	briefe, err := repo.MahnbriefeTx(ctx, tx, []string{loanOffen, loanZurueck})
	if err != nil {
		t.Fatalf("MahnbriefeTx: %v", err)
	}

	// Nur die noch offene Ausleihe darf auftauchen.
	var buecher int
	for _, b := range briefe {
		buecher += len(b.Buecher)
	}
	if len(briefe) != 1 || buecher != 1 {
		t.Errorf("erwartet 1 Brief mit 1 Buch (nur die offene Ausleihe), waren %d Briefe mit %d Büchern — "+
			"eine zurückgegebene Ausleihe landete in der Mahnung", len(briefe), buecher)
	}
}
