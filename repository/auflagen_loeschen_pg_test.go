package repository

import (
	"context"
	"testing"
)

// Ein Werk mit weniger als zwei Titeln fällt — auch, wenn Titel gelöscht werden
// (Rasterdurchgang 25.09.2026, docs/OFFEN.md 4.18, Frage 1 und 12). Bis dahin stand die Regel
// nur im Lösen: Ein gelöschter Titel ließ den anderen allein an seinem Werk zurück.
func TestDeleteTitle_HaeltDieRegelDerAuflagen(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()

	a := seedAuflage(t, pool, "Mathe 7", 2019, true)
	b := seedAuflage(t, pool, "Mathe 7", 2023, true)
	c := seedAuflage(t, pool, "Mathe 7", 2025, true)
	for _, andere := range []string{b, c} {
		if _, err := FasseAuflagenZusammen(ctx, pool, a, andere); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewAuditRepository(pool)
	bearbeiter := seedBearbeiter(t, pool)
	werke := func() (n int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM werke`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Drei Auflagen, eine gelöscht: Das Buch bleibt mit zweien.
	if err := repo.DeleteTitle(ctx, c, bearbeiter); err != nil {
		t.Fatalf("DeleteTitle: %v", err)
	}
	if n := werke(); n != 1 {
		t.Errorf("nach dem Löschen der dritten Auflage: %d Werke — erwartet 1", n)
	}
	if auflagen, err := AuflagenDesTitels(ctx, pool, a); err != nil || len(auflagen) != 2 {
		t.Errorf("Auflagen nach dem Löschen: %+v, %v — erwartet zwei", auflagen, err)
	}

	// Noch eine gelöscht: Das Werk fällt, der letzte Titel ist wieder sein eigenes Buch.
	if err := repo.DeleteTitle(ctx, b, bearbeiter); err != nil {
		t.Fatalf("DeleteTitle: %v", err)
	}
	var werkID *string
	if err := pool.QueryRow(ctx, `SELECT werk_id::text FROM buecher_titel WHERE id = $1`, a).Scan(&werkID); err != nil {
		t.Fatal(err)
	}
	if werkID != nil {
		t.Errorf("nach dem Löschen der zweiten Auflage hängt die letzte noch an %s — erwartet ihr eigenes Buch", *werkID)
	}
	if n := werke(); n != 0 {
		t.Errorf("nach dem Löschen der zweiten Auflage: %d Werke — erwartet keins", n)
	}
}
