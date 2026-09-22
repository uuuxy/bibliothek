package main

// Das Seed-Werkzeug läuft gegen eine echte Datenbank und hatte keinen Test (OFFEN.md 5.10).
// Geprüft wird derselbe Lauf, nur klein — mit einem Umfang, der kein Vielfaches der
// Portionsgröße ist, und zweimal hintereinander.

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/internal/pgtest"
)

func raeumeSeedAuf(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	for _, sql := range []string{
		`DELETE FROM buecher_exemplare WHERE barcode_id ~ '^B[0-9]{7}$'`,
		`DELETE FROM buecher_titel WHERE isbn LIKE 'ISBN-%'`,
		`DELETE FROM leser WHERE barcode_id ~ '^S[0-9]{6}$'`,
		`DELETE FROM benutzer WHERE email = '` + adminEmail + `'`,
		`DELETE FROM leser WHERE barcode_id = '` + adminBarcode + `'`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Errorf("aufräumen (%s): %v", sql, err)
		}
	}
}

func zaehle(t *testing.T, pool *pgxpool.Pool, sql string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

// 25 Exemplare in Portionen zu 10: Bis zum 22.09.2026 lief die Schleife drei volle
// Portionen und legte 30 an — beim vollen Umfang (80.000 / 10.000) fiel das nie auf.
func TestSeed_LegtGenauDenUmfangAn(t *testing.T) {
	pool := pgtest.Pool(t)
	raeumeSeedAuf(t, pool)
	t.Cleanup(func() { raeumeSeedAuf(t, pool) })

	var out bytes.Buffer
	adminID, err := seed(context.Background(), pool, Umfang{Schueler: 7, Titel: 4, Exemplare: 25, Chunk: 10}, &out)
	if err != nil {
		t.Fatalf("seed: %v\n%s", err, out.String())
	}

	if n := zaehle(t, pool, `SELECT count(*) FROM schueler WHERE barcode_id ~ '^S[0-9]{6}$' AND deleted_at IS NULL`); n != 7 {
		t.Errorf("%d Schüler, erwartet 7", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_titel WHERE isbn LIKE 'ISBN-%'`); n != 4 {
		t.Errorf("%d Titel, erwartet 4", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_exemplare WHERE barcode_id ~ '^B[0-9]{7}$'`); n != 25 {
		t.Errorf("%d Exemplare, erwartet 25", n)
	}
	// Jedes Exemplar hängt an einem der vier Seed-Titel.
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.barcode_id ~ '^B[0-9]{7}$' AND t.isbn NOT LIKE 'ISBN-%'`); n != 0 {
		t.Errorf("%d Exemplare hängen an fremden Titeln", n)
	}
	// Der Test-Admin: ein Konto, dessen Leserzeile den Scanner-Ausweis trägt.
	var art string
	if err := pool.QueryRow(context.Background(), `
		SELECT l.art FROM benutzer b JOIN leser l ON l.id = b.leser_id
		WHERE b.id = $1 AND l.barcode_id = $2`, adminID, adminBarcode).Scan(&art); err != nil {
		t.Fatalf("Test-Admin %s mit Ausweis %s: %v", adminID, adminBarcode, err)
	}
	if art != "lehrkraft" {
		t.Errorf("Art der Admin-Leserzeile: %q", art)
	}
}

// Der zweite Lauf ist derselbe Lauf: nichts doppelt, und das JWT gehört zu dem Konto,
// das es gibt. Bis zum 22.09.2026 bekam der zweite Lauf eine frische UUID, deren INSERT
// am vorhandenen Konto abprallte (ON CONFLICT DO NOTHING) — die gedruckte Kennung
// gehörte zu niemandem, und der Ausweis fand keine Leserzeile.
func TestSeed_ZweiterLaufIstDerselbeAdmin(t *testing.T) {
	pool := pgtest.Pool(t)
	raeumeSeedAuf(t, pool)
	t.Cleanup(func() { raeumeSeedAuf(t, pool) })
	ctx := context.Background()
	klein := Umfang{Schueler: 3, Titel: 2, Exemplare: 5, Chunk: 5}

	var out bytes.Buffer
	erster, err := seed(ctx, pool, klein, &out)
	if err != nil {
		t.Fatalf("erster Lauf: %v", err)
	}
	zweiter, err := seed(ctx, pool, klein, &out)
	if err != nil {
		t.Fatalf("zweiter Lauf: %v", err)
	}
	if zweiter != erster {
		t.Errorf("zweiter Lauf nennt Admin %s, der erste %s", zweiter, erster)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM benutzer WHERE email = '`+adminEmail+`'`); n != 1 {
		t.Errorf("%d Admin-Konten, erwartet 1", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_exemplare WHERE barcode_id ~ '^B[0-9]{7}$'`); n != 5 {
		t.Errorf("%d Exemplare nach zwei Läufen, erwartet 5", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_titel WHERE isbn LIKE 'ISBN-%'`); n != 2 {
		t.Errorf("%d Titel nach zwei Läufen, erwartet 2", n)
	}
}
