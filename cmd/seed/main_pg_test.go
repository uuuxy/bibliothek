package main

// Das Werkzeug läuft gegen eine echte Datenbank. Geprüft wird derselbe Lauf, nur klein, mit
// einem Umfang, der kein Vielfaches der Portion ist, und zweimal hintereinander.

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/internal/pgtest"
)

func raeumeTestdatenAuf(t *testing.T, pool *pgxpool.Pool) {
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

// 25 Exemplare in Portionen zu 10: Liefe die Schleife drei volle Portionen, entstünden 30;
// beim vollen Umfang (80.000 / 10.000) fiele das nie auf.
func TestFuelle_LegtGenauDenUmfangAn(t *testing.T) {
	pool := pgtest.Pool(t)
	raeumeTestdatenAuf(t, pool)
	t.Cleanup(func() { raeumeTestdatenAuf(t, pool) })

	var ausgabe bytes.Buffer
	adminID, err := fuelle(context.Background(), pool, Umfang{Schueler: 7, Titel: 4, Exemplare: 25, Portion: 10}, &ausgabe)
	if err != nil {
		t.Fatalf("seed: %v\n%s", err, ausgabe.String())
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
	// Jedes Exemplar hängt an einem der vier Titel des Laufs.
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

// Der zweite Lauf ist derselbe Lauf: nichts doppelt, und das JWT gehört zu dem Konto, das es
// gibt.
func TestFuelle_ZweiterLaufIstDerselbeAdmin(t *testing.T) {
	pool := pgtest.Pool(t)
	raeumeTestdatenAuf(t, pool)
	t.Cleanup(func() { raeumeTestdatenAuf(t, pool) })
	ctx := context.Background()
	klein := Umfang{Schueler: 3, Titel: 2, Exemplare: 5, Portion: 5}

	var ausgabe bytes.Buffer
	erster, err := fuelle(ctx, pool, klein, &ausgabe)
	if err != nil {
		t.Fatalf("erster Lauf: %v", err)
	}
	zweiter, err := fuelle(ctx, pool, klein, &ausgabe)
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
