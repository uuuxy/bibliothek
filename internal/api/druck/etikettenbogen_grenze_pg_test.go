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
