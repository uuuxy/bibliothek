package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

	// body nil = Altclient ohne Body (vor Migration 088) — muss weiter abschliessen.
	erledigeMit := func(t *testing.T, body *string) (*httptest.ResponseRecorder, string) {
		t.Helper()
		var resID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, notiz, angefordert_von)
			VALUES ($1, '9c', 25, 'ab 15. September', $2) RETURNING id`, titelID, lehrerID).Scan(&resID); err != nil {
			t.Fatalf("Reservierung anlegen: %v", err)
		}
		var r io.Reader
		if body != nil {
			r = strings.NewReader(*body)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/reservierungen/klassensatz/"+resID+"/erledigen", r)
		req.SetPathValue("id", resID)
		rec := httptest.NewRecorder()
		srv.ErledigeKlassensatzReservierungHandler()(rec, req)
		return rec, resID
	}
	erledige := func(t *testing.T) *httptest.ResponseRecorder {
		rec, _ := erledigeMit(t, nil)
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

	// Migration 088: Die Antwort der Bibliothek landet in der Mail UND in der Zeile;
	// die Notiz der Lehrkraft steht in der Mail, damit die Antwort einen Bezug hat.
	// Ohne Body (Altclient) geht die Mail ohne Bibliotheks-Notiz, aber sie geht.
	t.Run("Notiz der Bibliothek steht in Mail und Zeile", func(t *testing.T) {
		var gesendet MailRequest
		alt := SendEmail
		SendEmail = func(m MailRequest) error { gesendet = m; return nil }
		t.Cleanup(func() { SendEmail = alt })

		body := `{"notiz":"24 von 30, der Rest ist bei der 8a"}`
		rec, resID := erledigeMit(t, &body)
		if got := mailStatusAusAntwort(t, rec); got != "versendet" {
			t.Fatalf("mail=%q, erwartet versendet", got)
		}
		for _, muss := range []string{"24 von 30, der Rest ist bei der 8a", "Ihre Notiz: ab 15. September", "Der Besuch der alten Dame"} {
			if !strings.Contains(gesendet.Body, muss) {
				t.Errorf("Mail ohne %q:\n%s", muss, gesendet.Body)
			}
		}
		var gespeichert string
		if err := pool.QueryRow(ctx, `SELECT erledigt_notiz FROM klassensatz_reservierungen WHERE id = $1`, resID).Scan(&gespeichert); err != nil {
			t.Fatalf("erledigt_notiz lesen: %v", err)
		}
		if gespeichert != "24 von 30, der Rest ist bei der 8a" {
			t.Fatalf("erledigt_notiz=%q", gespeichert)
		}

		gesendet = MailRequest{}
		rec, _ = erledigeMit(t, nil)
		if got := mailStatusAusAntwort(t, rec); got != "versendet" {
			t.Fatalf("ohne Body: mail=%q, erwartet versendet", got)
		}
		if strings.Contains(gesendet.Body, "Notiz der Bibliothek") {
			t.Errorf("ohne Body darf keine Bibliotheks-Notiz in der Mail stehen:\n%s", gesendet.Body)
		}
	})
}
