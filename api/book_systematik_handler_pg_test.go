package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// TestUpdateTitelSignaturHandler_SpeichertNeueSignatur belegt den Bearbeitungspfad
// im Bestellkorb: Ein Vorschlag ("BIB Jugendbuch") lässt sich vor dem Bestellen
// korrigieren, ohne über das große Buchformular zu gehen.
func TestUpdateTitelSignaturHandler_SpeichertNeueSignatur(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	titelID := titelMitSignatur(t, pool, "Die Tribute von Panem", "BIB Jugendbuch", 0)

	req := httptest.NewRequest(http.MethodPut, "/api/buecher/titel/"+titelID+"/signatur",
		strings.NewReader(`{"signatur":"JF Panem"}`))
	req.SetPathValue("id", titelID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.UpdateTitelSignaturHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var signatur string
	if err := pool.QueryRow(ctx, `SELECT signatur FROM buecher_titel WHERE id = $1`, titelID).Scan(&signatur); err != nil {
		t.Fatalf("Signatur lesen: %v", err)
	}
	if signatur != "JF Panem" {
		t.Errorf("Signatur in DB = %q, want %q", signatur, "JF Panem")
	}
}

// TestUpdateTitelSignaturHandler_UnbekannteIDGibt404 belegt, dass eine nicht
// existierende Titel-ID sauber als 404 gemeldet wird statt als 500.
func TestUpdateTitelSignaturHandler_UnbekannteIDGibt404(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	unbekannteID := "00000000-0000-0000-0000-000000000000"
	req := httptest.NewRequest(http.MethodPut, "/api/buecher/titel/"+unbekannteID+"/signatur",
		strings.NewReader(`{"signatur":"JF Panem"}`))
	req.SetPathValue("id", unbekannteID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.UpdateTitelSignaturHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want 404, body: %s", rec.Code, rec.Body.String())
	}
}
