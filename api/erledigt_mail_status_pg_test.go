package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// Ausfallmatrix 20.08.2026: Die Erledigt-Handler (Klassensatz bereit, Anliegen
// abgehakt) verschicken ihre Benachrichtigung NACH dem Commit, best effort. Vorher
// antworteten sie 204 — ein Mailversand-Ausfall war nur eine Server-Logzeile, die
// Theke las "Erledigt — die Lehrkraft bekommt eine Mail", und niemand erfuhr, dass
// die Mail nie ankam. Jetzt trägt die Antwort den Mail-Status; die Oberfläche warnt
// bei "fehlgeschlagen".
//
// Echter PG-Test, weil beide Handler das Erledigen und den Mail-Empfänger in EINER
// CTE-Query erledigen — pgxmock sähe einen Schema-Drift nicht.

// stubSendEmail ersetzt den Versand für einen Test und stellt ihn danach wieder her.
func stubSendEmail(t *testing.T, fehler error) {
	t.Helper()
	alt := SendEmail
	SendEmail = func(MailRequest) error { return fehler }
	t.Cleanup(func() { SendEmail = alt })
}

// mailStatusAusAntwort erwartet 200 und liest das "mail"-Feld.
func mailStatusAusAntwort(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200 — Antwort: %s", rec.Code, rec.Body.String())
	}
	var antwort map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort kein JSON: %v — %s", err, rec.Body.String())
	}
	return antwort["mail"]
}

func TestErledigeAnliegen_MailAusfallStehtInDerAntwort(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var lehrerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Lisa', 'Lehrkraft', 'anliegen-mailstatus@test.invalid', 'mitarbeiter', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&lehrerID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}

	erledige := func(t *testing.T, anliegenID string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/anliegen/"+anliegenID+"/erledigen",
			strings.NewReader(`{"notiz":"besorgt"}`))
		req.SetPathValue("id", anliegenID)
		rec := httptest.NewRecorder()
		srv.ErledigeAnliegenHandler()(rec, req)
		return rec
	}

	neuesAnliegen := func(t *testing.T) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO lehrer_anliegen (art, titel_text, angefordert_von)
			VALUES ('wunsch', 'Faust II', $1) RETURNING id`, lehrerID).Scan(&id); err != nil {
			t.Fatalf("Anliegen anlegen: %v", err)
		}
		return id
	}

	t.Run("Versand kaputt → fehlgeschlagen", func(t *testing.T) {
		stubSendEmail(t, errors.New("smtp down"))
		if got := mailStatusAusAntwort(t, erledige(t, neuesAnliegen(t))); got != "fehlgeschlagen" {
			t.Fatalf("mail=%q, erwartet fehlgeschlagen", got)
		}
	})

	t.Run("Versand ok → versendet", func(t *testing.T) {
		stubSendEmail(t, nil)
		if got := mailStatusAusAntwort(t, erledige(t, neuesAnliegen(t))); got != "versendet" {
			t.Fatalf("mail=%q, erwartet versendet", got)
		}
	})

	t.Run("ohne Konto → keine_adresse, Versand wird nicht versucht", func(t *testing.T) {
		stubSendEmail(t, errors.New("darf nicht aufgerufen werden"))
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO lehrer_anliegen (art, titel_text)
			VALUES ('meldung', 'Rücken kaputt') RETURNING id`).Scan(&id); err != nil {
			t.Fatalf("Anliegen ohne Konto anlegen: %v", err)
		}
		if got := mailStatusAusAntwort(t, erledige(t, id)); got != "keine_adresse" {
			t.Fatalf("mail=%q, erwartet keine_adresse", got)
		}
	})
}

func TestErledigeKlassensatz_MailAusfallStehtInDerAntwort(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var lehrerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Karl', 'Klassensatz', 'klassensatz-mailstatus@test.invalid', 'mitarbeiter', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&lehrerID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel) VALUES ('Der Besuch der alten Dame') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	erledige := func(t *testing.T) *httptest.ResponseRecorder {
		t.Helper()
		var resID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, angefordert_von)
			VALUES ($1, '9c', 25, $2) RETURNING id`, titelID, lehrerID).Scan(&resID); err != nil {
			t.Fatalf("Reservierung anlegen: %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/reservierungen/klassensatz/"+resID+"/erledigen", nil)
		req.SetPathValue("id", resID)
		rec := httptest.NewRecorder()
		srv.ErledigeKlassensatzReservierungHandler()(rec, req)
		return rec
	}

	t.Run("Versand kaputt → fehlgeschlagen", func(t *testing.T) {
		stubSendEmail(t, errors.New("smtp down"))
		if got := mailStatusAusAntwort(t, erledige(t)); got != "fehlgeschlagen" {
			t.Fatalf("mail=%q, erwartet fehlgeschlagen", got)
		}
	})

	t.Run("Versand ok → versendet", func(t *testing.T) {
		stubSendEmail(t, nil)
		if got := mailStatusAusAntwort(t, erledige(t)); got != "versendet" {
			t.Fatalf("mail=%q, erwartet versendet", got)
		}
	})
}
