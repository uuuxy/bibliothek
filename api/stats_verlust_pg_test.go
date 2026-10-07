package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Ein bestelltes Exemplar, das nie eintraf und als verloren ausgebucht wurde, ist kein Verlust.
// Die Statistik zählt in der Grenze des Abgangsbuchs; ein Exemplar aus dem Bestand ist die
// Gegenprobe.
func TestStatistik_NieEingetroffenesIstKeinVerlust(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	titelID := titelMitSignatur(t, pool, "Statistik Zulauf", "Sta 1", 0)
	nie := zulaufExemplar(t, pool, titelID, "ST-ZULAUF")
	imBestand := exemplar(t, pool, titelID, "ST-BESTAND", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET einkaufspreis = 20 WHERE id = ANY($1)`,
		[]string{nie, imBestand}); err != nil {
		t.Fatalf("Preis setzen: %v", err)
	}
	buecher := repository.NewBookRepository(pool)
	for _, id := range []string{nie, imBestand} {
		if err := buecher.UpdateCopyStatus(ctx, id, false, true, "", nil); err != nil {
			t.Fatalf("als verloren ausbuchen: %v", err)
		}
	}

	rec := httptest.NewRecorder()
	srv.GetStatisticsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/statistiken", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Verluste struct {
			Verlorene int `json:"verlorene_exemplare"`
		} `json:"loss_stats"`
		Wiederbeschaffung float64 `json:"wiederbeschaffungswert_defekt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if antwort.Verluste.Verlorene != 1 {
		t.Errorf("verlorene Exemplare: %d, erwartet 1 — ST-ZULAUF war nie im Bestand", antwort.Verluste.Verlorene)
	}
	if antwort.Wiederbeschaffung != 20 {
		t.Errorf("Wiederbeschaffungswert: %.2f, erwartet 20.00 — nur ST-BESTAND muss nachgekauft werden", antwort.Wiederbeschaffung)
	}
}
