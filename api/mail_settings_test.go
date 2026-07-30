package api

// mail_settings_test.go — Der Testversand ist der einzige Knopf, mit dem ein Admin
// die SMTP-Konfiguration prüfen kann. Sein Wert hängt vollständig daran, dass 200
// wirklich "versendet" heißt und ein Fehlschlag sagt, was schiefgelaufen ist.
//
// Warum hier ein echter (falscher) SMTP-Server läuft und nicht in der E2E-Suite:
// Die lokale Stack-Konfiguration zeigt auf den Schulserver — ein E2E-Test, der den
// Versandweg wirklich durchläuft, würde echte Mails verschicken. Der Fake-Server
// hier nimmt die Nachricht an, ohne sie irgendwohin zu liefern, und lässt sich
// gleichzeitig danach befragen, was tatsächlich über die Leitung ging.

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

// smtpSitzung hält fest, was der Server tatsächlich zu sehen bekam.
type smtpSitzung struct {
	empfaenger []string
	nachricht  string
}

// starteFakeSMTP nimmt genau eine Sitzung an und spricht das Minimum an SMTP, das
// net/smtp ohne Auth braucht. Bewusst ohne STARTTLS/AUTH in der EHLO-Antwort: Damit
// bleibt der Ablauf der schlichte Klartext-Versand, den ein Schulserver im LAN auch
// fährt, und der Test hängt nicht an einem Testzertifikat.
func starteFakeSMTP(t *testing.T) (host, port string, sitzungen <-chan smtpSitzung) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Fake-SMTP konnte nicht lauschen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() }) //nolint:errcheck // Test-Listener: ein Fehler beim Schließen ändert keine Zusicherung

	ch := make(chan smtpSitzung, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }() //nolint:errcheck // s.o.

		leser := bufio.NewReader(conn)
		// Schreibfehler brauchen keine Behandlung: Bricht die Verbindung weg, scheitert
		// der Versand — und genau darüber urteilt der Test ohnehin anhand der Antwort.
		schreibe := func(zeile string) {
			_, _ = conn.Write([]byte(zeile + "\r\n")) //nolint:errcheck
		}

		var sitzung smtpSitzung
		schreibe("220 fake ESMTP")
		for {
			zeile, err := leser.ReadString('\n')
			if err != nil {
				break
			}
			befehl := strings.ToUpper(strings.TrimSpace(zeile))
			switch {
			case strings.HasPrefix(befehl, "EHLO"), strings.HasPrefix(befehl, "HELO"):
				schreibe("250 fake")
			case strings.HasPrefix(befehl, "MAIL FROM"):
				schreibe("250 2.1.0 Ok")
			case strings.HasPrefix(befehl, "RCPT TO"):
				sitzung.empfaenger = append(sitzung.empfaenger, adresseAusBefehl(zeile))
				schreibe("250 2.1.5 Ok")
			case befehl == "DATA":
				schreibe("354 Ende mit <CR><LF>.<CR><LF>")
				var body strings.Builder
				for {
					datenzeile, err := leser.ReadString('\n')
					if err != nil {
						break
					}
					if strings.TrimRight(datenzeile, "\r\n") == "." {
						break
					}
					body.WriteString(datenzeile)
				}
				sitzung.nachricht = body.String()
				schreibe("250 2.0.0 Ok: queued")
			case befehl == "QUIT":
				schreibe("221 2.0.0 Bye")
				ch <- sitzung
				return
			default:
				schreibe("250 Ok")
			}
		}
		ch <- sitzung
	}()

	_, p, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("Port des Fake-SMTP unlesbar: %v", err)
	}
	return "127.0.0.1", p, ch
}

// adresseAusBefehl schält die Adresse aus "RCPT TO:<a@b.de>".
func adresseAusBefehl(zeile string) string {
	auf := strings.Index(zeile, "<")
	zu := strings.LastIndex(zeile, ">")
	if auf == -1 || zu <= auf {
		return strings.TrimSpace(zeile)
	}
	return zeile[auf+1 : zu]
}

// geschlossenerPort liefert einen Port, auf dem sicher niemand lauscht.
func geschlossenerPort(t *testing.T) (host, port string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Port konnte nicht belegt werden: %v", err)
	}
	_, p, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("Port unlesbar: %v", err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("Port konnte nicht freigegeben werden: %v", err)
	}
	return "127.0.0.1", p
}

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
	host, port, sitzungen := starteFakeSMTP(t)
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
	if len(sitzung.empfaenger) != 1 || sitzung.empfaenger[0] != "admin@schule.de" {
		t.Fatalf("Empfänger am Server = %v, want genau [admin@schule.de]", sitzung.empfaenger)
	}
	for _, want := range []string{"From: bibliothek@schule.de", "To: admin@schule.de", "Subject: Test-E-Mail"} {
		if !strings.Contains(sitzung.nachricht, want) {
			t.Errorf("Nachricht enthält %q nicht:\n%s", want, sitzung.nachricht)
		}
	}
}

// Ein unerreichbarer Server ist der häufigste Fall (falscher Port, Tippfehler im
// Host) — und der, an dem sich zeigt, ob die Diagnose beim Admin ankommt: Als 500
// dampft apierrors sie auf "Ein interner Datenbankfehler ist aufgetreten" ein.
func TestPostTestMail_SMTPFehlerKommtLesbarBeimAdminAn(t *testing.T) {
	host, port := geschlossenerPort(t)
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
