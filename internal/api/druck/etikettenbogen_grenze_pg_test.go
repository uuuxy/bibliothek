package druck_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"bibliothek/internal/api/druck"
	"bibliothek/internal/pdf"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
)

// POST /api/print/labels nimmt höchstens druck.MaxBuchEtiketten Etiketten je Auftrag an, auf
// beiden Wegen (Bogen und Lernmittel-Etikett). Ohne die Grenze rechnete der Server an einem
// Auftrag mit 150.000 Einträgen weiter, nachdem die Anfrage längst abgelaufen war.
func TestEtikettenbogen_ZuGrosserAuftrag(t *testing.T) {
	pool := pgtest.Pool(t)
	_, rufe := routertest.MitAufruf(t, pool)

	etiketten := make([]map[string]string, druck.MaxBuchEtiketten+1)
	for i := range etiketten {
		etiketten[i] = map[string]string{"BarcodeID": fmt.Sprintf("%08d", i+1)}
	}
	for _, format := range []string{"zweckform_l4760", pdf.LernmittelEtikettFormat} {
		t.Run(format, func(t *testing.T) {
			roh, err := json.Marshal(map[string]any{"formatId": format, "startPosition": 1, "items": etiketten})
			if err != nil {
				t.Fatal(err)
			}
			rec := rufe(t, http.MethodPost, "/api/print/labels", string(roh))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), strconv.Itoa(druck.MaxBuchEtiketten)) {
				t.Errorf("die Meldung nennt die Grenze nicht: %s", rec.Body.String())
			}
		})
	}
}

// Ein unbekanntes Format und eine Startposition außerhalb des Bogens weist POST /api/print/labels
// ab, wie die Schüler-Etiketten. Bisher druckte ein vertipptes Format still auf den Bogen der
// Vorgabe, und das fiel erst am verdruckten Bogen auf.
func TestEtikettenbogen_PrueftFormatUndStartposition(t *testing.T) {
	pool := pgtest.Pool(t)
	_, rufe := routertest.MitAufruf(t, pool)
	etiketten := []map[string]string{{"BarcodeID": "00000001"}}

	for _, fall := range []struct {
		name   string
		format string
		start  int
		status int
	}{
		{"unbekanntes Format", "avery_9999", 1, http.StatusBadRequest},
		{"Startposition hinter dem Bogen", "zweckform_l4760", 22, http.StatusBadRequest},
		{"letzter Platz des Bogens", "zweckform_l4760", 21, http.StatusOK},
		{"ohne Format die Vorgabe", "", 0, http.StatusOK},
	} {
		t.Run(fall.name, func(t *testing.T) {
			roh, err := json.Marshal(map[string]any{"formatId": fall.format, "startPosition": fall.start, "items": etiketten})
			if err != nil {
				t.Fatal(err)
			}
			if rec := rufe(t, http.MethodPost, "/api/print/labels", string(roh)); rec.Code != fall.status {
				t.Errorf("Status %d, erwartet %d: %s", rec.Code, fall.status, pdftest.ErsteBytes(rec.Body.Bytes(), 200))
			}
		})
	}
}
