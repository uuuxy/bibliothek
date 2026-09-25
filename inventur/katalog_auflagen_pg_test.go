package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Medienkatalog fasst die Auflagen eines Buchs zu einer Kachel zusammen (docs/OFFEN.md
// 4.18, Stufe 6). Dafür trägt jeder Titel der Katalogliste sein Buch (werkId) und seinen Rang
// darin (werkRang, 1 = die neueste). Gezählt wird über ALLE Auflagen des Buchs: Die neueste hat
// hier noch kein Exemplar und steht deshalb nicht in der Liste, ihr Rang zählt trotzdem — die
// Reihenfolge der übrigen bleibt dieselbe, und der Einzel-Read sagt dasselbe wie die Liste.
func TestKatalog_TraegtBuchUndRangDerAuflage(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	var werk, bestellt, neu, alt, einzeln string
	if err := pool.QueryRow(ctx, `INSERT INTO werke DEFAULT VALUES RETURNING id`).Scan(&werk); err != nil {
		t.Fatalf("Werk: %v", err)
	}
	titel := func(ziel *string, name string, jahr int, mitWerk bool) {
		t.Helper()
		var w any
		if mitWerk {
			w = werk
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_titel (titel, erscheinungsjahr, ist_lernmittel, werk_id)
			VALUES ($1, $2, true, $3) RETURNING id`, name, jahr, w).Scan(ziel); err != nil {
			t.Fatalf("Titel %s: %v", name, err)
		}
	}
	titel(&bestellt, "Katalog-Auflagen 2025", 2025, true)
	titel(&neu, "Katalog-Auflagen 2023", 2023, true)
	titel(&alt, "Katalog-Auflagen 2019", 2019, true)
	titel(&einzeln, "Katalog-Auflagen ohne Buch", 2020, false)
	t.Cleanup(func() {
		// Exemplare hängen per ON DELETE CASCADE am Titel.
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`,
			[]string{bestellt, neu, alt, einzeln}); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM werke WHERE id = $1`, werk); err != nil {
			t.Errorf("Aufräumen Werk: %v", err)
		}
	})
	for i, id := range []string{neu, alt, einzeln} {
		if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`,
			id, "KATAUF-"+string(rune('1'+i))); err != nil {
			t.Fatalf("Exemplar: %v", err)
		}
	}

	liste, err := repo.ListBooks(ctx, "", nil, "", false)
	if err != nil {
		t.Fatalf("ListBooks: %v", err)
	}
	je := map[string]Book{}
	for _, b := range liste {
		je[b.ID] = b
	}
	if _, da := je[bestellt]; da {
		t.Fatal("die Auflage ohne Exemplar steht in der Katalogliste — der Aufbau des Tests stimmt nicht")
	}
	for _, fall := range []struct {
		id, name string
		werk     string
		rang     int
	}{
		{neu, "2023", werk, 2},
		{alt, "2019", werk, 3},
		{einzeln, "ohne Buch", "", 0},
	} {
		b, da := je[fall.id]
		if !da {
			t.Errorf("%s fehlt in der Katalogliste", fall.name)
			continue
		}
		if b.WerkID != fall.werk || b.WerkRang != fall.rang {
			t.Errorf("%s: werkId %q, werkRang %d — erwartet %q, %d", fall.name, b.WerkID, b.WerkRang, fall.werk, fall.rang)
		}
	}

	// Der Einzel-Read (Akte, Bearbeiten) liest dieselben Spalten: auch für die Auflage, die
	// die Liste nicht zeigt.
	einzel, err := repo.ListBooksByIDs(ctx, []string{bestellt})
	if err != nil {
		t.Fatalf("ListBooksByIDs: %v", err)
	}
	if len(einzel) != 1 || einzel[0].WerkID != werk || einzel[0].WerkRang != 1 {
		t.Errorf("Einzel-Read der neuesten Auflage: %+v — erwartet werkId %s, werkRang 1", einzel, werk)
	}
}
