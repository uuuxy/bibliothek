package schaden

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Elternbrief und Ersatzforderung tragen den Kopf der Schule, die Zahlstelle und die
// Bankverbindung aus den Einstellungen. Sind sie nicht lesbar, liefert das Lesen die Vorgaben
// des Programms. Ein Blatt daraus nennte keine Schule und die Bankverbindung der Vorgabe statt
// der eingetragenen; beide Türen müssen den Lesefehler deshalb als Fehler behandeln.

func TestElternbrief_EinstellungenUnlesbar_KeinBriefKeinVermerk(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	sid := testdaten.LegeSchuelerAn(t, pool, "S-BRIEF-LESEFEHLER", "Elternbriefkind", "08G2")
	testdaten.LegeOffenenSchadenAn(t, pool, sid, "BRIEF-LF")
	var fall string
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM schadensfaelle WHERE schueler_id = $1`, sid).Scan(&fall); err != nil {
		t.Fatalf("Schadensfall lesen: %v", err)
	}
	erzeugt := func() int {
		return pgtest.ZaehleZeilen(t, pool,
			`SELECT count(*) FROM schadensfaelle WHERE id = $1 AND elternbrief_generiert`, fall)
	}

	rufe := func(datenbank db.PgxPoolIface) *httptest.ResponseRecorder {
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: datenbank}}}
		req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle/"+fall+"/pdf", nil)
		req.SetPathValue("id", fall)
		rec := httptest.NewRecorder()
		srv.GenerateDamagePDFHandler().ServeHTTP(rec, req)
		return rec
	}

	rec := rufe(testdaten.PoolMitLesefehler{PgxPoolIface: pool, Muster: testdaten.SQLEinstellungen})
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500: Der Brief entsteht mit den Vorgaben statt mit den Angaben der Schule", rec.Code)
	}
	if bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Error("trotz Lesefehler kommt ein Brief zurück")
	}
	if erzeugt() != 0 {
		t.Error("am Schadensfall steht „Elternbrief erzeugt“, obwohl kein Brief entstand")
	}

	// Gegenprobe: Am ungestörten Pool entsteht der Brief und der Vermerk.
	rec = rufe(pool)
	if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("ungestört: Status %d, erwartet 200 mit einem Brief", rec.Code)
	}
	if erzeugt() != 1 {
		t.Error("ungestört: am Schadensfall fehlt „Elternbrief erzeugt“")
	}
}

func TestErsatzforderung_EinstellungenUnlesbar_KeinBlatt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	sid := testdaten.LegeSchuelerAn(t, pool, "S-RECH-LESEFEHLER", "Rechnungskind", "08G2")
	testdaten.LegeOffenenSchadenAn(t, pool, sid, "RECH-LF")

	rufe := func(datenbank db.PgxPoolIface) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/print/rechnung/"+sid, nil)
		req.SetPathValue("schueler_id", sid)
		rec := httptest.NewRecorder()
		(&Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: datenbank}}}).PrintRechnungHandler()(rec, req)
		return rec
	}

	rec := rufe(testdaten.PoolMitLesefehler{PgxPoolIface: pool, Muster: testdaten.SQLEinstellungen})
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
