//go:build raster

package inventur

// Nachstellung Rasterdurchgang 03.10.2026 über die Änderungen seit dem 02.10.2026 mittags
// (63784dcd). Build-Tag raster: läuft nur mit -tags raster. Rot heißt „bestätigt".

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Frage 3, zwei Türen: Maske, Bestellsuche und Bestelltür kennen die ISBN seit a7ae60e9 in
// beiden Längen. Der Listenimport führt eine Liste mit der dreizehnstelligen ISBN gegen einen
// Katalog, der das Buch mit der zehnstelligen trägt, und legt es ein zweites Mal an. Der Test
// bleibt rot, bis der Punkt in OFFEN.md 5.5 gebaut ist.
func TestRaster_Listenimport_KenntDieAndereLaengeDerISBN(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const zehn, dreizehn = "3551551677", "9783551551672"
	loesche := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = ANY($1)`, []string{zehn, dreizehn}); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_titel (titel, autor, isbn) VALUES ('Harry Potter und der Stein der Weisen', 'Rowling', $1)`,
		zehn); err != nil {
		t.Fatalf("Titel aus Littera anlegen: %v", err)
	}

	repo := &BookRepository{db: pool}
	if _, err := repo.UpsertBooksBatch(ctx, []Book{{ISBN: dreizehn, Title: "Harry Potter und der Stein der Weisen", Author: "Rowling", Stock: 1}}); err != nil {
		t.Fatalf("Listenimport: %v", err)
	}

	var titel int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM buecher_titel WHERE isbn = ANY($1)`, []string{zehn, dreizehn}).Scan(&titel); err != nil {
		t.Fatal(err)
	}
	if titel != 1 {
		t.Errorf("nach dem Listenimport stehen %d Titel für dasselbe Buch im Katalog, erwartet 1", titel)
	}
}
