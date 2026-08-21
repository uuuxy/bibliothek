package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"
)

// Durch die Vordertür: Eine Excel-Datei ohne Schüler-ID geht als Multipart-Upload an den
// Vorschau-Handler und kommt als Namensmodus-Vorschau zurück. Beweist die Verdrahtung
// Upload → Formaterkennung → Parser → Klassifizierung, nicht nur die Einzelteile.
func TestLusdPreviewHandler_XlsxOhneIDDurchDieVordertuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	geb := time.Date(2012, 3, 4, 0, 0, 0, 0, time.UTC)
	legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Max", nachname: "Mustermann", klasse: "5a", barcode: "UP-1", geb: &geb})

	xlsx := baueXlsx(t, map[string][][]any{
		"Klassenliste": {
			{"Klassenliste — Schuljahr 2026/27"},
			{"Nachname", "Vorname", "Klasse", "Geburtsdatum"},
			{"Mustermann", "Max", "6a", geb},
			{"Neu", "Kind", "5b", time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	teil, err := mw.CreateFormFile("csvFile", "lusd_export.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := teil.Write(xlsx); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/lusd/preview", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	s.PostLusdPreviewHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	var res LusdPreviewResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Modus != "name_geburtsdatum" || len(res.ClassChanges) != 1 || len(res.NewStudents) != 1 {
		t.Fatalf("Vorschau aus Excel falsch: %+v", res)
	}
	// Vorschau schreibt nichts.
	if n := zaehle(t, pool, "nachname='Neu'"); n != 0 {
		t.Errorf("Vorschau hat geschrieben (n=%d)", n)
	}
}
