package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// Name und Geburtsdatum sind unter den Lesern ohne LUSD-ID eindeutig
// (unique_schueler_name_gebdatum). Anlegen und Wiederherstellen sagen bei einer Dublette,
// woran es liegt (409). Das Ändern der Stammdaten lief gegen denselben Index und antwortete
// als Serverfehler (500) mit dem allgemeinen Satz „Ein Eintrag mit diesen eindeutigen
// Eigenschaften existiert bereits.", ohne Name und Geburtsdatum zu nennen.
func TestLeserAendern_NameUndGeburtsdatumSchonVergeben(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"leser-dublette-testgeheimnis-32-bytes!!!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Dora', 'Dublette', 'leser-dublette@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(kontoID, "DUBL-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	var bea string
	if err := pool.QueryRow(ctx, `
		WITH anna AS (
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, geburtsdatum)
			VALUES ('S-DUBL-A', 'Anna', 'Muster', '07A', 2031, '2012-01-02')
		)
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, geburtsdatum)
		VALUES ('S-DUBL-B', 'Bea', 'Muster', '07A', 2031, '2012-01-02') RETURNING id`).Scan(&bea); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	aendere := func(rumpf string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+bea, strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, mitCSRF(req))
		return rec
	}
	vorname := func() string {
		var v string
		if err := pool.QueryRow(ctx, `SELECT vorname FROM leser WHERE id = $1`, bea).Scan(&v); err != nil {
			t.Fatalf("Vorname lesen: %v", err)
		}
		return v
	}

	rec := aendere(`{"vorname":"Anna"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status %d, erwartet 409 — %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "existiert bereits") {
		t.Errorf("die Antwort nennt den Grund nicht: %s", rec.Body.String())
	}
	if v := vorname(); v != "Bea" {
		t.Errorf("Vorname nach der Ablehnung = %q, erwartet unverändert Bea", v)
	}

	// Gegenprobe: Ein Name, den niemand trägt, wird gespeichert.
	if rec := aendere(`{"vorname":"Berta"}`); rec.Code != http.StatusOK {
		t.Fatalf("freier Name: Status %d — %s", rec.Code, rec.Body.String())
	}
	if v := vorname(); v != "Berta" {
		t.Errorf("Vorname = %q, erwartet Berta", v)
	}
}
