package mailservice

import (
	"errors"
	"strings"
	"testing"

	"bibliothek/internal/smtptest"
)

func testKonfig(host, port string) SMTPKonfig {
	return SMTPKonfig{Host: host, Port: port, Absender: "bibliothek@schule.de"}
}

// Der Normalfall — und zugleich der Beleg, dass genau ein Empfänger im Envelope steht.
func TestVersendeUeberSMTPStelltZu(t *testing.T) {
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	err := VersendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n"))
	if err != nil {
		t.Fatalf("Versand fehlgeschlagen: %v", err)
	}

	sitzung := <-sitzungen
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "lehrerin@schule.de" {
		t.Fatalf("Empfänger = %v, want [lehrerin@schule.de]", sitzung.Empfaenger)
	}
	if !strings.Contains(sitzung.Nachricht, "Inhalt") {
		t.Errorf("Nachricht kam nicht an: %q", sitzung.Nachricht)
	}
}

// Der teure Fall: Der Server nimmt die Nachricht an (250 nach dem Punkt) und kappt
// die Verbindung, ohne das QUIT zu beantworten. Das ist ZUGESTELLT.
//
// Als Fehlschlag gemeldet, wäre der Schaden real: Der Mahnlauf zählte eine bereits
// zugestellte Mahnung als „nicht versendet", und der nächste Lauf schickte sie ein
// zweites Mal an dieselbe Klassenleitung — bei einer Mahnung an Eltern kein
// kosmetischer Fehler. Vorher gab der Code genau hier einen Fehler zurück, weil er
// das Ergebnis von QUIT durchreichte.
func TestVersendeUeberSMTPWertetAbbruchNachAnnahmeAlsZugestellt(t *testing.T) {
	host, port, sitzungen := smtptest.Starte(t, smtptest.BrichtVorQuitAb)

	err := VersendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n"))
	if err != nil {
		t.Fatalf("angenommene Nachricht wurde als Fehlschlag gemeldet: %v", err)
	}

	sitzung := <-sitzungen
	if !strings.Contains(sitzung.Nachricht, "Inhalt") {
		t.Errorf("Server hat die Nachricht nicht vollständig gesehen: %q", sitzung.Nachricht)
	}
}

// Die Gegenprobe: Weist der Server die Nachricht ab, ist das ein Fehlschlag — mit
// Marker (→ HTTP 502) und der Serverantwort im Klartext.
func TestVersendeUeberSMTPMeldetAbweisung(t *testing.T) {
	host, port, _ := smtptest.Starte(t, smtptest.LehntNachrichtAb)

	err := VersendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n"))
	if err == nil {
		t.Fatal("abgewiesene Nachricht wurde als versendet gemeldet")
	}
	if !errors.Is(err, ErrSMTPVersand) {
		t.Errorf("Fehler trägt den Marker nicht (→ falscher HTTP-Status): %v", err)
	}
	if !strings.Contains(err.Error(), "552") {
		t.Errorf("Antwort des Servers fehlt in der Meldung: %v", err)
	}
}

// Unerreichbarer Server: die häufigste Fehlkonfiguration überhaupt.
func TestVersendeUeberSMTPMeldetUnerreichbarenServer(t *testing.T) {
	host, port := smtptest.GeschlossenerPort(t)

	err := VersendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n"))
	if !errors.Is(err, ErrSMTPVersand) {
		t.Fatalf("Fehler = %v, want ErrSMTPVersand", err)
	}
	if !strings.Contains(err.Error(), host+":"+port) {
		t.Errorf("Meldung nennt den Server nicht: %v", err)
	}
}

// Ein Zeilenumbruch in einer Envelope-Adresse schmuggelt SMTP-Befehle in die
// Sitzung. net/smtp.SendMail prüfte das mit; auf dem eigenen Weg muss die Hürde
// ausdrücklich stehen — und zwar VOR dem Verbindungsaufbau.
func TestVersendeUeberSMTPWeistUmbruchImEnvelopeAb(t *testing.T) {
	host, port := smtptest.GeschlossenerPort(t) // wird nie erreicht

	faelle := map[string]struct{ absender, empfaenger string }{
		"Umbruch im Absender":  {"bib@schule.de\r\nRCPT TO:<mitleser@example.com>", "lehrerin@schule.de"},
		"Umbruch im Empfänger": {"bib@schule.de", "lehrerin@schule.de\r\nRCPT TO:<mitleser@example.com>"},
	}
	for name, f := range faelle {
		t.Run(name, func(t *testing.T) {
			err := VersendeUeberSMTP(testKonfig(host, port), f.absender, []string{f.empfaenger}, []byte("x"))
			if !errors.Is(err, ErrHeaderUmbruch) {
				t.Fatalf("Fehler = %v, want ErrHeaderUmbruch", err)
			}
		})
	}
}
