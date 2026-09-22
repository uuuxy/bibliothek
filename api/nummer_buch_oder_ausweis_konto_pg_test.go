package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Migration 131: Eine Nummer ist entweder ein Buch oder ein Ausweis. Die drei Schüler-Türen
// (Anlegen, Akte, Wiederherstellen) übersetzen die Ablehnung des Wächters in eine Auskunft
// (409). Das Kollegiumskonto prüfte vor dem Schreiben nur die Leserzeilen
// (CheckBarcodeExists) und reichte die Ablehnung der Datenbank als 500 durch — die
// Bibliothek sah „interner Fehler“ statt „das ist die Nummer eines Buchs“
// (Rasterdurchgang 22.09.2026, Frage 5: stille Fehler).
func TestKollegiumskonto_NummerEinesBuchsIstEineAuskunft(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	admin := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a131", Rolle: auth.RoleAdmin}

	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp)
		VALUES ('Nummer-131-Band', 'Prüfer', 'Buch') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	buch := func(t *testing.T, barcode string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			VALUES ($1, $2, true)`, titelID, barcode); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
	}
	t.Cleanup(func() {
		auf := context.Background()
		if _, err := pool.Exec(auf, `DELETE FROM leser WHERE id IN
			(SELECT leser_id FROM benutzer WHERE email LIKE '%@nummer131.invalid')`); err != nil {
			t.Errorf("Leserzeilen aufräumen: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM benutzer WHERE email LIKE '%@nummer131.invalid'`); err != nil {
			t.Errorf("Konten aufräumen: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, titelID); err != nil {
			t.Errorf("Exemplare aufräumen: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Titel aufräumen: %v", err)
		}
	})

	// Ein Konto samt Leserzeile (Trigger trg_benutzer_hat_leserzeile, Migration 125).
	konto := func(t *testing.T, email string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ('Leh', 'Rer', $1, 'kollegium', true) RETURNING id`, email).Scan(&id); err != nil {
			t.Fatalf("Konto anlegen: %v", err)
		}
		return id
	}
	fahre := func(h http.Handler, methode, pfad, id, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(methode, pfad, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if id != "" {
			req.SetPathValue("id", id)
		}
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, admin))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	pruefe := func(t *testing.T, rec *httptest.ResponseRecorder, status int, teil string) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("Status %d, erwartet %d: %s", rec.Code, status, rec.Body.String())
		}
		if teil != "" && !strings.Contains(rec.Body.String(), teil) {
			t.Errorf("Meldung nennt den Grund nicht (%q fehlt): %s", teil, rec.Body.String())
		}
	}

	t.Run("Konto anlegen mit der Nummer eines Buchs", func(t *testing.T) {
		buch(t, "N131-A")
		rec := fahre(srv.CreateUserHandler(userRepo), http.MethodPost, "/api/benutzer", "",
			`{"barcode_id":"N131-A","vorname":"Anna","nachname":"Anlage","email":"a@nummer131.invalid","rolle":"kollegium"}`)
		pruefe(t, rec, http.StatusConflict, "Buch")
	})

	t.Run("Konto ändern auf die Nummer eines Buchs", func(t *testing.T) {
		buch(t, "N131-B")
		id := konto(t, "b@nummer131.invalid")
		rec := fahre(srv.UpdateUserHandler(userRepo), http.MethodPut, "/api/benutzer/"+id, id,
			`{"barcode_id":"N131-B","vorname":"Leh","nachname":"Rer","email":"b@nummer131.invalid","rolle":"kollegium","aktiv":true}`)
		pruefe(t, rec, http.StatusConflict, "Buch")
	})

	t.Run("Gegenprobe: eine freie Nummer geht durch", func(t *testing.T) {
		rec := fahre(srv.CreateUserHandler(userRepo), http.MethodPost, "/api/benutzer", "",
			`{"barcode_id":"N131-FREI","vorname":"Clara","nachname":"Frei","email":"c@nummer131.invalid","rolle":"kollegium"}`)
		pruefe(t, rec, http.StatusOK, "")
		var getragen string
		if err := pool.QueryRow(ctx, `SELECT l.barcode_id FROM leser l JOIN benutzer b ON b.leser_id = l.id
			WHERE b.email = 'c@nummer131.invalid'`).Scan(&getragen); err != nil || getragen != "N131-FREI" {
			t.Fatalf("die Leserzeile trägt %q (%v), erwartet N131-FREI", getragen, err)
		}
	})
}
