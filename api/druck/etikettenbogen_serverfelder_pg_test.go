package druck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Der Bogen des Druck-Centers trägt, was nur der Server kennt. Der Browser schickt Barcode,
// Titel und Autor (drucke in etiketten.svelte.js); Anschaffungsjahr und Signatur trägt
// buchetikett.ErgaenzeServerfelder nach. Fehlt dort ein Feld, antwortet die Tür trotzdem mit 200
// und einem gültigen PDF; erst der gedruckte Text zeigt es.
func TestEtikettenbogen_DrucktDieFelderDesServers(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Seydlitz Geographie", "LMF-Deutsch 5", 0)
	testdaten.LegeExemplarAn(t, pool, titelID, "B-ETI-1", true, "")
	// Ein festes Zugangsjahr: Das Jahr des Testlaufs stünde auch dann da, wenn das Feld fehlte
	// und der Erzeuger es selbst bildete.
	if _, err := pool.Exec(context.Background(),
		`UPDATE buecher_exemplare SET zugang_am = '2016-08-01' WHERE barcode_id = 'B-ETI-1'`); err != nil {
		t.Fatalf("Zugang setzen: %v", err)
	}

	// Genau die Nutzlast aus etiketten.svelte.js: nur BarcodeID, Titel und Autor. Wird hier ein
	// Feld ergänzt, gehört es nicht hierher, sondern in buchetikett.ErgaenzeServerfelder.
	payload := `{"formatId": "avery_3475", "startPosition": 1, "isQR": false,
		"items": [{"BarcodeID": "B-ETI-1", "Titel": "Seydlitz Geographie", "Autor": "Klaus Berger"}]}`
	rec := httptest.NewRecorder()
	srv.Etikettenbogen()(rec, httptest.NewRequest(http.MethodPost, "/api/print/labels", strings.NewReader(payload)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d — %s", rec.Code, rec.Body.String())
	}

	const want = "Ansch.J. 2016 · LMF-Deutsch 5"
	if texte := pdftest.Texte(t, rec.Body.Bytes()); !strings.Contains(strings.Join(texte, "\n"), want) {
		t.Errorf("%q fehlt auf dem Bogen. Gedruckt wurde:\n  %s", want, strings.Join(texte, "\n  "))
	}
}
