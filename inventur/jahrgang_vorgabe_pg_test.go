package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Ein neuer Titel ohne Jahrgangsangabe bekommt die Vorgabe der Datenbank (5–10), nicht
// 0–0. Bis zum 10.09.2026 schrieb CreateBook den Go-Nullwert ausdrücklich in die Spalte —
// der DEFAULT aus Migration 008 griff nie. Der Scanner-Dialog und der Listenimport schicken
// keinen Jahrgang; jede offene Ausleihe eines solchen Titels stand dann im Mahnwesen-Modus
// „Jahrgang" (klasse_num > jahrgang_bis) für jeden Schüler mit Ziffernklasse, eine
// Klassen-Inventur sah den Titel nie, und der Jahrgangsfilter des Portals blendete ihn aus
// (Bestands-Durchgang, neue Bugklasse „DEFAULT-Umgehung durch Nullwert"). Die Zwillinge
// Littera und Bestands-CSV schrieben längst COALESCE(NULLIF($n,0),5/10).
func TestNeuerTitel_OhneJahrgang_BekommtDieVorgabe(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = '978-9-99-200000-1'`); err != nil {
		t.Fatal(err)
	}
	id, err := NewBookRepository(pool).CreateBook(ctx, Book{ISBN: "978-9-99-200000-1", Title: "Scanner-Neuanlage"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	var von, bis int
	if err := pool.QueryRow(ctx, `SELECT jahrgang_von, jahrgang_bis FROM buecher_titel WHERE id = $1`, id).Scan(&von, &bis); err != nil {
		t.Fatal(err)
	}
	if von != 5 || bis != 10 {
		t.Errorf("Jahrgang %d–%d, want 5–10 (Vorgabe der Datenbank)", von, bis)
	}
}
