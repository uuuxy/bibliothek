//go:build raster

package inventur

// Nachstellung Rasterdurchgang 03.10.2026 über die Änderungen seit dem 02.10.2026 mittags
// (63784dcd). Build-Tag raster: läuft nur mit -tags raster. Rot heißt „bestätigt".

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/internal/pgtest"
)

// Frage 14, Datenlage: Ein Titel ohne ISBN (isbn NULL, wie ihn die Importe anlegen) wird in
// der Maske „Buch bearbeiten" geöffnet und bekommt eine Signatur. Die Maske schickt die leere
// ISBN mit, und die Tür lehnt ab: Der Titel lässt sich nicht ändern. Der Test bleibt rot bis
// zur Entscheidung in OFFEN.md 5.47.
func TestRaster_TitelOhneISBN_LaesstSichAendern(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const titel = "Rasterprobe ohne ISBN 0310"
	loesche := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, autor, isbn) VALUES ($1, 'Autorin', NULL) RETURNING id::text`,
		titel).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	handler := &APIHandler{repo: &BookRepository{db: pool}}
	daten, err := json.Marshal(map[string]any{"isbn": "", "title": titel, "author": "Autorin", "signatur": "Ras 1"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/books/"+id, bytes.NewReader(daten))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	handler.BearbeiteBuchAktualisieren(rec, req)

	var signatur string
	if err := pool.QueryRow(ctx, `SELECT COALESCE(signatur, '') FROM buecher_titel WHERE id = $1`, id).Scan(&signatur); err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	if rec.Code != http.StatusOK || signatur != "Ras 1" {
		t.Errorf("Titel ohne ISBN ändern: Status %d, Antwort %s, Signatur danach %q", rec.Code, rec.Body.String(), signatur)
	}
}

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
