package mahnwesen

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Der Mahnbrief trägt die Absenderzeile der Schule aus den Einstellungen, und sein Druck zählt
// die Mahnung. Sind die Einstellungen nicht lesbar, entsteht kein Brief, und die Mahnstufe
// bleibt stehen: Ein Brief ohne Absender wäre gezählt und nicht zu gebrauchen.
func TestMahnbriefDruck_EinstellungenUnlesbar_KeinBriefKeineMahnstufe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	sid := testdaten.LegeSchuelerAn(t, pool, "MB-LESEFEHLER", "Mia", "7a")
	ausleihe := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "Buch über der Frist", time.Now().AddDate(0, 0, -30))
	rumpf, err := json.Marshal(MahnbriefeAnfrage{AusleihIDs: []string{ausleihe}})
	if err != nil {
		t.Fatalf("Anfrage bauen: %v", err)
	}

	rufe := func(datenbank db.Pool) *httptest.ResponseRecorder {
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: datenbank}}}
		req := httptest.NewRequest(http.MethodPost, "/api/admin/mahnungen/bulk-print", bytes.NewReader(rumpf))
		rec := httptest.NewRecorder()
		srv.MahnbriefeDrucken()(rec, req)
		return rec
	}

	rec := rufe(testdaten.PoolMitLesefehler{Pool: pool, Muster: testdaten.SQLEinstellungen})
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500: Der Brief entsteht ohne die Absenderzeile der Schule", rec.Code)
	}
	if bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Error("trotz Lesefehler kommt ein Brief zurück")
	}
	if stufe, datum := testdaten.LadeMahnstufe(t, pool, ausleihe); stufe != 0 || datum != nil {
		t.Errorf("Mahnstufe %d, Mahndatum %v: Die Mahnung ist gezählt, obwohl kein Brief entstand", stufe, datum)
	}

	// Gegenprobe: Am ungestörten Pool entsteht der Brief, und die Mahnung ist gezählt.
	rec = rufe(pool)
	if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("ungestört: Status %d, erwartet 200 mit einem Brief", rec.Code)
	}
	if stufe, _ := testdaten.LadeMahnstufe(t, pool, ausleihe); stufe != 1 {
		t.Errorf("ungestört: Mahnstufe %d, erwartet 1", stufe)
	}
}
