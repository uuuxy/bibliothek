package tuer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Ein freiwilliger Rumpf wird gelesen, wenn einer kommt, auch ohne Längenangabe im Kopf
// (HTTP/2, chunked: ContentLength -1). Vorher verwarf die Theke ihn dann still, und die gezeigte
// Frist schützte nicht mehr vor dem doppelten Verlängern. Ohne Rumpf bleibt das Ziel, wie es ist.
func TestLeseFreiwilligeAnfrage(t *testing.T) {
	type anfrage struct {
		Wert string `json:"wert"`
	}
	lies := func(rumpf string, laenge int64) (anfrage, bool, int) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(rumpf))
		req.ContentLength = laenge
		rec := httptest.NewRecorder()
		var ziel anfrage
		ok := LeseFreiwilligeAnfrage(rec, req, &ziel)
		return ziel, ok, rec.Code
	}

	if ziel, ok, _ := lies(`{"wert":"gezeigt"}`, -1); !ok || ziel.Wert != "gezeigt" {
		t.Errorf("ohne Längenangabe: %+v, ok=%v; erwartet den Wert aus dem Rumpf", ziel, ok)
	}
	if ziel, ok, _ := lies(`{"wert":"gezeigt"}`, 18); !ok || ziel.Wert != "gezeigt" {
		t.Errorf("mit Längenangabe: %+v, ok=%v; erwartet den Wert aus dem Rumpf", ziel, ok)
	}
	for _, laenge := range []int64{0, -1} {
		if ziel, ok, _ := lies("", laenge); !ok || ziel.Wert != "" {
			t.Errorf("leerer Rumpf, Länge %d: %+v, ok=%v; erwartet ohne Wert und ok", laenge, ziel, ok)
		}
	}
	if _, ok, code := lies(`{"wert":`, -1); ok || code != http.StatusBadRequest {
		t.Errorf("unlesbarer Rumpf: ok=%v, Status %d; erwartet 400", ok, code)
	}
}
