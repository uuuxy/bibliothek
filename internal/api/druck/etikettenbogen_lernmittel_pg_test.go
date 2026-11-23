package druck

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Das Druck-Center bietet das große Lernmittel-Etikett als weiteres Format an, mit dem Erzeuger
// des Lieferanten-Wegs und für ein Buch aus jedem Topf: Wer druckt, wählt die Größe. Ein Titel
// ohne Kennung der Lernmittel entsteht in den Testdaten als Buch der Schülerbücherei
// (repository.ExemplarTopfSQL).
func TestEtikettenbogen_GrossesLernmittelEtikett(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	lernmittel := testdaten.LegeTitelMitSignaturAn(t, pool, "Seydlitz Geographie", "LMF-Deutsch 5")
	testdaten.LegeExemplarAn(t, pool, lernmittel, "B-LMF-1", true, "")
	buecherei := testdaten.LegeTitelAn(t, pool, "Die Brüder Löwenherz")
	testdaten.LegeExemplarAn(t, pool, buecherei, "B-SB-1", true, "")

	drucke := func(items string) *httptest.ResponseRecorder {
		payload := `{"formatId": "lernmittel_gross", "startPosition": 1, "isQR": false, "items": [` + items + `]}`
		rec := httptest.NewRecorder()
		srv.Etikettenbogen()(rec, httptest.NewRequest(http.MethodPost, "/api/print/labels", strings.NewReader(payload)))
		return rec
	}
	const lmfEtikett = `{"BarcodeID": "B-LMF-1", "Titel": "Seydlitz Geographie", "Autor": ""}`
	const buechereiEtikett = `{"BarcodeID": "B-SB-1", "Titel": "Die Brüder Löwenherz", "Autor": ""}`

	t.Run("Lernmittel bekommt das große Etikett", func(t *testing.T) {
		rec := drucke(lmfEtikett)
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d — %s", rec.Code, rec.Body.String())
		}
		// Die Spalte der Ausleihtabelle gibt es nur auf dem großen Etikett.
		if texte := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n"); !strings.Contains(texte, "Name des Schülers") {
			t.Errorf("kein großes Etikett gedruckt:\n  %s", texte)
		}
	})

	t.Run("Buch der Schülerbücherei bekommt es ebenso", func(t *testing.T) {
		rec := drucke(lmfEtikett + "," + buechereiEtikett)
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
		}
		texte := strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
		for _, titel := range []string{"Seydlitz Geographie", "Die Brüder Löwenherz"} {
			if !strings.Contains(texte, titel) {
				t.Errorf("das große Etikett für %q fehlt:\n  %s", titel, texte)
			}
		}
	})
}

func TestBarcodeAufzaehlung(t *testing.T) {
	tests := []struct {
		barcodes []string
		want     string
	}{
		{[]string{"A"}, "A"},
		{[]string{"A", "B", "C"}, "A, B, C"},
		{[]string{"A", "B", "C", "D", "E"}, "A, B, C und 2 weitere"},
	}
	for _, tt := range tests {
		if got := barcodeAufzaehlung(tt.barcodes); got != tt.want {
			t.Errorf("barcodeAufzaehlung(%q) = %q, erwartet %q", tt.barcodes, got, tt.want)
		}
	}
}
