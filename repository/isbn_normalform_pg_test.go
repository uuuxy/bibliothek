package repository

import (
	"context"
	"errors"
	"testing"

	"bibliothek/pkg/isbnutil"

	"github.com/jackc/pgx/v5/pgconn"
)

// Eine ISBN hat EINE Schreibweise in der Datenbank (OFFEN.md 5.12, 22.09.2026).
//
// Der UNIQUE-Index auf buecher_titel.isbn fing nur die zeichengleiche Dublette:
// „978-3-16-148410-0" und „9783161484100" waren zwei Titel, also zwei Bestände, zwei
// Meldebestände und zwei Zeilen in der Nachbestellung. Die Maske prüfte seit dem
// 17.09.2026 beide Schreibweisen (inventur/dublettenkontrolle.go), die Importe nicht —
// und jeder weitere Schreiber müsste die Regel kennen.
//
// Seit Migration 133 bringt die Datenbank selbst jede geschriebene ISBN in die
// Normalform (ohne Bindestriche und Leerzeichen, Prüfzeichen X groß — nur, wenn das
// Ergebnis eine ISBN ist; anderes bleibt wie geschrieben), an JEDER Tür: Maske,
// Importe, Littera-Übernahme, Skripte. Damit greift der vorhandene UNIQUE-Index über alle
// Schreibweisen. Was schon anders gespeichert ist, bleibt bis zur Messung am Server
// unangetastet (OFFEN.md 5.5); die Dublettenkontrolle der Maske deckt die Zwischenzeit.
func TestISBN_HatEineSchreibweiseInDerDatenbank(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel LIKE 'Normalform-%'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	lies := func(t *testing.T, titel string) string {
		t.Helper()
		var isbn string
		if err := pool.QueryRow(ctx, `SELECT coalesce(isbn, '') FROM buecher_titel WHERE titel = $1`, titel).Scan(&isbn); err != nil {
			t.Fatal(err)
		}
		return isbn
	}

	faelle := []struct{ titel, roh, erwartet string }{
		{"Normalform-Bindestriche", "978-3-16-148410-0", "9783161484100"},
		{"Normalform-Leerzeichen", " 978 3 16 148410 0 ", "9783161484100"},
		{"Normalform-Pruefzeichen", "3-16-148410-x", "316148410X"},
		// Keine ISBN: bleibt, wie geschrieben — der Seed liest solche Kennungen per LIKE
		// zurück, und ein Altwert mit Beiwerk wird nicht still zu etwas anderem.
		{"Normalform-KeineISBN", "ISBN-0000000001", "ISBN-0000000001"},
		{"Normalform-Beiwerk", "3-12-345678-9 kart.", "3-12-345678-9 kart."},
	}
	for i, f := range faelle {
		if i == 1 {
			// Zweite Schreibweise derselben Nummer: jetzt greift der UNIQUE-Index.
			_, err := pool.Exec(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2)`, f.titel, f.roh)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
				t.Fatalf("zweite Schreibweise derselben ISBN: erwartet 23505 (unique), bekommen %v", err)
			}
			continue
		}
		if _, err := pool.Exec(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2)`, f.titel, f.roh); err != nil {
			t.Fatalf("%s anlegen: %v", f.titel, err)
		}
		if got := lies(t, f.titel); got != f.erwartet {
			t.Errorf("%s: gespeichert %q, erwartet %q", f.titel, got, f.erwartet)
		}
	}

	// Auch beim Ändern, und leer bleibt NULL (die Spalte ist nullbar; '' wäre eine
	// zweite Art von „keine ISBN").
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET isbn = '978-0-306-40615-7 ' WHERE titel = 'Normalform-Pruefzeichen'`); err != nil {
		t.Fatalf("ändern: %v", err)
	}
	if got := lies(t, "Normalform-Pruefzeichen"); got != "9780306406157" {
		t.Errorf("nach dem Ändern: %q, erwartet 9780306406157", got)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET isbn = ' - ' WHERE titel = 'Normalform-Pruefzeichen'`); err != nil {
		t.Fatalf("leeren: %v", err)
	}
	var istNull bool
	if err := pool.QueryRow(ctx, `SELECT isbn IS NULL FROM buecher_titel WHERE titel = 'Normalform-Pruefzeichen'`).Scan(&istNull); err != nil {
		t.Fatal(err)
	}
	if !istNull {
		t.Error("eine ISBN ohne Ziffern muss NULL werden, nicht ''")
	}
}

// Go und SQL rechnen dieselbe Normalform — sonst fände ein Lesepfad (Import-Zuordnung,
// Schnellanlage, Suche) still nichts, was die Datenbank gerade geschrieben hat.
func TestISBN_NormalformGoUndSQLSindGleich(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	for _, roh := range []string{
		"978-3-16-148410-0", " 978 3 16 148410 0 ", "3-16-148410-x", "9783161484100",
		"ISBN-0000000001", "3-12-345678-9 kart.", " - ", "", "978-3-16-ü", "12345",
	} {
		var sql *string
		if err := pool.QueryRow(ctx, `SELECT isbn_normalform($1)`, roh).Scan(&sql); err != nil {
			t.Fatalf("isbn_normalform(%q): %v", roh, err)
		}
		sqlWert := ""
		if sql != nil {
			sqlWert = *sql
		}
		if goWert := isbnutil.Normalform(roh); goWert != sqlWert {
			t.Errorf("Normalform(%q): Go %q, SQL %q — die beiden Seiten sind auseinander", roh, goWert, sqlWert)
		}
	}
}
