package smtptest_test

import (
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/smtptest"
	"bibliothek/pkg/closeutil"
)

// lies, schreibe und waehle sprechen den Server auf Protokollebene an — dort, wo
// net/smtp zu viel abnimmt (unbekannte Befehle, RCPT ohne spitze Klammern, Abbruch
// mitten in DATA). Sie prüfen ihre Fehler, weil ein stiller Lesefehler den folgenden
// Vergleich gegen einen leeren String laufen ließe: der Test bliebe grün und der
// geprüfte Pfad wäre nie gelaufen.
func lies(t *testing.T, conn net.Conn, buf []byte) string {
	t.Helper()
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Antwort des Servers konnte nicht gelesen werden: %v", err)
	}
	return string(buf[:n])
}

func schreibe(t *testing.T, conn net.Conn, befehl string) {
	t.Helper()
	if _, err := conn.Write([]byte(befehl)); err != nil {
		t.Fatalf("Befehl %q konnte nicht gesendet werden: %v", strings.TrimSpace(befehl), err)
	}
}

// schliesseHart beendet die Verbindung absichtlich mitten im Dialog. Der Fehler wird
// hier bewusst nur protokolliert: das Schließen IST die geprüfte Handlung.
func schliesseHart(t *testing.T, conn net.Conn) {
	t.Helper()
	closeutil.LogClose(conn, "Testverbindung")
}

func TestStarte_Normal(t *testing.T) {
	host, port, ch := smtptest.Starte(t, smtptest.Normal)
	addr := host + ":" + port

	err := smtp.SendMail(addr, nil, "sender@test.de", []string{"empf@test.de"}, []byte("Subject: Test\r\n\r\nHallo")) // lgtm[go/mail-injection]
	if err != nil {
		t.Fatalf("SendMail fehler: %v", err)
	}

	sitzung := <-ch
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "empf@test.de" {
		t.Errorf("Unerwartete Empfaenger: %v", sitzung.Empfaenger)
	}
	if sitzung.Nachricht != "Subject: Test\r\n\r\nHallo\r\n" {
		t.Errorf("Unerwartete Nachricht: %q", sitzung.Nachricht)
	}
}

func TestStarte_LehntNachrichtAb(t *testing.T) {
	host, port, ch := smtptest.Starte(t, smtptest.LehntNachrichtAb)
	addr := host + ":" + port

	err := smtp.SendMail(addr, nil, "sender@test.de", []string{"empf@test.de"}, []byte("Subject: Test\r\n\r\nHallo")) // lgtm[go/mail-injection]
	if err == nil {
		t.Fatal("Erwartete einen Fehler bei LehntNachrichtAb, bekam nil")
	}
	if !strings.Contains(err.Error(), "Nachricht zu groß") {
		t.Errorf("Unerwarteter Fehler: %v", err)
	}

	sitzung := <-ch
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "empf@test.de" {
		t.Errorf("Unerwartete Empfaenger: %v", sitzung.Empfaenger)
	}
}

func TestStarte_BrichtVorQuitAb(t *testing.T) {
	host, port, ch := smtptest.Starte(t, smtptest.BrichtVorQuitAb)
	addr := host + ":" + port

	// BrichtVorQuitAb kappt die Verbindung, bevor QUIT quittiert wird — SendMail MUSS
	// hier scheitern. Wäre der Fehler nil, hätte der Server den Abbruch nicht ausgeführt
	// und die Sitzungsprüfung unten liefe gegen ein unbeabsichtigt normales Gespräch.
	err := smtp.SendMail(addr, nil, "sender@test.de", []string{"empf@test.de"}, []byte("Subject: Test\r\n\r\nHallo")) // lgtm[go/mail-injection]
	if err == nil {
		t.Fatal("Erwartete einen Fehler, weil der Server vor QUIT abbricht, bekam nil")
	}

	sitzung := <-ch
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "empf@test.de" {
		t.Errorf("Unerwartete Empfaenger: %v", sitzung.Empfaenger)
	}
	if sitzung.Nachricht != "Subject: Test\r\n\r\nHallo\r\n" {
		t.Errorf("Unerwartete Nachricht: %q", sitzung.Nachricht)
	}
}

func TestGeschlossenerPort(t *testing.T) {
	host, port := smtptest.GeschlossenerPort(t)
	addr := host + ":" + port

	err := smtp.SendMail(addr, nil, "sender@test.de", []string{"empf@test.de"}, []byte("Test")) // lgtm[go/mail-injection]
	if err == nil {
		t.Fatal("Erwartete einen Verbindungsfehler am geschlossenen Port, bekam nil")
	}
}

