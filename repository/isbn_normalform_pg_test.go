package repository

import (
	"context"
	"errors"
	"testing"

	"bibliothek/pkg/isbnutil"

	"github.com/jackc/pgx/v5/pgconn"
)

// Eine ISBN hat eine Schreibweise in der Datenbank, und eine Länge.
//
// Der UNIQUE-Index auf buecher_titel.isbn fing nur die zeichengleiche Dublette:
// „978-3-16-148410-0" und „9783161484100" waren zwei Titel, also zwei Bestände, zwei
// Meldebestände und zwei Zeilen in der Nachbestellung. Dasselbe galt für die zehnstellige
// ISBN vom Titelblatt neben der dreizehnstelligen vom Strichcode.
//
// Die Datenbank bringt jede geschriebene ISBN in die Normalform (Migration 133 und 157), an
// jeder Tür: Maske, Importe, Littera-Übernahme, Skripte. Ohne Bindestriche und Leerzeichen,
// Prüfzeichen X groß, und eine zehnstellige mit richtigem Prüfzeichen wird dreizehnstellig.
// Damit greift der vorhandene UNIQUE-Index über alle Schreibweisen und beide Längen.
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

	// erwartet leer heißt: Die Nummer trägt schon ein Titel, der UNIQUE-Index lehnt ab.
	faelle := []struct{ titel, roh, erwartet string }{
		{"Normalform-Bindestriche", "978-3-16-148410-0", "9783161484100"},
		{"Normalform-Leerzeichen", " 978 3 16 148410 0 ", ""},
		// Dasselbe Buch mit der zehnstelligen ISBN vom Titelblatt.
		{"Normalform-AndereLaenge", "3-16-148410-x", ""},
		{"Normalform-Zehnstellig", "0-306-40615-2", "9780306406157"},
		// Falsches Prüfzeichen: bleibt zehnstellig. Gerechnet führte die Nummer auf
		// 9783499500251, am Testserver die ISBN eines anderen Buchs.
		{"Normalform-FalschesPruefzeichen", "3499500252", "3499500252"},
		{"Normalform-NebenDerFalschen", "9783499500251", "9783499500251"},
		// Keine ISBN: bleibt, wie geschrieben — der Seed liest solche Kennungen per LIKE
		// zurück, und ein Altwert mit Beiwerk wird nicht still zu etwas anderem.
		{"Normalform-KeineISBN", "ISBN-0000000001", "ISBN-0000000001"},
		{"Normalform-Beiwerk", "3-12-345678-9 kart.", "3-12-345678-9 kart."},
	}
	for _, f := range faelle {
		_, err := pool.Exec(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2)`, f.titel, f.roh)
		if f.erwartet == "" {
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
				t.Errorf("%s (%q): erwartet 23505 (unique), bekommen %v", f.titel, f.roh, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s anlegen: %v", f.titel, err)
		}
		if got := lies(t, f.titel); got != f.erwartet {
			t.Errorf("%s: gespeichert %q, erwartet %q", f.titel, got, f.erwartet)
		}
	}

	// Auch beim Ändern, und leer bleibt NULL (die Spalte ist nullbar; '' wäre eine
	// zweite Art von „keine ISBN").
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET isbn = '0-8044-2957-x ' WHERE titel = 'Normalform-Zehnstellig'`); err != nil {
		t.Fatalf("ändern: %v", err)
	}
	if got := lies(t, "Normalform-Zehnstellig"); got != "9780804429573" {
		t.Errorf("nach dem Ändern: %q, erwartet 9780804429573", got)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET isbn = ' - ' WHERE titel = 'Normalform-Zehnstellig'`); err != nil {
		t.Fatalf("leeren: %v", err)
	}
	var istNull bool
	if err := pool.QueryRow(ctx, `SELECT isbn IS NULL FROM buecher_titel WHERE titel = 'Normalform-Zehnstellig'`).Scan(&istNull); err != nil {
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
	faelle := []string{
		"978-3-16-148410-0", " 978 3 16 148410 0 ", "3-16-148410-x", "9783161484100",
		"0-306-40615-2", "3499500252", "9791234567896", "0000000000",
		"ISBN-0000000001", "3-12-345678-9 kart.", " - ", "", "978-3-16-ü", "12345",
	}
	// Je Kern alle elf Prüfzeichen: Genau eines ist richtig und wird dreizehnstellig, die
	// zehn übrigen bleiben. So laufen beide Rechnungen über jede Ziffer an jeder Stelle.
	umgerechnet := 0
	for _, kern := range []string{"316148410", "030640615", "080442957", "349950025", "355155167", "000000000", "999999999", "123456789", "987654321", "501928374"} {
		for _, zeichen := range "0123456789X" {
			zehn := kern + string(zeichen)
			if len(isbnutil.Normalform(zehn)) == 13 {
				umgerechnet++
			}
			faelle = append(faelle, zehn)
		}
	}
	if umgerechnet != 10 {
		t.Errorf("%d von 10 Kernen wurden dreizehnstellig — je Kern ist genau ein Prüfzeichen richtig", umgerechnet)
	}
	for _, roh := range faelle {
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
