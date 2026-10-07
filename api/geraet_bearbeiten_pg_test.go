package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
	"bibliothek/sse"
)

// Ein defektes Gerät bleibt defekt, wenn jemand nur seine Stammdaten bearbeitet.
//
// Der Bearbeiten-Dialog (GeraeteVerwaltung.svelte) hat fünf Felder — modellname,
// barcode_id, seriennummer, zubehoer, zustand_notiz — und schickt genau diese. Das
// Defekt-Kennzeichen liegt auf einem eigenen Knopf und ist im Formular NICHT enthalten.
//
// Im Handler stand dazu `istAusleihbar := true`, wenn das Feld fehlt. Ein fehlendes
// Feld war damit kein "unverändert", sondern ein Wert: Wer bei einem als defekt
// markierten Gerät einen Tippfehler im Zubehör korrigierte, gab es damit still wieder
// zur Ausleihe frei. Auf dem Bildschirm stand "Gerät gespeichert".
func TestGeraetBearbeiten_HebtDefektNichtAuf(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	repo := repository.NewGeraeteRepository(pool)

	if _, err := pool.Exec(ctx, `DELETE FROM geraete WHERE barcode_id = 'G-DEFEKT'`); err != nil {
		t.Fatalf("aufräumen: %v", err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO geraete (modellname, barcode_id, seriennummer, zubehoer, ist_ausleihbar)
		VALUES ('Tablet 10', 'G-DEFEKT', 'SN-4711', 'Ladekabel', false)
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}

	// GENAU der Rumpf des Bearbeiten-Dialogs: fünf Felder, kein ist_ausleihbar.
	// Die Seriennummer wird MIT GEÄNDERT: Das Formular bietet das Feld an, der Handler
	// liess es bis zum 23.08.2026 unter den Tisch fallen — die Korrektur einer falschen
	// Seriennummer war folgenlos, mit "Gerät gespeichert" daneben.
	body := `{"modellname":"Tablet 10","barcode_id":"G-DEFEKT","seriennummer":"SN-9999",
	          "zubehoer":"Ladekabel und Hülle","zustand_notiz":""}`
	req := httptest.NewRequest(http.MethodPut, "/api/geraete/"+id, strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.UpdateGeraetHandler(repo)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT gab %d: %s", rec.Code, rec.Body.String())
	}

	var ausleihbar bool
	var zubehoer, seriennummer string
	if err := pool.QueryRow(ctx,
		`SELECT ist_ausleihbar, coalesce(zubehoer,''), coalesce(seriennummer,'') FROM geraete WHERE id = $1`, id).
		Scan(&ausleihbar, &zubehoer, &seriennummer); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}
	if zubehoer != "Ladekabel und Hülle" {
		t.Errorf("die gewollte Änderung kam nicht an: zubehoer = %q", zubehoer)
	}
	if seriennummer != "SN-9999" {
		t.Errorf("die Seriennummer wurde verworfen: %q — das Formular bietet das Feld an", seriennummer)
	}
	if ausleihbar {
		t.Error("das defekte Gerät ist wieder ausleihbar — ein fehlendes Feld wurde als " +
			"Wert gelesen statt als \"unverändert\"")
	}
}

