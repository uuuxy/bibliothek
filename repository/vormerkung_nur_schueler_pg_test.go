package repository

import (
	"context"
	"errors"
	"testing"
)

// Vormerken lässt sich nur für Schüler — und das muss die Tür sagen, nicht die Oberfläche.
//
// Die Warteschlange liest an vier Stellen die Sicht `schueler` (loan_return.go,
// vormerkung_nachruecken.go, loan_checkout.go, List in vormerkung.go). Die Oberfläche
// bietet zum Vormerken auch nur Schüler an (`/api/schueler?q=` ohne `art=alle`). Create
// nahm bis zum 21.09.2026 aber JEDE Leser-Id: Eine Vormerkung für einen Kollegen entstand,
// stand ohne Namen in der Liste, und beim Rückgabe-Vorgang ging die Warteschlange über sie
// hinweg — für immer „wartend", ohne dass es jemand merkt (OFFEN.md 5.19).
func TestVormerkungCreate_NurFuerSchueler(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "NurSchueler", 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	kollege := seedKollege(t, pool, "A-VM-K1", "Kim")
	schueler := seedSchueler(t, pool, "VM-S1", "Mia", "7a")
	repo := NewVormerkungRepository(pool)

	if _, err := repo.Create(ctx, titelID, "", kollege); !errors.Is(err, ErrVormerkungNurFuerSchueler) {
		t.Errorf("Vormerkung für einen Kollegen: err = %v — want ErrVormerkungNurFuerSchueler", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM vormerkungen WHERE schueler_id = $1`, kollege).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d Vormerkung(en) für den Kollegen geschrieben — die Warteschlange bedient sie nie", n)
	}

	// Eine Id, die es nicht gibt, ist dieselbe Antwort und kein roher FK-Fehler (500).
	if _, err := repo.Create(ctx, titelID, "", "00000000-0000-4000-8000-000000000001"); !errors.Is(err, ErrVormerkungNurFuerSchueler) {
		t.Errorf("Vormerkung für eine unbekannte Id: err = %v — want ErrVormerkungNurFuerSchueler", err)
	}

	// Gegenprobe: Schüler und die Vormerkung ohne Person (Notiz für die Theke) gehen weiter.
	if _, err := repo.Create(ctx, titelID, "", schueler); err != nil {
		t.Errorf("Vormerkung für einen Schüler: %v", err)
	}
	if _, err := repo.Create(ctx, titelID, "für die Projektwoche", ""); err != nil {
		t.Errorf("Vormerkung ohne Person: %v", err)
	}
}
