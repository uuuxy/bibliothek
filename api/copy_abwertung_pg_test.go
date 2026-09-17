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

// Der Beschädigungsgrad am ECHTEN Postgres — Schreiben, Lesen und das Nicht-Anfassen.
//
// Drei Dinge fängt nur dieser Test:
//
//  1. Die Spalte gibt es wirklich und sie nimmt den Wert (der Mock-Test glaubt jedes SQL).
//  2. Die Lese-Tür liefert ihn in der RICHTIGEN Reihenfolge mit. Beim Einfügen einer
//     Spalte in die Mitte eines SELECTs verschiebt sich jeder Scan danach — das ergibt
//     stille Verwechslungen, hier zwischen „beschädigt" und „verfügbar".
//  3. Ein Statuswechsel ohne das Feld lässt den erfassten Schaden stehen.
func TestExemplarAbwertung_SchreibenLesenUndUnangetastet(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	ausleihbaresExemplar(t, pool, "Atlas mit Wasserrand", "B-ABW-1")
	var exID, titelID string
	if err := pool.QueryRow(ctx,
		`SELECT id, titel_id FROM buecher_exemplare WHERE barcode_id = 'B-ABW-1'`,
	).Scan(&exID, &titelID); err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	statusTuer := srv.UpdateCopyStatusHandler(repository.NewBookRepository(pool))

	setze := func(t *testing.T, koerper string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+exID+"/status",
			strings.NewReader(koerper))
		req.SetPathValue("id", exID)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		statusTuer.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Status setzen (%s): erwartet 200, war %d: %s", koerper, rec.Code, rec.Body.String())
		}
	}
	grad := func(t *testing.T) int {
		t.Helper()
		var p int
		if err := pool.QueryRow(ctx,
			`SELECT zustand_abwertung_prozent FROM buecher_exemplare WHERE id = $1`, exID).Scan(&p); err != nil {
			t.Fatalf("Grad lesen: %v", err)
		}
		return p
	}

	// Ausgangslage: jedes Exemplar startet bei 0 (NOT NULL DEFAULT 0, Migration 127).
	if p := grad(t); p != 0 {
		t.Fatalf("Startwert = %d, erwartet 0", p)
	}

	// 1. Erfassen: 20 % Wasserschaden, Buch bleibt im Umlauf.
	setze(t, `{"ist_ausleihbar":true,"ist_ausgesondert":false,"zustand_notiz":"","zustand_abwertung_prozent":20}`)
	if p := grad(t); p != 20 {
		t.Fatalf("nach dem Erfassen = %d, erwartet 20", p)
	}

	// 2. Ein Statuswechsel OHNE das Feld darf den Schaden nicht räumen.
	setze(t, `{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"in Reparatur"}`)
	if p := grad(t); p != 20 {
		t.Errorf("nach dem Sperren = %d, erwartet 20 — ein Statuswechsel ohne das Feld "+
			"hat den erfassten Schaden gelöscht", p)
	}

	// 3. Freigeben räumt die Notiz (bestehende Regel), den Schaden aber nicht.
	setze(t, `{"ist_ausleihbar":true,"ist_ausgesondert":false,"zustand_notiz":"in Reparatur"}`)
	var notiz string
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(zustand_notiz, '') FROM buecher_exemplare WHERE id = $1`, exID).Scan(&notiz); err != nil {
		t.Fatalf("Notiz lesen: %v", err)
	}
	if notiz != "" {
		t.Errorf("Notiz = %q, erwartet leer — Freigeben räumt sie", notiz)
	}
	if p := grad(t); p != 20 {
		t.Errorf("nach dem Freigeben = %d, erwartet 20 — ein Band mit Wasserrand darf "+
			"ausleihbar sein und trägt seinen Abschlag weiter", p)
	}

	// 4. Die Lese-Tür der Buchakte liefert den Grad mit — an der richtigen Stelle.
	leseReq := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/"+titelID+"/exemplare", nil)
	leseReq.SetPathValue("id", titelID)
	leseRec := httptest.NewRecorder()
	srv.GetTitleCopiesHandler().ServeHTTP(leseRec, leseReq)
	if leseRec.Code != http.StatusOK {
		t.Fatalf("Exemplare lesen: erwartet 200, war %d: %s", leseRec.Code, leseRec.Body.String())
	}
	var gelesen []struct {
		BarcodeID               string `json:"barcode_id"`
		ZustandAbwertungProzent int    `json:"zustand_abwertung_prozent"`
		IstAusleihbar           bool   `json:"ist_ausleihbar"`
		IstVerfuegbar           bool   `json:"ist_verfuegbar"`
	}
	if err := json.Unmarshal(leseRec.Body.Bytes(), &gelesen); err != nil {
		t.Fatalf("Antwort lesen: %v — %s", err, leseRec.Body.String())
	}
	if len(gelesen) != 1 {
		t.Fatalf("erwartet ein Exemplar, waren %d", len(gelesen))
	}
	if gelesen[0].ZustandAbwertungProzent != 20 {
		t.Errorf("gelesener Grad = %d, erwartet 20", gelesen[0].ZustandAbwertungProzent)
	}
	// Die beiden Nachbarfelder belegen, dass die Scan-Reihenfolge stimmt: Das Exemplar
	// ist ausleihbar und nicht verliehen. Wäre der Scan verschoben, stünde hier false.
	if !gelesen[0].IstAusleihbar || !gelesen[0].IstVerfuegbar {
		t.Errorf("ist_ausleihbar=%v, ist_verfuegbar=%v — beide müssen true sein; "+
			"false heißt: die neue Spalte hat den Scan verschoben",
			gelesen[0].IstAusleihbar, gelesen[0].IstVerfuegbar)
	}
	if gelesen[0].BarcodeID != "B-ABW-1" {
		t.Errorf("barcode_id = %q, erwartet B-ABW-1", gelesen[0].BarcodeID)
	}
}
