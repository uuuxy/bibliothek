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

// erlaubeKlartext schaltet für einen Test die Klartext-Ausnahme frei.
//
// Der Test-Server (internal/smtptest) kündigt bewusst kein STARTTLS an — er soll die
// SMTP-Ablaufe prüfen, nicht TLS. Seit sichereVerbindung den Versand ohne STARTTLS
// abbricht, brauchen genau deshalb alle Ablauf-Tests diese Ausnahme. Dass die
// Erzwingung selbst wirkt, prüft TestVersendeUeberSMTPBrichtOhneSTARTTLSAb — sonst
// hätte man sie hier nur wegkonfiguriert.
func erlaubeKlartext(t *testing.T) {
	t.Helper()
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true")
}

// Das Gate: Ein Server ohne STARTTLS bekommt keine Nachricht zu sehen.
//
// Ohne diesen Test wäre die Erzwingung nicht belegt — alle anderen Tests laufen mit
// gesetzter Ausnahme, und ein versehentliches Zurückdrehen auf "dann eben ohne" fiele
// nirgends auf.
func TestVersendeUeberSMTPBrichtOhneSTARTTLSAb(t *testing.T) {
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	err := VersendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n"))
	if err == nil {
		t.Fatal("Versand über eine Klartextverbindung wurde als erfolgreich gemeldet")
	}
	if !errors.Is(err, ErrSMTPKlartext) {
		t.Errorf("Fehler = %v, want ErrSMTPKlartext", err)
	}
	// Der Marker entscheidet über den HTTP-Status (502 statt neutralisierter 500) —
	// ohne ihn erreicht die Begründung den Admin im Formular nicht.
	if !errors.Is(err, ErrSMTPVersand) {
		t.Errorf("Fehler trägt den Marker nicht (→ falscher HTTP-Status): %v", err)
	}

	// Und der Server hat den Mailtext nie gesehen: Der Abbruch liegt vor DATA.
	select {
	case sitzung := <-sitzungen:
		t.Fatalf("Server hat trotz Abbruch eine Nachricht empfangen: %q", sitzung.Nachricht)
	default:
	}
}

// Die Gegenprobe zum Gate: Mit gesetzter Ausnahme geht der Versand durch — der
// Escape-Hatch für ein Legacy-Relay im Schulnetz ist erreichbar und nicht nur
// dokumentiert.
func TestVersendeUeberSMTPErlaubtKlartextMitAusnahme(t *testing.T) {
	erlaubeKlartext(t)
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	if err := VersendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n")); err != nil {
		t.Fatalf("Versand mit SMTP_ALLOW_PLAINTEXT=true fehlgeschlagen: %v", err)
	}
	if sitzung := <-sitzungen; !strings.Contains(sitzung.Nachricht, "Inhalt") {
		t.Errorf("Nachricht kam nicht an: %q", sitzung.Nachricht)
	}
}

// Der Normalfall — und zugleich der Beleg, dass genau ein Empfänger im Envelope steht.
func TestVersendeUeberSMTPStelltZu(t *testing.T) {
	erlaubeKlartext(t)
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
	erlaubeKlartext(t)
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
	erlaubeKlartext(t)
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

// Der Mailtext selbst ist NICHT gefiltert — mit Absicht. Er darf alles enthalten, was
// ein Mensch schreiben kann, und ein Mahntext mit einer Zeile "." oder einem Doppelpunkt
// darin ist kein Angriff.
//
// Dieser Test hält fest, WARUM das gefahrlos ist, denn genau darauf zeigt CodeQL mit
// go/email-injection (Email content injection, versand.go w.Write):
//
//  1. Der Text steht hinter der Leerzeile, die Kopf und Rumpf trennt. Was dort wie eine
//     Kopfzeile aussieht, IST keine — es bleibt Fließtext.
//  2. Eine Zeile, die nur aus einem Punkt besteht, beendet in SMTP die Übertragung.
//     smtp.Client.Data() schreibt über textproto.DotWriter, der solche Zeilen verdoppelt
//     ("dot stuffing"). Ohne das könnte ein Schülername oder ein Vorlagentext die
//     Nachricht abschneiden und den Rest als SMTP-Befehle einschleusen.
//
// Die Kopfzeilen sind die Stelle, an der es wirklich gefährlich wäre — die werden beim
// Bauen geprüft (TestBaueTextNachrichtWeistAdressenUmbruchAb) und im Envelope nochmal
// (TestVersendeUeberSMTPWeistUmbruchImEnvelopeAb). Fällt dieser Test hier, ist die
// Einschätzung "Fließtext ist ungefährlich" nicht mehr gedeckt.
func TestVersendeUeberSMTPSchmuggeltNichtsAusDemMailtext(t *testing.T) {
	erlaubeKlartext(t)
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	// Ein Text, wie ihn ein Angreifer in eine Vorlage oder einen Namen schreiben würde.
	boesartig := "Guten Tag\r\n.\r\nMAIL FROM:<angreifer@example.com>\r\n" +
		"Bcc: mitleser@example.com\r\nEnde der Nachricht"

	nachricht, err := baueTextNachricht("bibliothek@schule.de", "lehrerin@schule.de", "Mahnung", boesartig)
	if err != nil {
		t.Fatalf("Nachricht konnte nicht gebaut werden: %v", err)
	}

	if err := VersendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, nachricht); err != nil {
		t.Fatalf("Versand fehlgeschlagen: %v", err)
	}

	sitzung := <-sitzungen

	// Der Envelope ist unberührt: Kein zweiter Empfänger, kein zweites MAIL FROM.
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "lehrerin@schule.de" {
		t.Fatalf("Empfänger = %v — aus dem Mailtext wurde ein Empfänger geschmuggelt", sitzung.Empfaenger)
	}

	// Die Übertragung ist nicht am eingeschmuggelten Punkt abgerissen: Der Server hat
	// den Schluss gesehen. Ohne Dot-Stuffing endete die Nachricht bei "Guten Tag".
	if !strings.Contains(sitzung.Nachricht, "Ende der Nachricht") {
		t.Fatalf("Nachricht wurde vorzeitig beendet, Server sah nur: %q", sitzung.Nachricht)
	}

	// Und das "Bcc:" ist Fließtext geblieben — es steht hinter der Trennleerzeile.
	kopf, rumpf, getrennt := strings.Cut(sitzung.Nachricht, "\r\n\r\n")
	if !getrennt {
		t.Fatal("Nachricht hatte keine Trennung zwischen Kopf und Rumpf")
	}
	if strings.Contains(kopf, "Bcc") || strings.Contains(kopf, "mitleser@example.com") {
		t.Errorf("aus dem Mailtext wurde eine Kopfzeile: %q", kopf)
	}
	if !strings.Contains(rumpf, "Bcc: mitleser@example.com") {
		t.Errorf("der Text kam nicht als Rumpf an: %q", rumpf)
	}
}
