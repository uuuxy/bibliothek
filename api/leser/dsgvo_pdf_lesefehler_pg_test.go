package leser

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
)

// Die Auskunft als Blatt trägt den Kopf der Schule aus den Einstellungen. Sind sie nicht lesbar,
// entsteht kein Blatt, und im Protokoll steht keine erteilte Auskunft: Der Eintrag hält fest,
// dass die Person ihr Blatt bekommen hat.
func TestDSGVOAuskunftPDF_EinstellungenUnlesbar_KeinBlattKeinEintrag(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	admin := testdaten.LegeBearbeiterAn(t, pool)
	leserID := testdaten.LegeSchuelerAn(t, pool, "S-DSGVO-LESEFEHLER", "Lea", "07G1")

	rufe := func(datenbank db.PgxPoolIface) *httptest.ResponseRecorder {
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: datenbank}}}
		return tuertest.RufeAlsAdmin(t, admin, srv.DSGVOAuskunftPDF(), http.MethodGet,
			"/api/schueler/"+leserID+"/dsgvo-auskunft/pdf", "", map[string]string{"id": leserID})
	}
	eintraege := func() int {
		return pgtest.ZaehleZeilen(t, pool,
			`SELECT count(*) FROM audit_log WHERE aktion = 'dsgvo_auskunft' AND datensatz_id = $1::uuid`, leserID)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_log WHERE aktion = 'dsgvo_auskunft' AND datensatz_id = $1::uuid`, leserID)
	})

	rec := rufe(testdaten.PoolMitLesefehler{PgxPoolIface: pool, Muster: testdaten.SQLEinstellungen})
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500: Die Auskunft entsteht ohne die Angaben der Schule", rec.Code)
	}
	if bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Error("trotz Lesefehler kommt ein Blatt zurück")
	}
	if n := eintraege(); n != 0 {
		t.Errorf("%d Einträge über eine erteilte Auskunft, obwohl kein Blatt entstand", n)
	}

	// Gegenprobe: Am ungestörten Pool entsteht das Blatt, und die Auskunft steht im Protokoll.
	rec = rufe(pool)
	if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("ungestört: Status %d, erwartet 200 mit einem Blatt", rec.Code)
	}
	if n := eintraege(); n != 1 {
		t.Errorf("ungestört: %d Einträge über die erteilte Auskunft, erwartet 1", n)
	}
}
