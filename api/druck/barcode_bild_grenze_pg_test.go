package druck_test

import (
	"net/http"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
)

// GET /api/barcode zeichnet ein Bild in der Größe, die die Anfrage nennt. Ohne Grenze legte der
// Server für width=100000&height=100000 ein Bild von 40 GB an; diesen Speichermangel fängt Go
// nicht ab, der Prozess endet und mit ihm Theke und Anmeldung. Die Route ist von der Bremse
// ausgenommen und verlangt nur view_books, das ab Werk auch die Helfer an der Theke haben.
func TestBarcodeBild_GroesseUndInhaltSindBegrenzt(t *testing.T) {
	pool := pgtest.Pool(t)
	_, rufe := routertest.MitAufruf(t, pool)

	// Die größte Bitte der Oberfläche: ein Strichcode über die ganze Ausweiskarte in 300 dpi
	// (frontend/src/lib/utils/strichcodeBild.js).
	if rec := rufe(t, http.MethodGet, "/api/barcode?content=B-10001&width=1011&height=638", ""); rec.Code != http.StatusOK {
		t.Fatalf("Ausweisgröße: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	// Code 128 trägt höchstens 80 Zeichen (github.com/boombuler/barcode/code128).
	if rec := rufe(t, http.MethodGet, "/api/barcode?content="+strings.Repeat("9", 80), ""); rec.Code != http.StatusOK {
		t.Fatalf("Nummer mit 80 Zeichen: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}

	for name, pfad := range map[string]string{
		"zu breit":       "/api/barcode?content=B-10001&width=5000&height=100",
		"zu hoch":        "/api/barcode?content=B-10001&width=300&height=5000",
		"QR zu groß":     "/api/barcode?content=B-10001&qr=true&width=5000&height=5000",
		"Inhalt zu lang": "/api/barcode?content=" + strings.Repeat("9", 81),
	} {
		if rec := rufe(t, http.MethodGet, pfad, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: Status %d, erwartet 400", name, rec.Code)
		}
	}
}
