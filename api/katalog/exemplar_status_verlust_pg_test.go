package katalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// TestExemplarStatusVerloren_ZaehltAlsVerlust: Wird ein Exemplar in der Schnell-Statusleiste
// auf „Verloren" gesetzt, zählt es in der Verlustquote. Der Grund ist deshalb VERLUST, nicht
// AUSSORTIERT, das repository/statistik.go nicht als Verlust wertet; „Verloren" verschwände
// sonst still aus der Statistik.
func TestExemplarStatusVerloren_ZaehltAlsVerlust(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	testdaten.LegeAusleihbaresExemplarAn(t, pool, "Verlust-Roman", "B-VL-1")
	var exID string
	if err := pool.QueryRow(ctx,
		`SELECT id FROM buecher_exemplare WHERE barcode_id = 'B-VL-1'`).Scan(&exID); err != nil {
		t.Fatalf("Exemplar-ID lesen: %v", err)
	}

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	handler := srv.ExemplarStatusAendern(repository.NewKatalogAbfragen(pool), repository.NewBescheidAbfragen(pool))
	req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+exID+"/status",
		strings.NewReader(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"aus Ranzen verloren"}`))
	req.SetPathValue("id", exID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, tuertest.AlsKonto(req, testdaten.LegeBearbeiterAn(t, pool)))
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
