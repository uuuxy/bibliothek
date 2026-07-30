package api

// mail_settings_test.go — Der Testversand ist der einzige Knopf, mit dem ein Admin
// die SMTP-Konfiguration prüfen kann. Sein Wert hängt vollständig daran, dass 200
// wirklich "versendet" heißt und ein Fehlschlag sagt, was schiefgelaufen ist.
//
// Warum hier ein echter (falscher) SMTP-Server läuft und nicht in der E2E-Suite:
// Die lokale Stack-Konfiguration zeigt auf den Schulserver — ein E2E-Test, der den
// Versandweg wirklich durchläuft, würde echte Mails verschicken. internal/smtptest
// nimmt die Nachricht an, ohne sie irgendwohin zu liefern, und lässt sich danach
// befragen, was tatsächlich über die Leitung ging.

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/smtptest"
	"bibliothek/mailservice"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

func mailTestServer(t *testing.T) (*Server, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	return &Server{DB: &db.Database{Pool: mock}}, mock
}

// erwarteMailKonfig legt die gespeicherte SMTP-Konfiguration für einen Aufruf fest.
func erwarteMailKonfig(mock pgxmock.PgxPoolIface, host, port, absender string) {
	mock.ExpectQuery(`SELECT smtp_host`).
		WillReturnRows(pgxmock.NewRows([]string{"smtp_host", "smtp_port", "smtp_user", "smtp_password_encrypted", "sender_email"}).
			AddRow(host, port, "", []byte{}, absender))
}

func postTestmail(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/mail/test", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.PostTestMailSettingsHandler()(rec, req)
	return rec
}

func fehlermeldung(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Antwort ist kein JSON-Fehler (%s): %v", rec.Body.String(), err)
	}
	return body.Error
}

