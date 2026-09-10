package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Eine Bestellung kennt ihren Topf (Migration 109) — belegt am echten Handler und an der
// Zeile in bestellungen_verlauf, nicht am Request-Struct.
//
// Ohne Topf: 400 an der Tür, keine Bestellung, keine reservierten Barcodes. Mit Topf: die
// Spalte trägt ihn, und die Kundennummer auf dem Beleg ist die des Topfs — die zweite
// Nummer des Händlers, falls hinterlegt, sonst die erste.

// bestelleMitTopf löst eine Bestellung über den echten HTTP-Handler aus; mittel "" heißt:
// das Feld fehlt im Rumpf.
func bestelleMitTopf(t *testing.T, srv *Server, lieferantID, titelID, mittel string) *httptest.ResponseRecorder {
	t.Helper()
	topf := ""
	if mittel != "" {
		topf = fmt.Sprintf(`"mittel":%q,`, mittel)
	}
	rumpf := fmt.Sprintf(`{"supplier_id":%q,%s"items":[{"titel_id":%q,"menge":2,"preis":9.5,"generate_barcodes":false}]}`,
		lieferantID, topf, titelID)
	req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler := srv.SubmitOrderHandler(NewOrderService(srv.DB, repository.NewBookRepository(srv.DB.Pool)), NewPDFService())
	handler(rec, req)
	return rec
}

func zaehleBestellungen(t *testing.T, pool *pgxpool.Pool, lieferantID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferantID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBestellungOhneTopfWirdAbgewiesen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	lieferant := haendler(t, pool, "OhneTopf", false)
	titel := titelMitMeldebestand(t, pool, "LMF-OhneTopf", 0)

	for _, mittel := range []string{"", "kreis"} {
		rec := bestelleMitTopf(t, srv, lieferant, titel, mittel)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("mittel=%q: Status %d, want 400: %s", mittel, rec.Code, rec.Body.String())
		}
	}
	if n := zaehleBestellungen(t, pool, lieferant); n != 0 {
		t.Errorf("%d Bestellung(en) trotz fehlendem Topf angelegt", n)
	}
	// Keine Exemplare reserviert — die Prüfung sitzt VOR dem ersten Datenbankzugriff.
	var exemplare int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM buecher_exemplare`).Scan(&exemplare); err != nil {
		t.Fatal(err)
	}
	if exemplare != 0 {
		t.Errorf("%d Exemplare reserviert, obwohl die Bestellung abgewiesen wurde", exemplare)
	}
}

func TestBestellungTraegtTopfUndKundennummerDesTopfs(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	titel := titelMitMeldebestand(t, pool, "LMF-Topf", 0)

	// Händler MIT zweiter Kundennummer für den Schulträger …
	mitZweiter := haendler(t, pool, "ZweiKonten", false)
	if _, err := pool.Exec(ctx, `UPDATE lieferanten SET kundennummer_schultraeger = 'BIB-77' WHERE id = $1`, mitZweiter); err != nil {
		t.Fatal(err)
	}
	// … und einer ohne.
	ohneZweite := haendler(t, pool, "EinKonto", false)

	faelle := []struct {
		name, lieferant, mittel, kundennummer string
	}{
		{"Land nimmt die erste Nummer", mitZweiter, repository.MittelLand, "K-ZweiKonten"},
		{"Schulträger nimmt die zweite Nummer", mitZweiter, repository.MittelSchultraeger, "BIB-77"},
		{"Schulträger ohne zweite Nummer nimmt die erste", ohneZweite, repository.MittelSchultraeger, "K-EinKonto"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			rec := bestelleMitTopf(t, srv, f.lieferant, titel, f.mittel)
			if rec.Code != http.StatusOK {
				t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
			}
			var mittel, kundennummer string
			err := pool.QueryRow(ctx, `
				SELECT mittel, kundennummer FROM bestellungen_verlauf
				WHERE lieferant_id = $1 ORDER BY bestelldatum DESC LIMIT 1`, f.lieferant).Scan(&mittel, &kundennummer)
			if err != nil {
				t.Fatalf("Bestellung lesen: %v", err)
			}
			if mittel != f.mittel {
				t.Errorf("mittel = %q, want %q", mittel, f.mittel)
			}
			if kundennummer != f.kundennummer {
				t.Errorf("kundennummer = %q, want %q", kundennummer, f.kundennummer)
			}
		})
	}
}

// Der CHECK in der Datenbank ist die zweite Tür: Auch wer an Handler und Service vorbei
// schreibt, bekommt keinen dritten Topf hinein.
func TestBestellungMittelCheckInDerDatenbank(t *testing.T) {
	pool := pgTestPool(t)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
		VALUES ('x', 'x@example.invalid', 'kreis')`)
	if err == nil || !strings.Contains(err.Error(), "bestellungen_verlauf_mittel_check") {
		t.Errorf("CHECK greift nicht: %v", err)
	}
}
