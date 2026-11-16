package system

// Der Absender der Mails aus Mein Portal: Der Testversand prüft ihn auf Wunsch, und eine
// unbrauchbare Adresse kommt nicht in die Datenbank.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/repository"
	"bibliothek/internal/smtptest"
	"bibliothek/internal/tuertest"
)

func TestTestmailSenden_PortalPrueftDenAbsenderFuerMeinPortal(t *testing.T) {
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true") // Testserver ohne STARTTLS
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)
	s, mock := mailTestTueren(t)
	erwarteMailKonfigMitPortal(mock, host, port, "bibliothek@schule.de", "portal@schule.de")

	rec := sendeTestmail(t, s, `{"to":"admin@schule.de","postfach":"portal"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	sitzung := <-sitzungen
	if !strings.Contains(sitzung.Nachricht, "From: portal@schule.de") {
		t.Errorf("Testversand ging nicht über den Absender für Mein Portal:\n%s", sitzung.Nachricht)
	}
}

func TestTestmailSenden_UnbekannterAbsenderIstEinEingabefehler(t *testing.T) {
	s, mock := mailTestTueren(t)

	rec := sendeTestmail(t, s, `{"to":"admin@schule.de","postfach":"haendler"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("rec.Code = %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("DB darf bei unbekanntem Absender nicht berührt werden: %v", err)
	}
}

func TestMailEinstellungenSchreiben_UnbrauchbarerPortalAbsenderKommtNichtInDieDatenbank(t *testing.T) {
	s, mock := mailTestTueren(t)
	repo := repository.NewMailEinstellungsAbfragen(mock)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings/mail", strings.NewReader(
		`{"smtp_host":"smtp.schule.de","smtp_port":"587","sender_email":"bibliothek@schule.de",`+
			`"portal":{"sender_email":"portal(at)schule.de","smtp_user":"","smtp_password":""}}`))
	rec := httptest.NewRecorder()
	s.MailEinstellungenSchreiben(repo)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("rec.Code = %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(tuertest.Fehlermeldung(t, rec), "zweiten Absenders") {
		t.Errorf("Meldung benennt das Feld nicht: %q", tuertest.Fehlermeldung(t, rec))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unbrauchbarer Absender wurde trotzdem geschrieben: %v", err)
	}
}
