package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// Der Lieferanten-Filter des Bestellberichts wird an der Tür geprüft (OFFEN.md 5.12).
//
// `lieferant_id` kam über eine Query-Variable (`q := r.URL.Query(); q.Get(…)`) herein —
// die eine Schreibweise, die die UUID-Ratsche (uuid_eingaben_test.go) bis zum 22.09.2026
// nicht sah. Der Wert ging ungeprüft in `AND lieferant_id = $n`; ein Text wie `x` endete
// als `invalid input syntax for type uuid` (22P02) und damit als 500 — ein Bedienfehler,
// der aussah wie ein Serverausfall. Am echten Postgres, weil nur der die 22P02 wirft.
func TestBestellbericht_LieferantIDWirdAnDerTuerGeprueft(t *testing.T) {
	pool := pgTestPool(t)
	srv := &Server{DB: &db.Database{Pool: pool}}
	rufe := func(lieferantID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/bestellungen/bericht.pdf?von=2026-09-01&bis=2026-09-30&lieferant_id="+lieferantID, nil)
		rec := httptest.NewRecorder()
		srv.GetBestellBerichtPDFHandler().ServeHTTP(rec, req)
		return rec
	}

	rec := rufe("x")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("lieferant_id=x: Status %d, erwartet 400 — der Wert ging ungeprüft an Postgres: %s", rec.Code, rec.Body.String())
	} else if !strings.Contains(rec.Body.String(), "lieferant_id") {
		t.Errorf("die Antwort nennt das Feld nicht: %s", rec.Body.String())
	}

	// Gegenprobe: Eine gültige Kennung ohne Treffer ist ein leerer Bericht, kein Fehler.
	if rec := rufe("00000000-0000-0000-0000-000000000512"); rec.Code != http.StatusOK {
		t.Errorf("gültige Kennung ohne Treffer: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
}
