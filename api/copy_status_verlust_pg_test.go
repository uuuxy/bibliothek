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

// TestCopyStatusVerloren_ZaehltAlsVerlust belegt die Betreiber-Entscheidung vom
// 19.08.2026 (Vokabular-Sweep P1): Wird ein Exemplar in der Schnell-Statusleiste auf
// "Verloren" gesetzt, muss es in der Verlustquote zählen. Vorher schrieb der Editor
// AUSSORTIERT, das api/stats.go ausdrücklich NICHT als Verlust wertet — "Verloren"
// verschwand still aus der Statistik. Jetzt ist der Default VERLUST.
func TestCopyStatusVerloren_ZaehltAlsVerlust(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	ausleihbaresExemplar(t, pool, "Verlust-Roman", "B-VL-1")
	var exID string
	if err := pool.QueryRow(ctx,
		`SELECT id FROM buecher_exemplare WHERE barcode_id = 'B-VL-1'`).Scan(&exID); err != nil {
		t.Fatalf("Exemplar-ID lesen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	handler := srv.UpdateCopyStatusHandler(repository.NewBookRepository(pool))
	req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+exID+"/status",
		strings.NewReader(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"aus Ranzen verloren"}`))
	req.SetPathValue("id", exID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status setzen: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}

	var grund string
	if err := pool.QueryRow(ctx,
		`SELECT aussonderung_grund FROM buecher_exemplare WHERE id = $1`, exID).Scan(&grund); err != nil {
		t.Fatalf("Grund lesen: %v", err)
	}
	if grund != "VERLUST" {
		t.Errorf(`"Verloren" muss als VERLUST gebucht werden, war %q`, grund)
	}

	// Und der Verlust-Filter der Statistik (VERLUST/BESCHAEDIGUNG) erfasst es jetzt.
	var alsVerlustGezaehlt bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM buecher_exemplare e
			WHERE e.id = $1 AND e.aussonderung_grund IN ('VERLUST', 'BESCHAEDIGUNG')
		)`, exID).Scan(&alsVerlustGezaehlt); err != nil {
		t.Fatalf("Verlust-Filter prüfen: %v", err)
	}
	if !alsVerlustGezaehlt {
		t.Error("das verlorene Exemplar fällt nicht unter den Verlust-Filter der Statistik")
	}
}
