package leser

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/xlsxtest"
)

// Durch die Vordertür: Eine Excel-Datei ohne Schüler-ID geht als Multipart-Upload an die
// Tür der Vorschau und kommt als Namensmodus-Vorschau zurück. Beweist die Verdrahtung
// Upload → Formaterkennung → Parser → Klassifizierung, nicht nur die Einzelteile.
func TestLUSDVorschau_XLSXOhneIDDurchDieVordertuer(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	geb := time.Date(2012, 3, 4, 0, 0, 0, 0, time.UTC)
	legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Max", nachname: "Mustermann", klasse: "5a", barcode: "UP-1", geb: &geb})

	xlsx := xlsxtest.Baue(t, map[string][][]any{
		"Klassenliste": {
			{"Klassenliste — Schuljahr 2026/27"},
			{"Nachname", "Vorname", "Klasse", "Geburtsdatum"},
			{"Mustermann", "Max", "6a", geb},
			{"Neu", "Kind", "5b", time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	teil, err := mw.CreateFormFile("csvFile", "lusd_export.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := teil.Write(xlsx); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/lusd/preview", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	s.LUSDVorschau().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var res lusd.Vorschau
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Modus != "name_geburtsdatum" || len(res.Klassenwechsel) != 1 || len(res.Neuzugaenge) != 1 {
		t.Fatalf("Vorschau aus Excel falsch: %+v", res)
	}
	// Vorschau schreibt nichts.
	if n := zaehle(t, pool, "nachname='Neu'"); n != 0 {
		t.Errorf("Vorschau hat geschrieben (n=%d)", n)
	}
}

// Die Datei, die Excel unter Windows als CSV speichert (Windows-1252), von der Vorschau bis in die
// Datenbank: Das Byte 0xFC ist kein gültiges UTF-8, und unumgewandelt endete die Übernahme mit 500.
func TestLUSD_CSVAusExcelInWindows1252BisInDieDatenbank(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	adminID := testdaten.LegeKontoAn(t, pool, "Ada", "Admin", "ada@kodierung.invalid", "admin", true)
	admin := &auth.Claims{KontoID: adminID, Rolle: auth.RolleAdmin}

	sende := func(pfad string) *httptest.ResponseRecorder {
		t.Helper()
		var koerper bytes.Buffer
		mw := multipart.NewWriter(&koerper)
		teil, err := mw.CreateFormFile("csvFile", "lusd_export.csv")
		if err != nil {
			t.Fatal(err)
		}
		// Das Nullzeichen steht wie der Umlaut für Bytes, die die Datenbank ablehnt.
		if _, err := teil.Write([]byte("Nachname;Vorname;Klasse;Geburtsdatum\r\nM\xfcl\x00ler;J\xfcrgen;5a;1.2.2012\r\n")); err != nil {
			t.Fatal(err)
		}
		if err := mw.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, pfad, &koerper)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		return ladeLUSDDateiHoch(s, req, admin)
	}

	vorschau := sende("/api/lusd/preview")
	if vorschau.Code != http.StatusOK {
		t.Fatalf("Vorschau = %d: %s", vorschau.Code, vorschau.Body.String())
	}
	if !strings.Contains(vorschau.Body.String(), "Jürgen") {
		t.Errorf("Die Vorschau nennt den Namen nicht richtig: %s", vorschau.Body.String())
	}
	if rec := sende("/api/lusd/import"); rec.Code != http.StatusOK {
		t.Fatalf("Übernahme = %d: %s", rec.Code, rec.Body.String())
	}
	if n := zaehle(t, pool, "vorname = 'Jürgen' AND nachname = 'Müller'"); n != 1 {
		t.Errorf("%d Zeilen mit dem richtigen Namen in der Datenbank, erwartet 1", n)
	}
}
