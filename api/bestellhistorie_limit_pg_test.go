package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
)

// Die Bestellhistorie ist gedeckelt — und ihre Kennzahlen sind es NICHT.
//
// Beides gehört zusammen geprüft: Ein Limit allein macht aus „Gesamtausgaben" still eine
// Teilsumme, die aussieht wie eine Gesamtsumme. Genau das wäre schlimmer als der lange
// Ladevorgang, den das Limit behebt (2,45 MB / 3,9 s auf einer gewachsenen Datenbank).
//
// Die Tests messen DIFFERENZEN statt absoluter Zahlen: Sie teilen sich die Test-DB mit
// dem übrigen Paket, und eine Erwartung wie „genau 6 Bestellungen" wäre grün, solange
// dieser Test allein läuft, und rot in der vollen Suite.

func holeHistorie(t *testing.T, srv *Server, query string) []BestellVerlaufResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/bestellhistorie"+query, nil)
	rec := httptest.NewRecorder()
	srv.GetBestellhistorieHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Historie Status = %d, body: %s", rec.Code, rec.Body.String())
	}

	var orders []BestellVerlaufResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &orders); err != nil {
		t.Fatalf("Historie lesen: %v", err)
	}
	return orders
}

func holeUebersicht(t *testing.T, srv *Server) BestellhistorieUebersicht {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.GetBestellhistorieUebersichtHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bestellhistorie/uebersicht", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Übersicht Status = %d, body: %s", rec.Code, rec.Body.String())
	}

	var u BestellhistorieUebersicht
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatalf("Übersicht lesen: %v", err)
	}
	return u
}

func TestBestellhistorie_UebersichtZaehltAlleTrotzLimit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	vorher := holeUebersicht(t, srv)

	mitBestaetigung := haendler(t, pool, "Naacher-Uebersicht", true)
	ohneBestaetigung := haendler(t, pool, "Normalo-Uebersicht", false)

	// Drei wartende, eine bestätigte, zwei ohne den externen Schritt.
	for i := 0; i < 3; i++ {
		bestellungFuerLieferant(t, pool, mitBestaetigung)
	}
	bestaetigte := bestellungFuerLieferant(t, pool, mitBestaetigung)
	if _, err := srv.bestaetigeBestellung(ctx, bestaetigte, "", "lieferant"); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}
	for i := 0; i < 2; i++ {
		bestellungFuerLieferant(t, pool, ohneBestaetigung)
	}

	// Das Limit greift auf der Liste …
	if geladen := holeHistorie(t, srv, "?limit=2"); len(geladen) != 2 {
		t.Fatalf("mit limit=2 geladen: %d Bestellungen, want 2", len(geladen))
	}

	// … die Kennzahlen sehen davon unbeeindruckt alle sechs neuen Bestellungen.
	nachher := holeUebersicht(t, srv)
	if delta := nachher.Gesamt - vorher.Gesamt; delta != 6 {
		t.Errorf("Gesamt stieg um %d, want 6", delta)
	}
	// Nur Lieferanten mit dem externen Schritt können warten — die zwei Bestellungen bei
	// „Normalo" dürfen den Zähler nicht aufblähen, sonst sucht jemand eine Bestätigung,
	// die es für diesen Händler gar nicht gibt.
	if delta := nachher.OffeneBestaetigungen - vorher.OffeneBestaetigungen; delta != 3 {
		t.Errorf("OffeneBestaetigungen stiegen um %d, want 3", delta)
	}
	if delta := nachher.GesamtExemplare - vorher.GesamtExemplare; delta != 18 {
		t.Errorf("GesamtExemplare stiegen um %d, want 18 (6 × 3)", delta)
	}
}

// Die Obergrenze muss auch dann halten, wenn jemand ?limit=99999 anhängt — sonst wäre
// der Deckel eine Bitte und keine Grenze.
func TestBestellhistorie_MaxLimitIstNichtVerhandelbar(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendler(t, pool, "Naacher-Masse", true)
	// In einer Anweisung, damit der Test schnell bleibt: mehr Zeilen als die Obergrenze.
	if _, err := pool.Exec(ctx, `
		INSERT INTO bestellungen_verlauf (lieferant_id, lieferant_name, lieferant_email, kundennummer, anzahl_exemplare)
		SELECT $1, 'Massen-Test', 'masse@example.invalid', 'K-MASSE', 1 FROM generate_series(1, $2)
	`, lieferant, bestellhistorieMaxLimit+20); err != nil {
		t.Fatalf("Massendaten anlegen: %v", err)
	}

	if geladen := holeHistorie(t, srv, "?limit=99999"); len(geladen) != bestellhistorieMaxLimit {
		t.Fatalf("geladen: %d, want %d (Obergrenze)", len(geladen), bestellhistorieMaxLimit)
	}
	if geladen := holeHistorie(t, srv, ""); len(geladen) != bestellhistorieStandardLimit {
		t.Fatalf("ohne Parameter geladen: %d, want %d (Vorgabe)", len(geladen), bestellhistorieStandardLimit)
	}
}

// Positionen dürfen nur für die geladenen Bestellungen kommen. Vorher holte die Abfrage
// die Positionen ALLER Bestellungen und warf die überzähligen beim Zuordnen weg — mit
// dem Limit wäre das der teuerste Teil der Anfrage geblieben.
func TestBestellhistorie_PositionenNurFuerGeladeneBestellungen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendler(t, pool, "Naacher-Positionen", true)
	for i := 0; i < 3; i++ {
		id := bestellungFuerLieferant(t, pool, lieferant)
		if _, err := pool.Exec(ctx, `
			INSERT INTO bestellungen_positionen (bestellung_id, titel_name, isbn, menge, einzelpreis)
			VALUES ($1, 'Titel', '', 1, 0)`, id); err != nil {
			t.Fatalf("Position anlegen: %v", err)
		}
	}

	geladen := holeHistorie(t, srv, "?limit=1")
	if len(geladen) != 1 {
		t.Fatalf("geladen = %d, want 1", len(geladen))
	}
	if len(geladen[0].Positionen) != 1 {
		t.Fatalf("Positionen = %d, want 1 — die Zuordnung muss auch mit Limit stimmen", len(geladen[0].Positionen))
	}
}