// Test edge cases and additional behavior for better coverage
func TestStarte_UnbekannterBefehl(t *testing.T) {
	host, port, ch := smtptest.Starte(t, smtptest.Normal)
	addr := host + ":" + port

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Verbindung fehlgeschlagen: %v", err)
	}
	defer schliesseHart(t, conn)

	buf := make([]byte, 1024)
	lies(t, conn, buf) // Begrüßung 220

	schreibe(t, conn, "UNBEKANNT\r\n")
	if antwort := lies(t, conn, buf); !strings.HasPrefix(antwort, "250 Ok") {
		t.Errorf("Erwartete 250 Ok, bekam: %s", antwort)
	}

	schreibe(t, conn, "QUIT\r\n")

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("Timeout warten auf Kanal")
	}
}

func TestAdresseAus_OhneKlammern(t *testing.T) {
	host, port, ch := smtptest.Starte(t, smtptest.Normal)
	addr := host + ":" + port

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Verbindung fehlgeschlagen: %v", err)
	}
	defer schliesseHart(t, conn)

	buf := make([]byte, 1024)
	lies(t, conn, buf) // Begrüßung 220

	// RCPT TO ohne spitze Klammern: der Helfer darf die Adresse dann NICHT schälen.
	schreibe(t, conn, "RCPT TO: raw@test.de\r\n")
	if antwort := lies(t, conn, buf); !strings.HasPrefix(antwort, "250 2.1.5 Ok") {
		t.Errorf("Erwartete 250 2.1.5 Ok, bekam: %s", antwort)
	}

	schreibe(t, conn, "QUIT\r\n")

	sitzung := <-ch
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "RCPT TO: raw@test.de" {
		t.Errorf("Unerwartete Empfänger (sollte ungeschält sein): %v", sitzung.Empfaenger)
	}
}

func TestStarte_VorzeitigerAbbruch(t *testing.T) {
	host, port, ch := smtptest.Starte(t, smtptest.Normal)
	addr := host + ":" + port

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Verbindung fehlgeschlagen: %v", err)
	}

	buf := make([]byte, 1024)
	lies(t, conn, buf) // Begrüßung 220

	// Verbindung sofort schließen — simuliert EOF beim Lesen.
	schliesseHart(t, conn)

	select {
	case sitzung := <-ch:
		if sitzung.Nachricht != "" {
			t.Errorf("Unerwartete Nachricht nach sofortigem Abbruch: %q", sitzung.Nachricht)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout warten auf Kanal")
	}
}

func TestLeseNachricht_Abbruch(t *testing.T) {
	host, port, ch := smtptest.Starte(t, smtptest.Normal)
	addr := host + ":" + port

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Verbindung fehlgeschlagen: %v", err)
	}

	buf := make([]byte, 1024)
	lies(t, conn, buf) // Begrüßung 220

	schreibe(t, conn, "DATA\r\n")
	lies(t, conn, buf) // 354

	// Text ohne abschließendes CRLF.CRLF, dann harter Abbruch: das \n am Ende ist
	// nötig, damit ReadString die Zeile überhaupt noch ausliefert.
	schreibe(t, conn, "Hallo\n")
	schliesseHart(t, conn)

	sitzung := <-ch
	if sitzung.Nachricht != "Hallo\n" {
		t.Errorf("Unerwartete Nachricht: %q", sitzung.Nachricht)
	}
}

// Der Helfer bedient die echten Mail-Tests nacheinander. Bliebe er nach der ersten
// Sitzung stehen, liefen alle folgenden Mail-Tests in einen Timeout statt in eine
// klare Aussage — deshalb wird der Mehrfachbetrieb hier ausdrücklich festgehalten.
func TestStarte_MehrereVerbindungen(t *testing.T) {
	host, port, ch := smtptest.Starte(t, smtptest.Normal)
	addr := host + ":" + port

	for i := 0; i < 3; i++ {
		err := smtp.SendMail(addr, nil, "sender@test.de", []string{"empf@test.de"}, []byte("Subject: Test\r\n\r\nHallo")) // lgtm[go/mail-injection]
		if err != nil {
			t.Fatalf("SendMail fehler: %v", err)
		}

		sitzung := <-ch
		if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "empf@test.de" {
			t.Errorf("Unerwartete Empfaenger: %v", sitzung.Empfaenger)
		}
	}
}
