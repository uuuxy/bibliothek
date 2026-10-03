package api

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
)

// Was der LUSD-Upload ablehnt, kommt über die Route mit dem Status an, an dem die Oberfläche
// die Rückfrage zum Massenabgang (409) von einer Fehleingabe (400) unterscheidet. Eine
// Ablehnung schreibt nichts, auch keinen Protokolleintrag; erst der bestätigte Lauf tut beides.
func TestLusdUpload_AblehnungenUeberDieRoute(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ada', 'Admin', 'ada@upload.invalid', 'admin', true) RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	admin := &auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}

	// Zehn bestätigte Schüler, der Export nennt sechs: Vier Abgänger sind 40 % und liegen über
	// der Schwelle für die Rückfrage.
	export := "vorname,nachname,klasse,geburtsdatum\n"
	for i := 1; i <= 10; i++ {
		legeNmSchuelerAn(t, ctx, pool, nmSchueler{
			vorname: "Kind", nachname: fmt.Sprintf("Nummer%02d", i), klasse: "7a",
			barcode: fmt.Sprintf("UPA-%02d", i), geb: datum(2012, 1, i), bestaetigt: true,
		})
		if i <= 6 {
			export += fmt.Sprintf("Kind,Nummer%02d,7a,2012-01-%02d\n", i, i)
		}
	}

	sende := func(t *testing.T, pfad string, mitDatei bool, felder map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		var koerper bytes.Buffer
		mw := multipart.NewWriter(&koerper)
		if mitDatei {
			teil, err := mw.CreateFormFile("csvFile", "lusd_export.csv")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := teil.Write([]byte(export)); err != nil {
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
		return fahreLusdUpload(srv, req, admin)
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

	t.Run("kein Formular", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/lusd/import", strings.NewReader("keine Datei"))
		req.Header.Set("Content-Type", "text/plain")
		erwarte(t, fahreLusdUpload(srv, req, admin), http.StatusBadRequest, "error")
	})
	t.Run("Formular ohne Datei", func(t *testing.T) {
		erwarte(t, sende(t, "/api/lusd/import", false, map[string]string{"confirm_graduates": "true"}),
			http.StatusBadRequest, "CSV-Datei fehlt")
	})
	t.Run("die Vorschau liest die Auswahl nicht", func(t *testing.T) {
		erwarte(t, sende(t, "/api/lusd/preview", true, map[string]string{"umbenennungen": "{kaputt"}),
			http.StatusOK, `"graduates"`)
	})
	t.Run("Auswahl unlesbar", func(t *testing.T) {
		erwarte(t, sende(t, "/api/lusd/import", true, map[string]string{"umbenennungen": "{kaputt"}),
			http.StatusBadRequest, "Umbenennungs-Auswahl unlesbar")
	})
	t.Run("Zuordnung nicht vorgeschlagen", func(t *testing.T) {
		wahl := `[{"zeile":2,"schueler_id":"00000000-0000-0000-0000-000000000000"}]`
		erwarte(t, sende(t, "/api/lusd/import", true, map[string]string{"umbenennungen": wahl}),
			http.StatusBadRequest, "Zeile 2 ist nicht mehr g")
	})
	t.Run("Massenabgang ohne Bestätigung", func(t *testing.T) {
		erwarte(t, sende(t, "/api/lusd/import", true, nil), http.StatusConflict, "4 von 10 aktiven")
	})

	if n := zaehle(t, pool, "ist_abgaenger"); n != 0 {
		t.Fatalf("eine Ablehnung hat geschrieben: %d Abgänger", n)
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = 'LUSD_IMPORT'`); n != 0 {
		t.Fatalf("vor dem bestätigten Lauf stehen %d Einträge im Protokoll", n)
	}

	t.Run("Massenabgang mit Bestätigung", func(t *testing.T) {
		erwarte(t, sende(t, "/api/lusd/import", true, map[string]string{"confirm_graduates": "true"}),
			http.StatusOK, `"graduates"`)
		if n := zaehle(t, pool, "ist_abgaenger"); n != 4 {
			t.Errorf("%d Abgänger nach dem bestätigten Lauf, erwartet 4", n)
		}
		if n := zaehleZeilen(t, pool, `
			SELECT count(*) FROM audit_logs
			WHERE aktion = 'LUSD_IMPORT' AND admin_id = $1
			  AND details->>'abgaenger' = '4' AND details->>'massenabgang_bestaetigt' = 'true'`, adminID); n != 1 {
			t.Errorf("%d Protokolleinträge zum bestätigten Lauf, erwartet 1", n)
		}
	})
}

// fahreLusdUpload schickt die Anfrage je nach Pfad an die Vorschau oder den Import.
func fahreLusdUpload(srv *Server, req *http.Request, claims *auth.Claims) *httptest.ResponseRecorder {
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
	rec := httptest.NewRecorder()
	if req.URL.Path == "/api/lusd/import" {
		srv.PostLusdImportHandler().ServeHTTP(rec, req)
	} else {
		srv.PostLusdPreviewHandler().ServeHTTP(rec, req)
	}
	return rec
}
