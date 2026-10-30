package repository_test

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Ein Cover von außen zeigt die Liste der Renner nur mit ISBN: Der Bild-Proxy braucht sie als
// Schlüssel, und coverSrc.js gibt ohne sie den Platzhalter zurück.
func TestLadeRennerNenntISBNUndCover(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	var titelID, exemplarID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, isbn, cover_url) VALUES ('Renner-Titel', '9783000000019', 'https://bilder.example/a.jpg') RETURNING id::text`).
		Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-RENNER-1') RETURNING id::text`, titelID).
		Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	leserID := testdaten.LegeSchuelerAn(t, pool, "A-RENNER-1", "Renner", "07A")
	if _, err := pool.Exec(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, rueckgabe_am) VALUES ($1, $2, now(), now())`,
		exemplarID, leserID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	renner, err := repository.LadeRenner(ctx, pool, "", "", 5)
	if err != nil {
		t.Fatalf("LadeRenner: %v", err)
	}
	if len(renner) != 1 {
		t.Fatalf("%d Renner, erwartet 1", len(renner))
	}
	if renner[0].ISBN != "9783000000019" || renner[0].CoverURL != "https://bilder.example/a.jpg" {
		t.Errorf("ISBN %q, Cover %q", renner[0].ISBN, renner[0].CoverURL)
	}
}
