package repository

import (
	"context"
	"testing"
)

// TestMahnbriefe_ZurueckgegebeneRausfiltern: Gibt ein Schüler sein Buch zwischen dem Laden
// der Mahnliste und dem Druck zurück, steht die Ausleihe nicht mehr auf dem Mahnbrief (und
// ihre Mahnstufe steigt nicht). Geprüft wird die Abfrage, die der Druck aus der Auswahl in
// seiner Transaktion ruft (api/mahnwesen/mahnbriefe.go).
func TestMahnbriefe_ZurueckgegebeneRausfiltern(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "MahnTest", 2)
	schueler := legeSchuelerAn(t, pool, "M-1", "Tom", "7a")
	bearbeiter := legeBearbeiterAn(t, pool)

	loanOffen := legeAusleiheAn(t, pool, ex[0], schueler, bearbeiter)
	loanZurueck := legeAusleiheAn(t, pool, ex[1], schueler, bearbeiter)
	// Beide Fristen sind abgelaufen: Die Abfrage nennt nur überfällige Bücher, und der
	// Unterschied der zwei Ausleihen soll allein die Rückgabe sein.
	if _, err := pool.Exec(ctx,
		`UPDATE ausleihen SET rueckgabe_frist = CURRENT_TIMESTAMP - interval '30 days' WHERE id = ANY($1)`,
		[]string{loanOffen, loanZurueck}); err != nil {
		t.Fatalf("Fristen ablaufen lassen: %v", err)
	}
	bucheRueckgabe(t, pool, loanZurueck) // dieses Buch ist schon zurück

	repo := NewMahnwesenAbfragen(pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Rollback nach dem Lesen ist hier der Normalfall
	briefe, err := repo.LadeMahnbriefe(ctx, tx, []string{loanOffen, loanZurueck})
	if err != nil {
		t.Fatalf("LadeMahnbriefe: %v", err)
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

// TestMahnbriefe_JederSchuelerEinBrief: Die Abfrage liefert je Buch eine Zeile; je Schüler entsteht
// genau ein Brief mit allen seinen Büchern. Ein Brief, der die Bücher eines anderen nennt, ginge
// mit fremden Ausleihen an die Eltern.
func TestMahnbriefe_JederSchuelerEinBrief(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "MahnBrief", 3)
	anna := legeSchuelerAn(t, pool, "MB-ANNA", "Anna", "7a")
	ben := legeSchuelerAn(t, pool, "MB-BEN", "Ben", "7b")
	bearbeiter := legeBearbeiterAn(t, pool)
	ausleihen := []string{
		legeAusleiheAn(t, pool, ex[0], anna, bearbeiter),
		legeAusleiheAn(t, pool, ex[1], anna, bearbeiter),
		legeAusleiheAn(t, pool, ex[2], ben, bearbeiter),
	}
	if _, err := pool.Exec(ctx,
		`UPDATE ausleihen SET rueckgabe_frist = CURRENT_TIMESTAMP - interval '30 days' WHERE id = ANY($1)`,
		ausleihen); err != nil {
		t.Fatalf("Fristen ablaufen lassen: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Rollback nach dem Lesen ist hier der Normalfall
	briefe, err := NewMahnwesenAbfragen(pool).LadeMahnbriefe(ctx, tx, ausleihen)
	if err != nil {
		t.Fatalf("LadeMahnbriefe: %v", err)
	}

	buecherJe := map[string]int{}
	for _, b := range briefe {
		buecherJe[b.SchuelerID] += len(b.Buecher)
	}
	if len(briefe) != 2 || buecherJe[anna] != 2 || buecherJe[ben] != 1 {
		t.Errorf("Briefe = %d, Bücher je Schüler %v; erwartet je Schüler einen Brief (Anna 2 Bücher, Ben 1)", len(briefe), buecherJe)
	}
}
