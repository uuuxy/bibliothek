package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// TestFindeImKatalog_LiefertSignatur belegt, dass die lokale Bestellsuche die Regalsignatur
// eines katalogisierten Titels mitliefert: Nur dann kann der Bestellkorb sie anzeigen und die
// vorhandene Systematik übernehmen. Läuft nur mit TEST_DATABASE_URL.
//
// Die Datenbank kommt über internal/pgtest: Ein eigener Advisory-Lock auf demselben Schlüssel
// wartete auf den, den pgtest im selben Testprogramm bis zum Ende hält, und ein eigenes DROP
// SCHEMA nähme den übrigen Tests die Tabellen.
func TestFindeImKatalog_LiefertSignatur(t *testing.T) {
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

	results := findeImKatalog(ctx, pool, "Effi Briest")
	if len(results) != 1 {
		t.Fatalf("Treffer = %d, want 1", len(results))
	}
	if results[0].Signatur != "Pg" {
		t.Errorf("Signatur = %q, want %q", results[0].Signatur, "Pg")
	}
	if results[0].Quelle != "local" {
		t.Errorf("Source = %q, want %q", results[0].Quelle, "local")
	}
}

// Die Bestellsuche im eigenen Katalog vergleicht eine getippte ISBN als Teilstring. Die
// zehnstellige vom Titelblatt und die dreizehnstellige, unter der die Datenbank den Titel
// führt (Migration 157), enden auf verschiedene Prüfzeichen: Ohne den Vergleich mit der
// Normalform des Suchtexts hieße ein vorhandenes Buch in der Bestellsuche „nicht gefunden".
func TestFindeImKatalog_FindetDenTitelUeberDieZehnstelligeISBN(t *testing.T) {
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
		treffer := findeImKatalog(ctx, pool, suchtext)
		if len(treffer) != 1 || treffer[0].ID != id || treffer[0].ISBN != dreizehn {
			t.Errorf("Suche %q: %+v, erwartet den Titel %s unter %s", suchtext, treffer, id, dreizehn)
		}
	}
	if treffer := findeImKatalog(ctx, pool, "0306406153"); len(treffer) != 0 {
		t.Errorf("falsches Prüfzeichen: %d Treffer, erwartet keinen", len(treffer))
	}
}

// Die Bestellsuche im eigenen Katalog vergleicht den Suchtext neben dem Volltext als
// Teilstring. Die Datenbank speichert Titeltexte mit einem Leerzeichen zwischen den Wörtern
// (Migration 160), der Suchtext geht in dieselbe Form: Sonst hieße ein vorhandenes Buch in
// der Bestellsuche „nicht gefunden" und würde ein zweites Mal aufgenommen. Die Suchtexte
// enden mitten im Wort, damit der Volltext nicht aushilft.
func TestFindeImKatalog_LeerraumInFolgeTrenntSuchtextUndTitelNicht(t *testing.T) {
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
		treffer := findeImKatalog(ctx, pool, suchtext)
		if len(treffer) != 1 || treffer[0].ID != id || treffer[0].Titel != "Leerraumprobe La Peste" {
			t.Errorf("Suche %+q: %+v, erwartet den Titel %s", suchtext, treffer, id)
		}
	}
	if treffer := findeImKatalog(ctx, pool, "Leerraumprobe LaPes"); len(treffer) != 0 {
		t.Errorf("ohne Leerzeichen: %d Treffer, erwartet keinen", len(treffer))
	}
}
