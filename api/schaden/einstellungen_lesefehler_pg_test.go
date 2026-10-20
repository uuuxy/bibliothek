package schaden

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/testweg"
)

// Die Ersatzforderung trägt den Kopf der Schule, die Zahlstelle und die Bankverbindung aus den
// Einstellungen. Sind sie nicht lesbar, liefert das Lesen die Vorgaben des Programms. Ein Blatt
// daraus nennte keine Schule und die Bankverbindung der Vorgabe statt der eingetragenen; die Tür
// muss den Lesefehler deshalb als Fehler behandeln.

func TestErsatzforderung_EinstellungenUnlesbar_KeinBlatt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	sid := testdaten.LegeSchuelerAn(t, pool, "S-RECH-LESEFEHLER", "Rechnungskind", "08G2")
	testdaten.LegeOffenenSchadenAn(t, pool, sid, "RECH-LF")

	rufe := func(datenbank db.Pool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/print/rechnung/"+sid, nil)
		req.SetPathValue("schueler_id", sid)
		rec := httptest.NewRecorder()
		(&Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: datenbank}}}).RechnungPDF()(rec, req)
		return rec
	}

	rec := rufe(testweg.PoolMitLesefehler{Pool: pool, Muster: testweg.SQLEinstellungen})
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500: Die Ersatzforderung entsteht mit den Vorgaben statt mit den Angaben der Schule", rec.Code)
	}
	if bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Error("trotz Lesefehler kommt ein Blatt zurück")
	}

	// Gegenprobe: Am ungestörten Pool entsteht das Blatt.
	if rec := rufe(pool); rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("ungestört: Status %d, erwartet 200 mit einem Blatt", rec.Code)
	}
}