// Der eigentliche Beweis: 200 bedeutet, dass eine Nachricht wirklich über SMTP
// hinausgegangen ist — mit genau einem Empfänger und ohne geschmuggelte Kopfzeile.
func TestPostTestMail_VersendetUeberSMTPUndMeldetDann200(t *testing.T) {
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)
	s, mock := mailTestServer(t)
	erwarteMailKonfig(mock, host, port, "bibliothek@schule.de")

	rec := postTestmail(t, s, `{"to":"admin@schule.de"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Konfiguration wurde nicht gelesen: %v", err)
	}

	sitzung := <-sitzungen
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "admin@schule.de" {
		t.Fatalf("Empfänger am Server = %v, want genau [admin@schule.de]", sitzung.Empfaenger)
	}
	for _, want := range []string{"From: bibliothek@schule.de", "To: admin@schule.de", "Subject: Test-E-Mail"} {
		if !strings.Contains(sitzung.Nachricht, want) {
			t.Errorf("Nachricht enthält %q nicht:\n%s", want, sitzung.Nachricht)
		}
	}
}

// Ein unerreichbarer Server ist der häufigste Fall (falscher Port, Tippfehler im
// Host) — und der, an dem sich zeigt, ob die Diagnose beim Admin ankommt: Als 500
// dampft apierrors sie auf "Ein interner Datenbankfehler ist aufgetreten" ein.
func TestPostTestMail_SMTPFehlerKommtLesbarBeimAdminAn(t *testing.T) {
	host, port := smtptest.GeschlossenerPort(t)
	s, mock := mailTestServer(t)
	erwarteMailKonfig(mock, host, port, "bibliothek@schule.de")

	rec := postTestmail(t, s, `{"to":"admin@schule.de"}`)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("erwartet 502 (Zielserver versagt), bekam %d: %s", rec.Code, rec.Body.String())
	}
	meldung := fehlermeldung(t, rec)
	if !strings.Contains(meldung, host+":"+port) {
		t.Errorf("Meldung nennt den Server nicht: %q", meldung)
	}
	if strings.Contains(meldung, "Datenbankfehler") {
		t.Errorf("SMTP-Fehler als Datenbankfehler ausgeliefert: %q", meldung)
	}
}

// Der Kern des Umbaus vom 30.07.2026: Der echte Versand (Mahnungen, Abgänger,
// Bestellungen) benutzt die in der Oberfläche gespeicherte Konfiguration — nicht die
// Umgebungsvariablen des Containers.
//
// Vorher las der Testversand die Datenbank und jeder echte Versand die Umgebung. Wer
// den SMTP-Server im Admin-Bereich umstellte, änderte am Versand nichts, und der
// Test-Knopf bestätigte eine Konfiguration, die kein Mahnlauf jemals benutzt hat.
// Deshalb steht hier die Umgebung absichtlich auf einem falschen Server: Landet die
// Mail trotzdem beim Fake-SMTP aus der Datenbank, ist die Frage beantwortet.
func TestSendEmail_BenutztDieGespeicherteKonfiguration(t *testing.T) {
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	t.Setenv("SMTP_HOST", "sollte-nicht-benutzt-werden.invalid")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_FROM", "umgebung@schule.de")

	_, mock := mailTestServer(t)
	erwarteMailKonfig(mock, host, port, "gespeichert@schule.de")
	BindeSMTPKonfigAnDatenbank(mock)
	t.Cleanup(func() {
		smtpKonfigLader = func() (mailservice.SMTPKonfig, error) { return mailservice.KonfigAusUmgebung(), nil }
	})

	err := SendEmail(MailRequest{To: "lehrerin@schule.de", Subject: "Mahnliste 5b", Body: "Anbei die Liste."})
	if err != nil {
		t.Fatalf("Versand fehlgeschlagen: %v", err)
	}

	sitzung := <-sitzungen
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "lehrerin@schule.de" {
		t.Fatalf("Empfänger am Server = %v, want [lehrerin@schule.de]", sitzung.Empfaenger)
	}
	if !strings.Contains(sitzung.Nachricht, "From: gespeichert@schule.de") {
		t.Errorf("Absender kommt nicht aus der gespeicherten Konfiguration:\n%s", sitzung.Nachricht)
	}
	if strings.Contains(sitzung.Nachricht, "umgebung@schule.de") {
		t.Errorf("Absender aus der Umgebung hat gewonnen:\n%s", sitzung.Nachricht)
	}
}

// Ohne hinterlegten Server ist der Versand keine Störung, sondern eine offene
// Einstellung: 503 mit Klartext, nicht 500 mit "Datenbankfehler".
func TestSendEmail_OhneKonfigurationMeldetOffeneEinstellung(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("SMTP_USER", "")

	err := SendEmail(MailRequest{To: "lehrerin@schule.de", Subject: "x", Body: "y"})
	if !errors.Is(err, mailservice.ErrMailNichtKonfiguriert) {
		t.Fatalf("Fehler = %v, want ErrMailNichtKonfiguriert", err)
	}
	if mailFehlerStatus(err) != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want 503", mailFehlerStatus(err))
	}
}

// Beim Speichern gilt für den Absender dasselbe wie für den Empfänger: Eine Adresse
// mit Tippfehler kam bisher in die Datenbank und fiel erst im Testversand auf — als
// 500, die der Admin als Datenbankfehler zu lesen bekam, obwohl sein eigenes Feld
// die Ursache war. Leer bleibt erlaubt (der Versand setzt seinen Standardabsender).
func TestUpdateMailSettings_UnbrauchbarerAbsenderKommtNichtInDieDatenbank(t *testing.T) {
	s, mock := mailTestServer(t)
	repo := repository.NewMailSettingsRepository(mock)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings/mail",
		strings.NewReader(`{"smtp_host":"smtp.schule.de","smtp_port":"587","sender_email":"bibliothek(at)schule.de"}`))
	rec := httptest.NewRecorder()
	s.UpdateMailSettingsHandler(repo)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(fehlermeldung(t, rec), "Absender") {
		t.Errorf("Meldung benennt das Feld nicht: %q", fehlermeldung(t, rec))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unbrauchbarer Absender wurde trotzdem geschrieben: %v", err)
	}
}

// Kein Empfänger, keine Verbindung: Der Fehler steht vor der SMTP-Sitzung, nicht
// erst darin — sonst hinge die Meldung von der Laune des Zielservers ab.
func TestPostTestMail_OhneEmpfaengerKeinVerbindungsversuch(t *testing.T) {
	for name, body := range map[string]string{
		"leer":            `{"to":""}`,
		"nur Leerzeichen": `{"to":"   "}`,
		"Feld fehlt":      `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			s, mock := mailTestServer(t)

			rec := postTestmail(t, s, body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("erwartet 400, bekam %d: %s", rec.Code, rec.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("DB darf ohne Empfänger nicht berührt werden: %v", err)
			}
		})
	}
}

// Eine unbrauchbare Adresse ist ein Eingabefehler des Admins (400) und keine
// Serverstörung (500) — nur so unterscheidet das Formular sie von einem SMTP-Ausfall.
// Die letzte Variante ist der Kopfzeilen-Schmuggel aus dem CodeQL-Fund.
func TestPostTestMail_UnbrauchbareAdresseIstEinEingabefehler(t *testing.T) {
	for name, adresse := range map[string]string{
		"ohne @":              `kein-empfaenger`,
		"nur Name":            `Admin`,
		"zwei Adressen":       `a@schule.de, b@schule.de`,
		"Bcc eingeschmuggelt": `admin@schule.de>\r\nBcc: mitleser@example.com`,
	} {
		t.Run(name, func(t *testing.T) {
			s, mock := mailTestServer(t)

			rec := postTestmail(t, s, `{"to":"`+adresse+`"}`)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("erwartet 400, bekam %d: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(fehlermeldung(t, rec), "Empfänger") {
				t.Errorf("Meldung benennt das Feld nicht: %q", fehlermeldung(t, rec))
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("DB darf für eine unbrauchbare Adresse nicht berührt werden: %v", err)
			}
		})
	}
}
