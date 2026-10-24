package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Speichern der Mail-Einstellungen, über die Tür: Ein Rumpf ohne Server oder mit einem Feld,
// das die Maske nicht schickt, wird abgewiesen, und die gespeicherte Zeile bleibt. Ohne Server
// nähme jeder Versand still die Umgebung (ladeSMTPKonfig), und Port, Benutzer und Absender wären
// verloren. Der Rumpf der Maske speichert weiter, auch mit leerem Port.
func TestMailEinstellungenSchreiben_OhneServerBleibtDieEinstellung(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := repository.NewMailEinstellungsAbfragen(pool)
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `UPDATE mail_settings_config
			SET smtp_host = 'localhost', smtp_port = '1025', smtp_user = '',
			    smtp_password_encrypted = NULL, sender_email = 'noreply@bibliothek-schule.de'
			WHERE id = 1`)
	})
	if err := repo.Schreibe(ctx, "smtp.schule.invalid", "465", "bibliothek", "", "bibliothek@schule.invalid"); err != nil {
		t.Fatalf("Ausgangslage schreiben: %v", err)
	}
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	speichere := func(rumpf string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		srv.MailEinstellungenSchreiben(repo)(rec,
			httptest.NewRequest(http.MethodPut, "/api/admin/settings/mail", strings.NewReader(rumpf)))
		return rec
	}

	for _, rumpf := range []string{
		``,
		`{}`,
		`{"host":"smtp.neu.invalid","port":"587"}`,
		`{"smtp_host":"   ","smtp_port":"587","smtp_user":"","smtp_password":"","sender_email":""}`,
		`{"smtp_host":"smtp.neu.invalid","smtp_port":"587","smtp_tls":true}`,
	} {
		if rec := speichere(rumpf); rec.Code != http.StatusBadRequest {
			t.Errorf("Rumpf %s: Status %d — %s, erwartet 400", rumpf, rec.Code, rec.Body.String())
		}
		e, err := repo.Lade(ctx)
		if err != nil {
			t.Fatalf("Einstellung lesen: %v", err)
		}
		if e.SMTPHost != "smtp.schule.invalid" || e.SMTPPort != "465" || e.SMTPBenutzer != "bibliothek" ||
			e.AbsenderEmail != "bibliothek@schule.invalid" {
			t.Fatalf("Rumpf %s: gespeichert steht %q, %q, %q, %q — erwartet die Ausgangslage",
				rumpf, e.SMTPHost, e.SMTPPort, e.SMTPBenutzer, e.AbsenderEmail)
		}
	}

	rec := speichere(`{"smtp_host":" smtp.neu.invalid ","smtp_port":"","smtp_user":"","smtp_password":"","sender_email":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Rumpf der Maske: Status %d — %s, erwartet 200", rec.Code, rec.Body.String())
	}
	e, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("Einstellung lesen: %v", err)
	}
	if e.SMTPHost != "smtp.neu.invalid" || e.SMTPPort != "" {
		t.Errorf("gespeichert steht Server %q, Port %q — erwartet smtp.neu.invalid und leer", e.SMTPHost, e.SMTPPort)
	}
}
