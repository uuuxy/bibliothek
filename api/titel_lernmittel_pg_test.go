package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/inventur"
)

// Ein über die DNB-Bestellsuche neu angelegter Titel entsteht OHNE Lernmittel-Kennzeichen
// und meldet das (ist_lernmittel=false in der Antwort) — das Staging-Fenster fragt nach
// und schreibt die Antwort über PUT /api/buecher/titel/{id}/lernmittel. Vorher gab es
// diese Rückfrage nicht: Jedes neue Schulbuch war für immer ein Bücherei-Titel.
func TestNeuerDnbTitelWirdAufRueckfrageZumLernmittel(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	resp, err := srv.upsertTitelAusMetadaten(ctx, "9783060000001", &inventur.MetadatenErgebnis{Titel: "Mathematik 7 Hessen"})
	if err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if resp.IstLernmittel {
		t.Error("ein frisch angelegter DNB-Titel darf nicht ungefragt Lernmittel sein")
	}

	req := httptest.NewRequest(http.MethodPut, "/api/buecher/titel/"+resp.TitelID+"/lernmittel",
		strings.NewReader(`{"ist_lernmittel":true}`))
	req.SetPathValue("id", resp.TitelID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.UpdateTitelLernmittelHandler()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}

	var istLernmittel bool
	if err := pool.QueryRow(ctx, `SELECT ist_lernmittel FROM buecher_titel WHERE id = $1`, resp.TitelID).Scan(&istLernmittel); err != nil {
		t.Fatal(err)
	}
	if !istLernmittel {
		t.Error("Kennzeichen nicht gespeichert")
	}

	// Und die lokale Suche liefert es jetzt mit — daraus wird der Topf-Vorschlag im Warenkorb.
	lokal, err := srv.findeLokalenTitel(ctx, "9783060000001")
	if err != nil || lokal == nil {
		t.Fatalf("lokaler Titel: %v %v", lokal, err)
	}
	if !lokal.IstLernmittel {
		t.Error("/aus-isbn meldet den vorhandenen Titel ohne Kennzeichen")
	}
}

func TestLernmittelKennzeichenUnbekannterTitelIst404(t *testing.T) {
	pool := pgTestPool(t)
	srv := &Server{DB: &db.Database{Pool: pool}}
	id := "00000000-0000-0000-0000-000000000000"
	req := httptest.NewRequest(http.MethodPut, "/api/buecher/titel/"+id+"/lernmittel", strings.NewReader(`{"ist_lernmittel":true}`))
	req.SetPathValue("id", id)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.UpdateTitelLernmittelHandler()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Status %d, want 404", rec.Code)
	}
}

// Fällt das Lernmittel-Kennzeichen, fällt der Mehrjahresband mit (Migration 134,
// Rasterdurchgang 23.09.2026, K2) — wie im Buchformular. Das Fenster der Bestellsuche
// schreibt über diese Tür auch das Kennzeichen eines VORHANDENEN Titels. Setzte sie allein
// ist_lernmittel, lehnte chk_mehrjahresband_spanne ab (am alten Code: Status 500, „violates
// check constraint"), und das Fenster meldete nur „konnte nicht gespeichert werden".
func TestLernmittelKennzeichenNimmtDenMehrjahresbandMit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	const isbnBand, isbnNormal = "9780000134031", "9780000134048"
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn IN ($1, $2)`, isbnBand, isbnNormal); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	var band, normal string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, isbn, ist_lernmittel, jahrgang_von, jahrgang_bis, mehrjahresband)
		VALUES ('Band Erdkunde 7-9', $1, true, 7, 9, true) RETURNING id`, isbnBand).Scan(&band); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, isbn, ist_lernmittel, jahrgang_von, jahrgang_bis)
		VALUES ('Normal Erdkunde 7', $1, true, 7, 9) RETURNING id`, isbnNormal).Scan(&normal); err != nil {
		t.Fatal(err)
	}
	setze := func(id, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/api/buecher/titel/"+id+"/lernmittel", strings.NewReader(body))
		req.SetPathValue("id", id)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.UpdateTitelLernmittelHandler()(rec, req)
		return rec
	}
	lies := func(id string) (lernmittel, schalter bool) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT ist_lernmittel, mehrjahresband FROM buecher_titel WHERE id = $1`, id).Scan(&lernmittel, &schalter); err != nil {
			t.Fatal(err)
		}
		return lernmittel, schalter
	}

	// Gegenprobe: ohne Schalter fällt nur das Kennzeichen.
	if rec := setze(normal, `{"ist_lernmittel":false}`); rec.Code != http.StatusOK {
		t.Fatalf("gewöhnlicher Titel: Status %d: %s", rec.Code, rec.Body.String())
	}

	// Ein Häkchen, das bleibt, lässt den Schalter stehen.
	if rec := setze(band, `{"ist_lernmittel":true}`); rec.Code != http.StatusOK {
		t.Fatalf("Mehrjahresband, Kennzeichen bleibt: Status %d: %s", rec.Code, rec.Body.String())
	}
	if lernmittel, schalter := lies(band); !lernmittel || !schalter {
		t.Errorf("Kennzeichen bleibt an: ist_lernmittel=%v, mehrjahresband=%v — erwartet beide an", lernmittel, schalter)
	}

	if rec := setze(band, `{"ist_lernmittel":false}`); rec.Code != http.StatusOK {
		t.Fatalf("Mehrjahresband, Kennzeichen aus: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if lernmittel, schalter := lies(band); lernmittel || schalter {
		t.Errorf("nach „kein Lernmittel“: ist_lernmittel=%v, mehrjahresband=%v — erwartet beide aus", lernmittel, schalter)
	}
}
