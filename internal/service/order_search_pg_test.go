package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// TestSearchLocalOrders_LiefertSignatur belegt, dass die lokale Bestellsuche die
// Regalsignatur eines bereits katalogisierten Titels mitliefert — Voraussetzung
// dafür, dass der Bestellkorb sie anzeigen und "die vorhandene Systematik
// übernehmen" statt sie zu verschweigen. Gated auf TEST_DATABASE_URL, siehe
// [[pg-integration-test-workflow]].
//
// Seit 01.09.2026 über internal/pgtest: Der Test hielt sich vorher einen EIGENEN
// Advisory-Lock auf demselben Schlüssel (0x42DB0001) samt eigenem Schema-Reset —
// sobald ein zweiter Test im selben Binary den Lock über pgtest hielt (der ihn
// bis Prozessende hält), wartete dieser Test für immer auf sein eigenes Binary.
// Genau so hat er ab dem ersten weiteren PG-Test in diesem Paket die komplette
// Suite in den 10-Minuten-Timeout gezogen; der eigene DROP SCHEMA mitten im
// Binary hätte zudem jedem nachfolgenden Test die Tabellen weggezogen.
func TestSearchLocalOrders_LiefertSignatur(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, autor, isbn, signatur) VALUES ($1, $2, $3, $4) RETURNING id`,
		"Effi Briest", "Fontane, Theodor", "9783150001", "Pg").Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})

	results := searchLocalOrders(ctx, pool, "Effi Briest")
	if len(results) != 1 {
		t.Fatalf("Treffer = %d, want 1", len(results))
	}
	if results[0].Signatur != "Pg" {
		t.Errorf("Signatur = %q, want %q", results[0].Signatur, "Pg")
	}
	if results[0].Source != "local" {
		t.Errorf("Source = %q, want %q", results[0].Source, "local")
	}
}

// Die Bestellsuche im eigenen Katalog vergleicht eine getippte ISBN als Teilstring. Die
// zehnstellige vom Titelblatt und die dreizehnstellige, unter der die Datenbank den Titel
// führt (Migration 157), enden auf verschiedene Prüfzeichen: Ohne den Vergleich mit der
// Normalform des Suchtexts hieße ein vorhandenes Buch in der Bestellsuche „nicht gefunden".
func TestSearchLocalOrders_FindetDenTitelUeberDieZehnstelligeISBN(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const zehn, dreizehn = "0306406152", "9780306406157"
	raeume := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = ANY($1)`, []string{zehn, dreizehn}); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ('Advanced Organic Chemistry', $1) RETURNING id::text`,
		zehn).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	for _, suchtext := range []string{zehn, "0-306-40615-2", dreizehn, "978-0-306-40615-7"} {
		treffer := searchLocalOrders(ctx, pool, suchtext)
		if len(treffer) != 1 || treffer[0].ID != id || treffer[0].ISBN != dreizehn {
			t.Errorf("Suche %q: %+v, erwartet den Titel %s unter %s", suchtext, treffer, id, dreizehn)
		}
	}
	if treffer := searchLocalOrders(ctx, pool, "0306406153"); len(treffer) != 0 {
		t.Errorf("falsches Prüfzeichen: %d Treffer, erwartet keinen", len(treffer))
	}
}

// Die Bestellsuche im eigenen Katalog vergleicht den Suchtext neben dem Volltext als
// Teilstring. Die Datenbank speichert Titeltexte mit einem Leerzeichen zwischen den Wörtern
// (Migration 160), der Suchtext geht in dieselbe Form: Sonst hieße ein vorhandenes Buch in
// der Bestellsuche „nicht gefunden" und würde ein zweites Mal aufgenommen. Die Suchtexte
// enden mitten im Wort, damit der Volltext nicht aushilft.
func TestSearchLocalOrders_LeerraumInFolgeTrenntSuchtextUndTitelNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	geschuetzt := string(rune(0x00A0))
	raeume := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel LIKE 'Leerraumprobe%'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor) VALUES ($1, $2) RETURNING id::text`,
		"Leerraumprobe La  Peste", "Camus,"+geschuetzt+"Albert").Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	for _, suchtext := range []string{
		"Leerraumprobe La Pes",
		"Leerraumprobe La  Pes",
		"leerraumprobe la" + geschuetzt + "pes",
		"Camus,  Alb",
	} {
		treffer := searchLocalOrders(ctx, pool, suchtext)
		if len(treffer) != 1 || treffer[0].ID != id || treffer[0].Titel != "Leerraumprobe La Peste" {
			t.Errorf("Suche %+q: %+v, erwartet den Titel %s", suchtext, treffer, id)
		}
	}
	if treffer := searchLocalOrders(ctx, pool, "Leerraumprobe LaPes"); len(treffer) != 0 {
		t.Errorf("ohne Leerzeichen: %d Treffer, erwartet keinen", len(treffer))
	}
}
