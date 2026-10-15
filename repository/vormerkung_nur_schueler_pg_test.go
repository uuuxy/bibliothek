package repository

import (
	"context"
	"errors"
	"testing"
)

// Vormerken lässt sich nur für Schüler, und das sagt LegeAn, nicht nur die Oberfläche. Die
// Warteschlange liest an vier Stellen die Sicht schueler, und die Oberfläche bietet zum
// Vormerken nur Schüler an (/api/schueler?q= ohne art=alle). Eine Vormerkung für einen Kollegen
// stünde ohne Namen in der Liste, und bei der Rückgabe ginge die Warteschlange über sie hinweg:
// für immer „wartend", ohne dass es jemand merkt.
func TestVormerkungLegeAn_NurFuerSchueler(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "NurSchueler", 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	kollege := legeKollegenAn(t, pool, "A-VM-K1", "Kim")
	schueler := legeSchuelerAn(t, pool, "VM-S1", "Mia", "7a")
	repo := NewVormerkungsAbfragen(pool)

	if _, err := repo.LegeAn(ctx, titelID, "", kollege); !errors.Is(err, ErrVormerkungNurFuerSchueler) {
		t.Errorf("Vormerkung für einen Kollegen: err = %v — want ErrVormerkungNurFuerSchueler", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM vormerkungen WHERE schueler_id = $1`, kollege).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d Vormerkung(en) für den Kollegen geschrieben — die Warteschlange bedient sie nie", n)
	}

	// Eine ID, die es nicht gibt, bekommt dieselbe Antwort und keinen Fehler des
	// Fremdschlüssels (500).
	if _, err := repo.LegeAn(ctx, titelID, "", "00000000-0000-4000-8000-000000000001"); !errors.Is(err, ErrVormerkungNurFuerSchueler) {
		t.Errorf("Vormerkung für eine unbekannte Id: err = %v — want ErrVormerkungNurFuerSchueler", err)
	}

	// Gegenprobe: Schüler und die Vormerkung ohne Person (Notiz für die Theke) gehen weiter.
	if _, err := repo.LegeAn(ctx, titelID, "", schueler); err != nil {
		t.Errorf("Vormerkung für einen Schüler: %v", err)
	}
	if _, err := repo.LegeAn(ctx, titelID, "für die Projektwoche", ""); err != nil {
		t.Errorf("Vormerkung ohne Person: %v", err)
	}
}
