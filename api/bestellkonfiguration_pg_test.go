package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Der Warenkorb warnt vor dem Bestellen, wenn der Hauptlieferant keinen Bestätigungs-Link
// bekäme: Es gibt einen Hauptlieferanten, aber keine öffentliche Adresse. Ein Feld aus lauter
// Leerzeichen ist keine Adresse, daraus entstünde kein Link.
func TestBestellKonfiguration_WarntWennDerLinkNichtEntstehenKann(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	// Ein Hauptlieferant aus einem früheren Test dieses Laufs zählte sonst mit.
	if _, err := pool.Exec(t.Context(), `UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant`); err != nil {
		t.Fatalf("bisherigen Hauptlieferanten räumen: %v", err)
	}
	t.Cleanup(func() { setzeOeffentlicheAdresse(t, pool, "") })
	srv := &Server{DB: &db.Database{Pool: pool}}
	ohneAdresse := func() bool {
		t.Helper()
		rec := httptest.NewRecorder()
		srv.BestellKonfigurationHandler(repository.NewSystemSettingsRepository(pool), repository.NewSupplierRepository(pool)).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bestellungen/konfiguration", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
		}
		var antwort BestellKonfiguration
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v", err)
		}
		return antwort.BestelllinkOhneAdresse
	}

	setzeOeffentlicheAdresse(t, pool, "")
	if ohneAdresse() {
		t.Error("ohne Hauptlieferant warnt die Konfiguration, obwohl kein Link verschickt würde")
	}
	haendler(t, pool, "Naacher", true)
	if !ohneAdresse() {
		t.Error("Hauptlieferant ohne öffentliche Adresse: keine Warnung")
	}
	setzeOeffentlicheAdresse(t, pool, "   ")
	if !ohneAdresse() {
		t.Error("eine Adresse aus lauter Leerzeichen gilt als hinterlegt: keine Warnung")
	}
	setzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	if ohneAdresse() {
		t.Error("mit öffentlicher Adresse warnt die Konfiguration weiter")
	}
}
