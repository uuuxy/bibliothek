package repository

import (
	"context"
	"testing"
)

// TestQueryUeberfaellige_ZurueckgegebeneRausfiltern sichert #4 ab: Gibt ein Schüler
// sein Buch zwischen dem Aufbereiten der Mahnliste und dem Druck zurück, darf die
// bereits zurückgegebene Ausleihe nicht mehr in der Mahnung erscheinen (und ihre
// Mahnstufe nicht steigen). Die Query muss rueckgabe_am IS NULL prüfen.
func TestQueryUeberfaellige_ZurueckgegebeneRausfiltern(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	_, ex := seedSignaturMitExemplaren(t, pool, "MahnTest", 2)
	schueler := seedSchueler(t, pool, "M-1", "Tom", "7a")
	bearbeiter := seedBearbeiter(t, pool)

	loanOffen := seedAusleihe(t, pool, ex[0], schueler, bearbeiter)
	loanZurueck := seedAusleihe(t, pool, ex[1], schueler, bearbeiter)
	returnLoan(t, pool, loanZurueck) // dieses Buch ist schon zurück

	repo := NewMahnwesenRepository(pool)
	klassen, err := repo.QueryUeberfaelligeByAusleiheIDs(ctx, []string{loanOffen, loanZurueck})
	if err != nil {
		t.Fatalf("QueryUeberfaelligeByAusleiheIDs: %v", err)
	}

	// Nur die noch offene Ausleihe darf auftauchen.
	var medien int
	for _, k := range klassen {
		for _, s := range k.Schueler {
			medien += len(s.Medien)
		}
	}
	if medien != 1 {
		t.Errorf("erwartet 1 gemahntes Medium (nur die offene Ausleihe), waren %d — "+
			"eine zurückgegebene Ausleihe landete in der Mahnung", medien)
	}
}
