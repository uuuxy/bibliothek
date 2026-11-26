package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/crypto"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
)

// Wechselt der Server, an den ein gespeichertes Passwort geht, verlangt die Tür das Passwort
// neu. Sonst stellte, wer Einstellungen ändern darf, den Server auf einen eigenen um und bekäme
// das gespeicherte Passwort beim nächsten Versand dorthin geschickt.
func TestMailEinstellungen_NeuerServerVerlangtDasPasswort(t *testing.T) {
	t.Setenv(crypto.SchluesselVariable, "mail-test-schluessel-32-zeichen!")
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := repository.NewMailEinstellungsAbfragen(pool)
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `UPDATE mail_settings_config
			SET smtp_host = 'localhost', smtp_port = '1025', smtp_user = '',
			    smtp_password_encrypted = NULL, sender_email = 'noreply@bibliothek-schule.de',
			    portal_absender_email = '', portal_smtp_benutzer = '',
			    portal_smtp_passwort_verschluesselt = NULL, portal_absender_fuer = '{portal}'
			WHERE id = 1`)
	})
	if err := repo.Schreibe(ctx, "smtp.schule.invalid", "465", "bibliothek", "geheim", "bibliothek@schule.invalid"); err != nil {
		t.Fatalf("Ausgangslage schreiben: %v", err)
	}
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	speichere := func(rumpf string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		srv.MailEinstellungenSchreiben(repo)(rec,
			httptest.NewRequest(http.MethodPut, "/api/admin/settings/mail", strings.NewReader(rumpf)))
		return rec
	}
	server := func() string {
		t.Helper()
		e, err := repo.Lade(ctx)
		if err != nil {
			t.Fatalf("Einstellung lesen: %v", err)
		}
		return e.SMTPHost
	}
	const kopf = `"smtp_port":"465","smtp_user":"bibliothek","sender_email":"bibliothek@schule.invalid"`

	t.Run("neuer Server ohne Passwort wird abgewiesen", func(t *testing.T) {
		rec := speichere(`{"smtp_host":"smtp.fremd.invalid",` + kopf + `,"smtp_password":""}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status %d — %s, erwartet 400", rec.Code, rec.Body.String())
		}
		if got := server(); got != "smtp.schule.invalid" {
			t.Errorf("gespeicherter Server %q, erwartet den alten", got)
		}
	})

	t.Run("derselbe Server speichert ohne Passwort", func(t *testing.T) {
		if rec := speichere(`{"smtp_host":" SMTP.schule.invalid ",` + kopf + `,"smtp_password":""}`); rec.Code != http.StatusOK {
			t.Errorf("Status %d — %s, erwartet 200", rec.Code, rec.Body.String())
		}
	})

	t.Run("der zweite Absender braucht sein Passwort ebenfalls neu", func(t *testing.T) {
		if err := repo.SchreibePortalAbsender(ctx, "portal@schule.invalid", "portal", "auch-geheim", nil); err != nil {
			t.Fatalf("zweiten Absender schreiben: %v", err)
		}
		rec := speichere(`{"smtp_host":"smtp.fremd.invalid",` + kopf + `,"smtp_password":"neu"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("ohne Passwort des zweiten Absenders: Status %d — %s, erwartet 400", rec.Code, rec.Body.String())
		}
		if got := server(); !strings.EqualFold(strings.TrimSpace(got), "smtp.schule.invalid") {
			t.Errorf("gespeicherter Server %q, erwartet den alten", got)
		}
		rec = speichere(`{"smtp_host":"smtp.neu.invalid",` + kopf + `,"smtp_password":"neu",
			"portal":{"sender_email":"portal@schule.invalid","smtp_user":"portal","smtp_password":"auch-neu"}}`)
		if rec.Code != http.StatusOK {
			t.Errorf("mit beiden Passwörtern: Status %d — %s, erwartet 200", rec.Code, rec.Body.String())
		}
		if got := server(); got != "smtp.neu.invalid" {
			t.Errorf("gespeicherter Server %q, erwartet den neuen", got)
		}
	})

	// Wer den zweiten Absender entfernt, nimmt sein Passwort mit: Es gehört zu keinem Absender
	// mehr, und ein späterer Serverwechsel verlangt es nicht neu.
	t.Run("ein entfernter zweiter Absender hält den Serverwechsel nicht auf", func(t *testing.T) {
		if err := repo.SchreibePortalAbsender(ctx, "portal@schule.invalid", "portal", "auch-geheim", nil); err != nil {
			t.Fatalf("zweiten Absender schreiben: %v", err)
		}
		rec := speichere(`{"smtp_host":"smtp.neu.invalid",` + kopf + `,"smtp_password":"",
			"portal":{"sender_email":"","smtp_user":"","smtp_password":""}}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("zweiten Absender entfernen: Status %d — %s, erwartet 200", rec.Code, rec.Body.String())
		}
		if e, err := repo.Lade(ctx); err != nil || len(e.PortalSMTPPasswortVerschluesselt) > 0 {
			t.Errorf("das Passwort des entfernten Absenders steht noch in der Datenbank (Lesefehler: %v)", err)
		}
		rec = speichere(`{"smtp_host":"smtp.anders.invalid",` + kopf + `,"smtp_password":"neu",
			"portal":{"sender_email":"","smtp_user":"","smtp_password":""}}`)
		if rec.Code != http.StatusOK {
			t.Errorf("Serverwechsel ohne zweiten Absender: Status %d — %s, erwartet 200", rec.Code, rec.Body.String())
		}
	})
}

// Dasselbe für das Webmail des Adressabgleichs: Eine neue Adresse verlangt das Passwort neu.
func TestAdressabgleichZugang_NeueAdresseVerlangtDasPasswort(t *testing.T) {
	t.Setenv(crypto.SchluesselVariable, "mail-test-schluessel-32-zeichen!")
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := repository.NewMailEinstellungsAbfragen(pool)
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `UPDATE mail_settings_config SET webmail_adresse = '', webmail_benutzer = '',
			webmail_passwort_verschluesselt = NULL WHERE id = 1`)
	})
	if err := repo.SchreibeWebmailZugang(ctx, "https://webmail.schule.invalid", "bibliothek", "geheim"); err != nil {
		t.Fatalf("Ausgangslage schreiben: %v", err)
	}
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	speichere := func(rumpf string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		srv.AdressabgleichZugangSchreiben(repo)(rec,
			httptest.NewRequest(http.MethodPut, "/api/admin/settings/adressabgleich", strings.NewReader(rumpf)))
		return rec
	}

	rec := speichere(`{"adresse":"https://webmail.fremd.invalid","benutzer":"bibliothek","passwort":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("neue Adresse ohne Passwort: Status %d — %s, erwartet 400", rec.Code, rec.Body.String())
	}
	e, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("Einstellung lesen: %v", err)
	}
	if e.WebmailAdresse != "https://webmail.schule.invalid" {
		t.Errorf("gespeicherte Adresse %q, erwartet die alte", e.WebmailAdresse)
	}
	if rec := speichere(`{"adresse":"https://webmail.schule.invalid/","benutzer":"bibliothek","passwort":""}`); rec.Code != http.StatusOK {
		t.Errorf("dieselbe Adresse ohne Passwort: Status %d — %s, erwartet 200", rec.Code, rec.Body.String())
	}
	if rec := speichere(`{"adresse":"https://webmail.neu.invalid","benutzer":"bibliothek","passwort":"neu"}`); rec.Code != http.StatusOK {
		t.Errorf("neue Adresse mit Passwort: Status %d — %s, erwartet 200", rec.Code, rec.Body.String())
	}
}
