package leser

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
)

// Was der LUSD-Upload ablehnt, kommt über die Route mit dem Status an, an dem die Oberfläche
// die Rückfrage zum Massenabgang (409) von einer Fehleingabe (400) unterscheidet. Eine
// Ablehnung schreibt nichts, auch keinen Protokolleintrag; erst der bestätigte Lauf tut beides.
func TestLUSDDatei_AblehnungenUeberDieRoute(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ada', 'Admin', 'ada@upload.invalid', 'admin', true) RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	admin := &auth.Claims{KontoID: adminID, Rolle: auth.RolleAdmin}

	// Zehn bestätigte Schüler, der Export nennt sechs: Vier Abgänger sind 40 % und liegen über
	// der Schwelle für die Rückfrage.
	export := "vorname,nachname,klasse,geburtsdatum\n"
	for i := 1; i <= 10; i++ {
		legeNmSchuelerAn(t, ctx, pool, nmSchueler{
			vorname: "Kind", nachname: fmt.Sprintf("Nummer%02d", i), klasse: "7a",
			barcode: fmt.Sprintf("UPA-%02d", i), geb: testdaten.Datum(2012, 1, i), bestaetigt: true,
		})
		if i <= 6 {
			export += fmt.Sprintf("Kind,Nummer%02d,7a,2012-01-%02d\n", i, i)
		}
	}

	// sendeDatei schickt das Formular mit dieser Datei; ohne Inhalt ohne Datei.
	sendeDatei := func(t *testing.T, pfad string, inhalt []byte, felder map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		var koerper bytes.Buffer
		mw := multipart.NewWriter(&koerper)
		if inhalt != nil {
			teil, err := mw.CreateFormFile("csvFile", "lusd_export.csv")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := teil.Write(inhalt); err != nil {
				t.Fatal(err)
			}
		}
		for k, v := range felder {
			if err := mw.WriteField(k, v); err != nil {
				t.Fatal(err)
			}
		}
		if err := mw.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, pfad, &koerper)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		return ladeLUSDDateiHoch(srv, req, admin)
	}
	sende := func(t *testing.T, pfad string, mitDatei bool, felder map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		if !mitDatei {
			return sendeDatei(t, pfad, nil, felder)
		}
		return sendeDatei(t, pfad, []byte(export), felder)
	}
	erwarte := func(t *testing.T, rec *httptest.ResponseRecorder, status int, stueck string) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("Status %d, erwartet %d: %s", rec.Code, status, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), stueck) {
			t.Errorf("Antwort nennt %q nicht: %s", stueck, rec.Body.String())
		}
	}
	erwarteMeldung := func(t *testing.T, rec *httptest.ResponseRecorder, status int, meldung string) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("Status %d, erwartet %d: %s", rec.Code, status, rec.Body.String())
		}
		if got := tuertest.Fehlermeldung(t, rec); got != meldung {
			t.Errorf("Meldung %q, erwartet %q", got, meldung)
		}
	}

	t.Run("kein Formular", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/lusd/import", strings.NewReader("keine Datei"))
		req.Header.Set("Content-Type", "text/plain")
		erwarte(t, ladeLUSDDateiHoch(srv, req, admin), http.StatusBadRequest, "error")
	})
	t.Run("Formular ohne Datei", func(t *testing.T) {
		erwarte(t, sende(t, "/api/lusd/import", false, map[string]string{"confirm_graduates": "true"}),
			http.StatusBadRequest, "Bitte die Exportdatei aus der LUSD auswählen")
	})
	t.Run("die Vorschau liest die Auswahl nicht", func(t *testing.T) {
		erwarte(t, sende(t, "/api/lusd/preview", true, map[string]string{"umbenennungen": "{kaputt"}),
			http.StatusOK, `"graduates"`)
	})
	t.Run("Pflichtspalte fehlt", func(t *testing.T) {
		erwarteMeldung(t, sendeDatei(t, "/api/lusd/preview", []byte("vorname,nachname,geburtsdatum\nKind,Nummer01,2012-01-01\n"), nil),
			http.StatusBadRequest, "In der Kopfzeile fehlt die Pflichtspalte „klasse“. Ist das die Exportdatei aus der LUSD?")
	})
	t.Run("altes Excel", func(t *testing.T) {
		erwarteMeldung(t, sendeDatei(t, "/api/lusd/preview", []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, nil),
			http.StatusBadRequest, "Die Datei ist ein altes Excel-Binärformat (.xls). "+
				"Bitte in Excel „Speichern unter“ → .xlsx oder CSV wählen und erneut hochladen.")
	})
	t.Run("Auswahl unlesbar", func(t *testing.T) {
		erwarteMeldung(t, sende(t, "/api/lusd/import", true, map[string]string{"umbenennungen": "{kaputt"}),
			http.StatusBadRequest, "Die Auswahl der Umbenennungen ließ sich nicht lesen. Bitte die Vorschau neu laden.")
	})
	t.Run("Zuordnung nicht vorgeschlagen", func(t *testing.T) {
		wahl := `[{"zeile":2,"schueler_id":"00000000-0000-0000-0000-000000000000"}]`
		erwarteMeldung(t, sende(t, "/api/lusd/import", true, map[string]string{"umbenennungen": wahl}),
			http.StatusBadRequest, "Die Zuordnung für Zeile 2 ist nicht mehr gültig. Bitte die Vorschau neu laden.")
	})
	t.Run("Massenabgang ohne Bestätigung", func(t *testing.T) {
		erwarteMeldung(t, sende(t, "/api/lusd/import", true, nil), http.StatusConflict,
			"4 von 10 aktiven Schülern würden zu Abgängern (Schwelle: 30%). Bitte die Datei prüfen. Ist der "+
				"Massenabgang beabsichtigt (Schuljahreswechsel), den Import mit Bestätigung wiederholen.")
	})

	if n := zaehle(t, pool, "ist_abgaenger"); n != 0 {
		t.Fatalf("eine Ablehnung hat geschrieben: %d Abgänger", n)
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = 'LUSD_IMPORT'`); n != 0 {
		t.Fatalf("vor dem bestätigten Lauf stehen %d Einträge im Protokoll", n)
	}

	t.Run("Massenabgang mit Bestätigung", func(t *testing.T) {
		erwarte(t, sende(t, "/api/lusd/import", true, map[string]string{"confirm_graduates": "true"}),
			http.StatusOK, `"graduates"`)
		if n := zaehle(t, pool, "ist_abgaenger"); n != 4 {
			t.Errorf("%d Abgänger nach dem bestätigten Lauf, erwartet 4", n)
		}
		if n := pgtest.ZaehleZeilen(t, pool, `
			SELECT count(*) FROM audit_logs
			WHERE aktion = 'LUSD_IMPORT' AND admin_id = $1
			  AND details->>'abgaenger' = '4' AND details->>'massenabgang_bestaetigt' = 'true'`, adminID); n != 1 {
			t.Errorf("%d Protokolleinträge zum bestätigten Lauf, erwartet 1", n)
		}
	})
}

// ladeLUSDDateiHoch schickt die Anfrage je nach Pfad an die Vorschau oder den Import.
func ladeLUSDDateiHoch(srv *Tueren, req *http.Request, claims *auth.Claims) *httptest.ResponseRecorder {
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, claims))
	rec := httptest.NewRecorder()
	if req.URL.Path == "/api/lusd/import" {
		srv.LUSDImport().ServeHTTP(rec, req)
	} else {
		srv.LUSDVorschau().ServeHTTP(rec, req)
	}
	return rec
}
