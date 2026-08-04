package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRequestFrist belegt die Fristen-Zuordnung ohne HTTP-Aufbau.
//
// Hintergrund (Audit 01.08.2026): Alle Pfade ausser /events liefen unter derselben
// 15-Sekunden-Frist — auch der Littera-Import. Der echte Altbestand hat 11.076 Titel
// und 61.580 Exemplare; darunter bricht der Server mitten in der Transaktion ab,
// waehrend der Browser noch 5 Minuten wartet.
func TestRequestFrist(t *testing.T) {
	const standard = 15 * time.Second

	faelle := []struct {
		pfad     string
		erwartet time.Duration
		warum    string
	}{
		{"/api/import/littera", LangLaufendeFrist, "Katalogimport dauert Minuten"},
		{"/api/admin/import-bestand", LangLaufendeFrist, "Bestandsimport dauert Minuten"},
		{"/api/admin/sync-covers", LangLaufendeFrist, "Cover-Abgleich laeuft ueber den ganzen Bestand"},
		{"/api/lusd/import", LangLaufendeFrist, "LUSD-Import umfasst ganze Jahrgaenge"},
		{"/api/lusd/preview", LangLaufendeFrist, "Vorschau liest dieselbe Datei"},
		{"/api/admin/mahnungen/bulk-print", LangLaufendeFrist, "Sammeldruck ueber alle Klassen"},

		// Alltagspfade behalten die kurze Frist — sonst haelt ein haengender Scan
		// minutenlang eine Datenbankverbindung fest.
		{"/api/buecher", standard, "normale Abfrage"},
		{"/api/inventur/scan", standard, "Scan muss schnell sein"},
		{"/api/schueler", standard, "normale Abfrage"},
		// Praefix-Grenze: /api/importe waere KEIN Import-Pfad.
		{"/api/importeur", standard, "aehnlicher Name, anderer Endpunkt"},
	}

	for _, f := range faelle {
		if got := RequestFrist(f.pfad, standard); got != f.erwartet {
			t.Errorf("RequestFrist(%q) = %v, erwartet %v (%s)", f.pfad, got, f.erwartet, f.warum)
		}
	}
}

// TestUUIDPruefungGreiftAmMux ist der Regressionstest zu Audit-Punkt 3.
//
// ValidateUUIDParamsMiddleware las r.PathValue("id"), umschloss den Mux aber von AUSSEN.
// PathValue fuellt erst der ServeMux beim Treffer — zur Laufzeit der Middleware war der
// Wert daher immer leer und die Pruefung lief nie. Der Test faehrt beide Anordnungen und
// zeigt den Unterschied am Verhalten, nicht am Code.
func TestUUIDPruefungGreiftAmMux(t *testing.T) {
	ziel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux := http.NewServeMux()
	mux.Handle("GET /api/dinge/{id}", ValidateUUIDParamsMiddleware(ziel))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/dinge/kein-uuid", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ungueltige UUID INNERHALB des Mux: erwartet 400, war %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/dinge/3f2504e0-4f89-11d3-9a0c-0305e82c3301", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("gueltige UUID: erwartet 200, war %d", rec.Code)
	}

	// Und der Beleg fuer den Befund: dieselbe Middleware VON AUSSEN um einen Mux gelegt,
	// der sie NICHT selbst traegt, laesst den ungueltigen Wert durch — PathValue ist dort
	// noch leer. Genau so war sie in router.go verdrahtet.
	blankerMux := http.NewServeMux()
	blankerMux.Handle("GET /api/dinge/{id}", ziel)
	aussen := ValidateUUIDParamsMiddleware(blankerMux)

	rec = httptest.NewRecorder()
	aussen.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/dinge/kein-uuid", nil))
	if rec.Code == http.StatusBadRequest {
		t.Error("unerwartet: von aussen umschlossen greift die Pruefung doch — Befund neu bewerten")
	}
}
