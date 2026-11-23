package druck

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Das Druck-Center bietet das Signaturetikett als weiteres Format an. Die Signatur holt der Server
// vom Titel, der Browser schickt nur Barcode, Titel und Autor. Ein Exemplar, dessen Titel keine
// Signatur trägt, sperrt den Auftrag mit seinem Barcode, statt ein leeres Etikett zu drucken.
func TestEtikettenbogen_SignaturEtikett(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	mitSignatur := testdaten.LegeTitelMitSignaturAn(t, pool, "Tintenherz", "Sk / Fun")
	testdaten.LegeExemplarAn(t, pool, mitSignatur, "B-SIG-1", true, "")
	ohneSignatur := testdaten.LegeTitelAn(t, pool, "Die Brüder Löwenherz")
	testdaten.LegeExemplarAn(t, pool, ohneSignatur, "B-SIG-2", true, "")

	drucke := func(items string) *httptest.ResponseRecorder {
		payload := `{"formatId": "signatur_52", "startPosition": 1, "isQR": false, "items": [` + items + `]}`
		rec := httptest.NewRecorder()
		srv.Etikettenbogen()(rec, httptest.NewRequest(http.MethodPost, "/api/print/labels", strings.NewReader(payload)))
		return rec
	}
	const mitEtikett = `{"BarcodeID": "B-SIG-1", "Titel": "Tintenherz", "Autor": "Funke, Cornelia"}`
	const ohneEtikett = `{"BarcodeID": "B-SIG-2", "Titel": "Die Brüder Löwenherz", "Autor": ""}`

	t.Run("Regaladresse und Kürzel stehen auf dem Etikett", func(t *testing.T) {
		rec := drucke(mitEtikett)
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d — %s", rec.Code, rec.Body.String())
		}
		texte := pdftest.Texte(t, rec.Body.Bytes())
		slices.Sort(texte)
		if strings.Join(texte, "|") != "Fun|Sk" {
			t.Errorf("Texte auf dem Bogen %q, erwartet nur „Sk“ und „Fun“", texte)
		}
	})

	t.Run("Titel ohne Signatur sperrt den Auftrag", func(t *testing.T) {
		rec := drucke(mitEtikett + "," + ohneEtikett)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Status %d, erwartet 400", rec.Code)
		}
		if body := rec.Body.String(); !strings.Contains(body, "B-SIG-2") || strings.Contains(body, "B-SIG-1") {
			t.Errorf("die Meldung nennt nicht genau das Exemplar ohne Signatur: %s", body)
		}
	})
}