// Und der Defekt-Knopf schaltet weiterhin, ohne die Stammdaten anzufassen: Er schickt
// die Seriennummer nicht mit, und nil heisst dort "nicht angefasst" — nicht "leeren".
func TestGeraetDefektKnopf_LaesstStammdatenStehen(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	repo := repository.NewGeraeteRepository(pool)

	if _, err := pool.Exec(ctx, `DELETE FROM geraete WHERE barcode_id = 'G-KNOPF'`); err != nil {
		t.Fatalf("aufräumen: %v", err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO geraete (modellname, barcode_id, seriennummer, zubehoer, ist_ausleihbar)
		VALUES ('Tablet 11', 'G-KNOPF', 'SN-0815', 'Ladekabel', true)
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}

	body := `{"modellname":"Tablet 11","zubehoer":"Ladekabel","zustand_notiz":"","ist_ausleihbar":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/geraete/"+id, strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.UpdateGeraetHandler(repo)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT gab %d: %s", rec.Code, rec.Body.String())
	}

	var ausleihbar bool
	var seriennummer string
	if err := pool.QueryRow(ctx,
		`SELECT ist_ausleihbar, coalesce(seriennummer,'') FROM geraete WHERE id = $1`, id).
		Scan(&ausleihbar, &seriennummer); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}
	if ausleihbar {
		t.Error("der Defekt-Knopf hat nicht geschaltet")
	}
	if seriennummer != "SN-0815" {
		t.Errorf("der Defekt-Knopf hat die Seriennummer verändert: %q", seriennummer)
	}
}

// Eine doppelte Seriennummer nennt die Seriennummer, beim Anlegen und beim Bearbeiten. Beim
// Anlegen stand dafür „Barcode ist bereits an ein anderes Gerät vergeben", beim Bearbeiten
// eine Störung. Der doppelte Barcode behält seine Meldung.
func TestGeraet_DoppelteSeriennummerNenntDieSeriennummer(t *testing.T) {
	pool := pgTestPool(t)
	ctx := t.Context()
	aufraeumenGeraete := func() { aufraeumen(t, pool, `DELETE FROM geraete WHERE barcode_id LIKE 'G-SNR-%'`) }
	aufraeumenGeraete()
	t.Cleanup(aufraeumenGeraete)

	authenticator, err := auth.NewAuthenticator(
		"geraet-seriennummer-testgeheimnis-32-b!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Gerda', 'Geraet', 'geraet-seriennummer@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	t.Cleanup(func() { aufraeumen(t, pool, `DELETE FROM benutzer WHERE email = 'geraet-seriennummer@example.org'`) })
	sitzung, err := authenticator.GenerateToken(kontoID, "GERAET-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()
	rufe := func(methode, pfad, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, mitCSRF(req))
		return rec
	}
	// abgelehnt prüft den Konflikt und welches Feld die Meldung nennt.
	abgelehnt := func(was string, rec *httptest.ResponseRecorder, nennt, nenntNicht string) {
		t.Helper()
		if rec.Code != http.StatusConflict {
			t.Errorf("%s: Status %d, erwartet 409 — %s", was, rec.Code, rec.Body.String())
			return
		}
		if !strings.Contains(rec.Body.String(), nennt) || strings.Contains(rec.Body.String(), nenntNicht) {
			t.Errorf("%s: die Meldung soll %q nennen und nicht %q — %s", was, nennt, nenntNicht, rec.Body.String())
		}
	}

	if rec := rufe(http.MethodPost, "/api/geraete",
		`{"modellname":"Tablet A","barcode_id":"G-SNR-A","seriennummer":"SN-DOPPELT"}`); rec.Code != http.StatusCreated {
		t.Fatalf("erstes Gerät anlegen: Status %d — %s", rec.Code, rec.Body.String())
	}
	abgelehnt("zweites Gerät mit derselben Seriennummer", rufe(http.MethodPost, "/api/geraete",
		`{"modellname":"Tablet B","barcode_id":"G-SNR-B","seriennummer":"SN-DOPPELT"}`), "Seriennummer", "Barcode")
	abgelehnt("zweites Gerät mit demselben Barcode", rufe(http.MethodPost, "/api/geraete",
		`{"modellname":"Tablet C","barcode_id":"G-SNR-A","seriennummer":"SN-ANDERE"}`), "Barcode", "Seriennummer")

	if rec := rufe(http.MethodPost, "/api/geraete",
		`{"modellname":"Tablet D","barcode_id":"G-SNR-D","seriennummer":"SN-FREI"}`); rec.Code != http.StatusCreated {
		t.Fatalf("viertes Gerät anlegen: Status %d — %s", rec.Code, rec.Body.String())
	}
	var idD string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM geraete WHERE barcode_id = 'G-SNR-D'`).Scan(&idD); err != nil {
		t.Fatalf("viertes Gerät lesen: %v", err)
	}
	abgelehnt("Seriennummer eines anderen Geräts beim Bearbeiten", rufe(http.MethodPut, "/api/geraete/"+idD,
		`{"modellname":"Tablet D","barcode_id":"G-SNR-D","seriennummer":"SN-DOPPELT","zubehoer":"","zustand_notiz":""}`),
		"Seriennummer", "Barcode")
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM geraete WHERE barcode_id = 'G-SNR-D' AND seriennummer = 'SN-FREI'`); n != 1 {
		t.Error("das abgelehnte Bearbeiten hat die Seriennummer trotzdem geändert")
	}
}

// Zwei Geräte ohne Seriennummer lassen sich beide bearbeiten. Der Dialog schickt das leere
// Feld mit; als leerer Text gespeichert, stieße das zweite Gerät an die Eindeutigkeit der
// Seriennummer.
func TestGeraetBearbeiten_LeereSeriennummerBleibtLeer(t *testing.T) {
	pool := pgTestPool(t)
	ctx := t.Context()
	srv := &Server{DB: &db.Database{Pool: pool}}
	repo := repository.NewGeraeteRepository(pool)
	aufraeumenGeraete := func() { aufraeumen(t, pool, `DELETE FROM geraete WHERE barcode_id LIKE 'G-OSN-%'`) }
	aufraeumenGeraete()
	t.Cleanup(aufraeumenGeraete)

	bearbeite := func(barcode string) *httptest.ResponseRecorder {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ('Tablet', $1) RETURNING id`,
			barcode).Scan(&id); err != nil {
			t.Fatalf("Gerät anlegen: %v", err)
		}
		// Der Rumpf des Bearbeiten-Dialogs für ein Gerät ohne Seriennummer.
		body := `{"modellname":"Tablet","barcode_id":"` + barcode + `","seriennummer":"","zubehoer":"Hülle","zustand_notiz":""}`
		req := httptest.NewRequest(http.MethodPut, "/api/geraete/"+id, strings.NewReader(body))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.UpdateGeraetHandler(repo)(rec, req)
		return rec
	}
	if rec := bearbeite("G-OSN-1"); rec.Code != http.StatusOK {
		t.Fatalf("erstes Gerät bearbeiten: Status %d — %s", rec.Code, rec.Body.String())
	}
	if rec := bearbeite("G-OSN-2"); rec.Code != http.StatusOK {
		t.Errorf("zweites Gerät ohne Seriennummer bearbeiten: Status %d — %s", rec.Code, rec.Body.String())
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM geraete WHERE barcode_id LIKE 'G-OSN-%' AND seriennummer IS NULL`); n != 2 {
		t.Errorf("%d der zwei Geräte tragen nach dem Bearbeiten keine Seriennummer (NULL), erwartet 2", n)
	}
}
