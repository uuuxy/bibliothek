package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Die Status-Tür prüft die Ausleihe — wie das Ausbuchen.
//
// „Ausgesondert" erreichte man auf drei Wegen: DELETE (Ausbuchen, MIT Ausleih-Prüfung
// und Audit), POST …/aussondern und PUT …/status — die letzten beiden bis zum
// 31.08.2026 als nackte UPDATEs: Ein Buch, das ein Schüler gerade in der Tasche hatte,
// ließ sich aussondern. Es verschwand aus Katalog, Kiosk und Inventur, der Schüler
// blieb in der Mahnstrecke, und bei der Rückgabe war das Exemplar gesperrt — ohne
// dass irgendjemand nachvollziehen konnte, wer es ausgebucht hatte. Unbekannte IDs
// waren auf beiden Türen ein stiller „Erfolg".
//
// POST …/aussondern ist am 22.09.2026 gestrichen (OFFEN.md 4.16): keine Oberfläche rief
// sie, und ein dritter Weg zum selben Zustand muss jede künftige Regel kennen. Es
// bleiben das Ausbuchen und der Status-Editor; dieser Test hält die Schranke am
// Status-Editor.
func TestAussondern_StatusTuerPrueftAusleihe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bookRepo := repository.NewBookRepository(pool)

	titelID := seedMonitorTitel(t, pool, "Tueren-Titel", "Jug Tue", false, 0)
	verliehen := exemplar(t, pool, titelID, "TUER-VERLIEHEN", true, "")
	frei := exemplar(t, pool, titelID, "TUER-FREI", true, "")
	sid := seedSchueler(t, pool, "S-TUER-1", "Mia", "5a")
	seedLeserAusleihe(t, pool, verliehen, sid)

	statusAussondern := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+id+"/status",
			strings.NewReader(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"weg damit"}`))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		// Server MIT Datenbank: Der Erfolgsfall fragt den Ersatzwert nach (9.8, Stufe 2b).
		(&Server{DB: &db.Database{Pool: pool}}).UpdateCopyStatusHandler(bookRepo, repository.NewBescheidRepository(pool))(rec, req)
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

	// PUT /status mit ist_ausgesondert=true auf ein verliehenes Exemplar — abgelehnt.
	if rec := statusAussondern(verliehen); rec.Code != http.StatusBadRequest {
		t.Errorf("status→ausgesondert verliehen: HTTP %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if istAusgesondert(verliehen) {
		t.Error("die Status-Tür hat ein verliehenes Exemplar ausgesondert")
	}

	// Unbekannte ID: 404 statt stillem Erfolg.
	unbekannt := "00000000-0000-0000-0000-000000000d0d"
	if rec := statusAussondern(unbekannt); rec.Code != http.StatusNotFound {
		t.Errorf("status unbekannt: HTTP %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}

	// Freies Exemplar: die Tür funktioniert weiter.
	if rec := statusAussondern(frei); rec.Code != http.StatusOK {
		t.Fatalf("status→ausgesondert frei: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if !istAusgesondert(frei) {
		t.Error("freies Exemplar wurde nicht ausgesondert")
	}
}
