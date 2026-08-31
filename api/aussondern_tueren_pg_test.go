package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/repository"

	_ "github.com/jackc/pgx/v5/pgxpool"
)

// Zwei Türen zum selben Zustand — beide ohne die Schranke der dritten.
//
// „Ausgesondert" erreichte man auf drei Wegen: DELETE (Ausbuchen, MIT Ausleih-Prüfung
// und Audit), POST …/aussondern und PUT …/status — die letzten beiden bis zum
// 31.08.2026 als nackte UPDATEs: Ein Buch, das ein Schüler gerade in der Tasche hatte,
// ließ sich aussondern. Es verschwand aus Katalog, Kiosk und Inventur, der Schüler
// blieb in der Mahnstrecke, und bei der Rückgabe war das Exemplar gesperrt — ohne
// dass irgendjemand nachvollziehen konnte, wer es ausgebucht hatte. Unbekannte IDs
// waren auf beiden Türen ein stiller „Erfolg".
func TestAussondern_BeideTuerenPruefenAusleihe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bookRepo := repository.NewBookRepository(pool)

	titelID := seedMonitorTitel(t, pool, "Tueren-Titel", "Jug Tue", false, 0)
	verliehen := exemplar(t, pool, titelID, "TUER-VERLIEHEN", true, "")
	frei := exemplar(t, pool, titelID, "TUER-FREI", true, "")
	sid := seedSchueler(t, pool, "S-TUER-1", "Mia", "5a")
	seedLeserAusleihe(t, pool, verliehen, sid)

	aussondern := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/buecher/exemplare/"+id+"/aussondern", nil)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		(&Server{}).AussondernCopyHandler(bookRepo)(rec, req)
		return rec
	}
	statusAussondern := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+id+"/status",
			strings.NewReader(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"weg damit"}`))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		(&Server{}).UpdateCopyStatusHandler(bookRepo)(rec, req)
		return rec
	}
	istAusgesondert := func(id string) bool {
		var aus bool
		if err := pool.QueryRow(context.Background(),
			`SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = $1`, id).Scan(&aus); err != nil {
			t.Fatalf("ist_ausgesondert lesen: %v", err)
		}
		return aus
	}

	// Tür 1: POST /aussondern auf ein verliehenes Exemplar — muss abgelehnt werden.
	if rec := aussondern(verliehen); rec.Code != http.StatusBadRequest {
		t.Errorf("aussondern verliehen: HTTP %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if istAusgesondert(verliehen) {
		t.Error("Tür 1 hat ein verliehenes Exemplar ausgesondert")
	}

	// Tür 2: PUT /status mit ist_ausgesondert=true — dieselbe Schranke.
	if rec := statusAussondern(verliehen); rec.Code != http.StatusBadRequest {
		t.Errorf("status→ausgesondert verliehen: HTTP %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if istAusgesondert(verliehen) {
		t.Error("Tür 2 hat ein verliehenes Exemplar ausgesondert")
	}

	// Unbekannte ID: 404 statt stillem Erfolg — auf beiden Türen.
	unbekannt := "00000000-0000-0000-0000-000000000d0d"
	if rec := aussondern(unbekannt); rec.Code != http.StatusNotFound {
		t.Errorf("aussondern unbekannt: HTTP %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if rec := statusAussondern(unbekannt); rec.Code != http.StatusNotFound {
		t.Errorf("status unbekannt: HTTP %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}

	// Freies Exemplar: beide Türen funktionieren weiter.
	if rec := aussondern(frei); rec.Code != http.StatusOK {
		t.Fatalf("aussondern frei: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if !istAusgesondert(frei) {
		t.Error("freies Exemplar wurde nicht ausgesondert")
	}
}
