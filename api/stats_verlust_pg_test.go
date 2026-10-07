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

// Ein bestelltes Exemplar zählt erst zum Bestand, wenn es eingetroffen ist. Sonst stünde es im
// Gesamtbestand und unter den aktiven Exemplaren, und die Zirkulationsquote teilte durch es mit.
func TestStatistik_BestelltesZaehltNichtZumBestand(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	titelID := titelMitSignatur(t, pool, "Statistik Bestand", "Sta 2", 0)
	zulaufExemplar(t, pool, titelID, "SB-ZULAUF")
	imBestand := exemplar(t, pool, titelID, "SB-BESTAND", true, "")
	seedLeserAusleihe(t, pool, imBestand, seedSchueler(t, pool, "SB-LESER", "Statistik", "07A"))

	rec := httptest.NewRecorder()
	srv.GetStatisticsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/statistiken", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Verluste struct {
			Gesamt int `json:"gesamt_bestand"`
		} `json:"loss_stats"`
		Zirkulation struct {
			Verliehen int `json:"aktuell_verliehen"`
			Aktiv     int `json:"aktiver_bestand"`
		} `json:"zirkulation"`
		Quote float64 `json:"zirkulationsquote"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if antwort.Verluste.Gesamt != 1 || antwort.Zirkulation.Aktiv != 1 {
		t.Errorf("Gesamtbestand %d, aktive Exemplare %d, erwartet je 1 — SB-ZULAUF ist noch nicht eingetroffen",
			antwort.Verluste.Gesamt, antwort.Zirkulation.Aktiv)
	}
	if antwort.Zirkulation.Verliehen != 1 || antwort.Quote != 100 {
		t.Errorf("verliehen %d, Zirkulationsquote %.2f, erwartet 1 und 100.00",
			antwort.Zirkulation.Verliehen, antwort.Quote)
	}
}
