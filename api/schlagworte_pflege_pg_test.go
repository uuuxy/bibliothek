package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Die Türen der Schlagwort-Pflege (Migration 143, OFFEN.md 4.20 Stufe 1): Die Regeln misst
// repository/schlagworte_pflege_pg_test.go; hier geht es darum, was an der Tür ankommt —
// Statuscode und Satz für jeden fachlichen Fehler, UUID-Prüfung vor der Datenbank.
func TestSchlagwortPflege_Tueren(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: &db.Database{Pool: pool}}

	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Tür-Probe') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.SetzeSchlagworte(ctx, pool, titelID, []string{"Tierfantasy", "Fantasy"}); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, w := range []string{"Tierfantasy", "Fantasy"} {
		var id string
		if err := pool.QueryRow(ctx, `SELECT id FROM schlagworte WHERE wort = $1`, w).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids[w] = id
	}

	ruf := func(h http.HandlerFunc, methode, pfad, id, koerper string) *httptest.ResponseRecorder {
		if pfad == "" {
			pfad = "/api/schlagworte/probe" // die Kennung kommt über SetPathValue
		}
		req := httptest.NewRequest(methode, pfad, strings.NewReader(koerper))
		req.Header.Set("Content-Type", "application/json")
		if id != "" {
			req.SetPathValue("id", id)
		}
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec
	}

	if rec := ruf(srv.GetSchlagwortPflegeHandler(), http.MethodGet, "/api/schlagworte/pflege", "", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"gesamt":2`) {
		t.Errorf("Liste: %d %s", rec.Code, rec.Body.String())
	}
	if rec := ruf(srv.PutSchlagwortWortHandler(), http.MethodPut, "", ids["Tierfantasy"], `{"wort":"fantasy"}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "Zusammenführen") {
		t.Errorf("umbenennen auf vorhandenes Wort: %d %s, want 409 mit Hinweis", rec.Code, rec.Body.String())
	}
	if rec := ruf(srv.PutSchlagwortWortHandler(), http.MethodPut, "", "keine-uuid", `{"wort":"X"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ungültige Kennung: %d, want 400", rec.Code)
	}
	if rec := ruf(srv.PutSchlagwortFilterHandler(), http.MethodPut, "", ids["Fantasy"], `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Filter ohne Feld: %d, want 400", rec.Code)
	}
	if rec := ruf(srv.PostSchlagwortZusammenfuehrenHandler(), http.MethodPost, "", ids["Tierfantasy"], `{"ziel_id":"keine-uuid"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Ziel keine UUID: %d, want 400", rec.Code)
	}

	rec := ruf(srv.PostSchlagwortZusammenfuehrenHandler(), http.MethodPost, "", ids["Tierfantasy"], `{"ziel_id":"`+ids["Fantasy"]+`"}`)
	var aenderung SchlagwortAenderung
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &aenderung) != nil || aenderung.Titel != 1 {
		t.Fatalf("zusammenführen: %d %s, want 200 mit titel=1", rec.Code, rec.Body.String())
	}
	if rec := ruf(srv.PutSchlagwortFilterHandler(), http.MethodPut, "", ids["Tierfantasy"], `{"ist_filter":true}`); rec.Code != http.StatusConflict {
		t.Errorf("Verweis als Filter: %d %s, want 409", rec.Code, rec.Body.String())
	}

	rec = ruf(srv.DeleteSchlagwortHandler(), http.MethodDelete, "", ids["Fantasy"], "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &aenderung) != nil || aenderung.Titel != 1 || aenderung.Verweise != 1 {
		t.Errorf("löschen: %d %s, want 200 mit titel=1, verweise=1", rec.Code, rec.Body.String())
	}
	if rec := ruf(srv.DeleteSchlagwortHandler(), http.MethodDelete, "", ids["Fantasy"], ""); rec.Code != http.StatusNotFound {
		t.Errorf("zweites Löschen: %d, want 404", rec.Code)
	}
}
