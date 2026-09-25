package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// Die Türen der Titelmaske für Auflagen (Migration 148, docs/OFFEN.md 4.18). Die Regeln
// selbst prüft repository/auflagen_pg_test.go; hier geht es um das, was nur an der Tür
// entstehen kann — eine falsche Kennung im Pfad oder im Körper, ein fehlendes Feld, die
// Übersetzung der fachlichen Fehler in 400 mit Text und 404.
func TestTitelAuflagen_TuerDerTitelmaske(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE werke CASCADE`); err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: &db.Database{Pool: pool}}
	titel := func(name string, jahr int, lernmittel bool) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel, erscheinungsjahr, ist_lernmittel) VALUES ($1, $2, $3) RETURNING id`,
			name, jahr, lernmittel).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	alt := titel("Lambacher Schweizer 7", 2019, true)
	neu := titel("Lambacher Schweizer 7", 2023, true)
	roman := titel("Tintenherz", 2003, false)

	aufruf := func(methode, titelID, koerper string) (int, TitelAuflagen, string) {
		t.Helper()
		req := httptest.NewRequest(methode, "/api/buecher/titel/"+titelID+"/auflagen", strings.NewReader(koerper))
		req.SetPathValue("id", titelID)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		switch methode {
		case http.MethodGet:
			srv.GetTitelAuflagenHandler()(rec, req)
		case http.MethodPost:
			srv.PostTitelAuflagenHandler()(rec, req)
		default:
			srv.DeleteTitelAuflagenHandler()(rec, req)
		}
		var antwort TitelAuflagen
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort unlesbar: %v — %s", err, rec.Body.String())
			}
		}
		return rec.Code, antwort, rec.Body.String()
	}

	// Ohne andere Auflage steht der Titel allein in der Liste — eine Liste, nicht null.
	code, antwort, text := aufruf(http.MethodGet, alt, "")
	if code != http.StatusOK || len(antwort.Auflagen) != 1 || antwort.Auflagen[0].ID != alt {
		t.Fatalf("GET ohne Werk: %d %s", code, text)
	}

	code, antwort, text = aufruf(http.MethodPost, alt, `{"titel_id":"`+neu+`"}`)
	if code != http.StatusOK || len(antwort.Auflagen) != 2 || antwort.Auflagen[0].ID != neu {
		t.Fatalf("POST: %d %s — erwartet beide Auflagen, die neue zuerst", code, text)
	}
	if code, antwort, text := aufruf(http.MethodGet, neu, ""); code != http.StatusOK || len(antwort.Auflagen) != 2 {
		t.Errorf("GET von der anderen Seite: %d %s", code, text)
	}

	// Die Regel kommt mit ihrem Text an, nicht als 500.
	code, _, text = aufruf(http.MethodPost, alt, `{"titel_id":"`+roman+`"}`)
	if code != http.StatusBadRequest || !strings.Contains(text, "kein Lernmittel") {
		t.Errorf("POST mit Roman: %d %s, erwartet 400 mit „kein Lernmittel“", code, text)
	}

	for _, koerper := range []string{`{}`, `{"titel_id":""}`, `{"titel_id":"keine-uuid"}`} {
		if code, _, text := aufruf(http.MethodPost, alt, koerper); code != http.StatusBadRequest {
			t.Errorf("POST %s: %d, erwartet 400: %s", koerper, code, text)
		}
	}

	const unbekannt = "00000000-0000-0000-0000-000000000148"
	if code, _, text := aufruf(http.MethodPost, alt, `{"titel_id":"`+unbekannt+`"}`); code != http.StatusNotFound {
		t.Errorf("POST mit unbekanntem Titel im Körper: %d, erwartet 404: %s", code, text)
	}
	for _, methode := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		if code, _, text := aufruf(methode, unbekannt, `{"titel_id":"`+neu+`"}`); code != http.StatusNotFound {
			t.Errorf("%s unbekannter Titel: %d, erwartet 404: %s", methode, code, text)
		}
		if code, _, text := aufruf(methode, "keine-uuid", `{"titel_id":"`+neu+`"}`); code != http.StatusBadRequest {
			t.Errorf("%s ohne UUID: %d, erwartet 400: %s", methode, code, text)
		}
	}

	code, antwort, text = aufruf(http.MethodDelete, neu, "")
	if code != http.StatusOK || len(antwort.Auflagen) != 1 || antwort.Auflagen[0].ID != neu {
		t.Fatalf("DELETE: %d %s — erwartet nur den gelösten Titel", code, text)
	}
	if code, antwort, _ := aufruf(http.MethodGet, alt, ""); code != http.StatusOK || len(antwort.Auflagen) != 1 {
		t.Errorf("nach dem Lösen hat alt noch %d Auflagen, erwartet 1", len(antwort.Auflagen))
	}
}
