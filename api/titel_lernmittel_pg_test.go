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
