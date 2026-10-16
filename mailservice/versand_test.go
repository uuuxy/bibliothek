package mailservice

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/smtptest"
)

func testKonfig(host, port string) SMTPKonfig {
	return SMTPKonfig{Host: host, Port: port, Absender: "bibliothek@schule.de"}
}

// erlaubeKlartext schaltet für einen Test die Klartext-Ausnahme frei.
//
// Der Testserver (internal/smtptest) kündigt kein STARTTLS an; er prüft die Abläufe von SMTP,
// nicht TLS. Weil sichereVerbindung ohne STARTTLS abbricht, brauchen die Tests der Abläufe diese
// Ausnahme. Dass die Erzwingung wirkt, prüft TestVersendeUeberSMTPBrichtOhneSTARTTLSAb; sonst
// hätte man sie hier nur abgeschaltet.
func erlaubeKlartext(t *testing.T) {
	t.Helper()
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true")
}

// Ein Server ohne STARTTLS bekommt keine Nachricht zu sehen.
//
// Ohne diesen Test wäre die Erzwingung nicht belegt: Alle anderen Tests laufen mit gesetzter
// Ausnahme, und wer sie versehentlich abschaltete, merkte es nirgends.
func TestVersendeUeberSMTPBrichtOhneSTARTTLSAb(t *testing.T) {
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	err := versendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n"))
	if err == nil {
		t.Fatal("Versand über eine Klartextverbindung wurde als erfolgreich gemeldet")
	}
	if !errors.Is(err, errSMTPKlartext) {
		t.Errorf("Fehler = %v, want errSMTPKlartext", err)
	}
	// Der Marker entscheidet über den Status der Antwort (502 statt eines neutralen 500); ohne ihn
	// erreicht die Begründung den Admin im Formular nicht.
	if !errors.Is(err, ErrSMTPVersand) {
		t.Errorf("Fehler trägt den Marker nicht (→ falscher HTTP-Status): %v", err)
	}

	// Und der Server hat den Mailtext nie gesehen: Der Abbruch liegt vor DATA.
	select {
	case sitzung := <-sitzungen:
		if sitzung.Nachricht != "" {
			t.Fatalf("Server hat trotz Abbruch eine Nachricht empfangen: %q", sitzung.Nachricht)
		}

	default:
	}
}

// Die Gegenprobe: Mit gesetzter Ausnahme geht der Versand durch; der Ausweg für einen alten
// Relay im Schulnetz ist erreichbar und nicht nur beschrieben.
func TestVersendeUeberSMTPErlaubtKlartextMitAusnahme(t *testing.T) {
	erlaubeKlartext(t)
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	if err := versendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n")); err != nil {
		t.Fatalf("Versand mit SMTP_ALLOW_PLAINTEXT=true fehlgeschlagen: %v", err)
	}
	if sitzung := <-sitzungen; !strings.Contains(sitzung.Nachricht, "Inhalt") {
		t.Errorf("Nachricht kam nicht an: %q", sitzung.Nachricht)
	}
}

// Der gewöhnliche Fall und zugleich der Beleg, dass genau ein Empfänger im Umschlag steht.
func TestVersendeUeberSMTPStelltZu(t *testing.T) {
	erlaubeKlartext(t)
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	err := versendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
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

// Der teure Fall: Der Server nimmt die Nachricht an (250 nach dem Punkt) und kappt die
// Verbindung, ohne das QUIT zu beantworten. Die Nachricht ist zugestellt.
//
// Als Fehlschlag gemeldet, zählte der Mahnlauf eine zugestellte Mahnung als „nicht versendet",
// und der nächste Lauf schickte sie ein zweites Mal an dieselbe Klassenleitung; bei einer
// Mahnung an Eltern ist das kein kosmetischer Fehler.
func TestVersendeUeberSMTPWertetAbbruchNachAnnahmeAlsZugestellt(t *testing.T) {
	erlaubeKlartext(t)
	host, port, sitzungen := smtptest.Starte(t, smtptest.BrichtVorQuitAb)

	err := versendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n"))
	if err != nil {
		t.Fatalf("angenommene Nachricht wurde als Fehlschlag gemeldet: %v", err)
	}

	sitzung := <-sitzungen
	if !strings.Contains(sitzung.Nachricht, "Inhalt") {
		t.Errorf("Server hat die Nachricht nicht vollständig gesehen: %q", sitzung.Nachricht)
	}
}

// Die Gegenprobe: Weist der Server die Nachricht ab, ist das ein Fehlschlag, mit
// Marker (→ HTTP 502) und der Serverantwort im Klartext.
func TestVersendeUeberSMTPMeldetAbweisung(t *testing.T) {
	erlaubeKlartext(t)
	host, port, _ := smtptest.Starte(t, smtptest.LehntNachrichtAb)

	err := versendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
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

	err := versendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, []byte("Subject: Test\r\n\r\nInhalt\r\n"))
	if !errors.Is(err, ErrSMTPVersand) {
		t.Fatalf("Fehler = %v, want ErrSMTPVersand", err)
	}
	if !strings.Contains(err.Error(), host+":"+port) {
		t.Errorf("Meldung nennt den Server nicht: %v", err)
	}
}

// Ein Zeilenumbruch in einer Adresse des Umschlags schmuggelt Befehle in die Sitzung.
// net/smtp.SendMail prüft das mit; auf dem eigenen Weg steht die Prüfung ausdrücklich, und zwar
// vor dem Aufbau der Verbindung.
func TestVersendeUeberSMTPWeistUmbruchImEnvelopeAb(t *testing.T) {
	host, port := smtptest.GeschlossenerPort(t) // wird nie erreicht

	faelle := map[string]struct{ absender, empfaenger string }{
		"Umbruch im Absender":  {"bib@schule.de\r\nRCPT TO:<mitleser@example.com>", "lehrerin@schule.de"},
		"Umbruch im Empfänger": {"bib@schule.de", "lehrerin@schule.de\r\nRCPT TO:<mitleser@example.com>"},
	}
	for name, f := range faelle {
		t.Run(name, func(t *testing.T) {
			err := versendeUeberSMTP(testKonfig(host, port), f.absender, []string{f.empfaenger}, []byte("x"))
			if !errors.Is(err, errUmbruchInKopfzeile) {
				t.Fatalf("Fehler = %v, want errUmbruchInKopfzeile", err)
			}
		})
	}
}

// Der Mailtext selbst ist mit Absicht nicht gefiltert. Er darf alles enthalten, was
// ein Mensch schreiben kann, und ein Mahntext mit einer Zeile "." oder einem Doppelpunkt
// darin ist kein Angriff.
//
// Dieser Test hält fest, warum das gefahrlos ist, denn darauf zeigt CodeQL mit
// go/email-injection (Email content injection, versand.go w.Write):
//
//  1. Der Text steht hinter der Leerzeile, die Kopf und Rumpf trennt. Was dort wie eine
//     Kopfzeile aussieht, ist keine; es bleibt Fließtext.
//  2. Eine Zeile, die nur aus einem Punkt besteht, beendet in SMTP die Übertragung.
//     smtp.Client.Data() schreibt über textproto.DotWriter, der solche Zeilen verdoppelt
//     ("dot stuffing"). Ohne das könnte ein Schülername oder ein Vorlagentext die
//     Nachricht abschneiden und den Rest als SMTP-Befehle einschleusen.
//
// Die Kopfzeilen sind die Stelle, an der es wirklich gefährlich wäre; sie werden beim
// Bauen geprüft (TestBaueTextNachrichtWeistAdressenUmbruchAb) und im Umschlag noch einmal
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

	if err := versendeUeberSMTP(testKonfig(host, port), "bibliothek@schule.de",
		[]string{"lehrerin@schule.de"}, nachricht); err != nil {
		t.Fatalf("Versand fehlgeschlagen: %v", err)
	}

	sitzung := <-sitzungen

	// Der Umschlag ist unberührt: kein zweiter Empfänger, kein zweites MAIL FROM.
	if len(sitzung.Empfaenger) != 1 || sitzung.Empfaenger[0] != "lehrerin@schule.de" {
		t.Fatalf("Empfänger = %v — aus dem Mailtext wurde ein Empfänger geschmuggelt", sitzung.Empfaenger)
	}

	// Die Übertragung ist nicht am eingeschmuggelten Punkt abgerissen: Der Server hat
	// den Schluss gesehen. Ohne Dot-Stuffing endete die Nachricht bei "Guten Tag".
	if !strings.Contains(sitzung.Nachricht, "Ende der Nachricht") {
		t.Fatalf("Nachricht wurde vorzeitig beendet, Server sah nur: %q", sitzung.Nachricht)
	}

	// Und das "Bcc:" ist Fließtext geblieben; es steht hinter der trennenden Leerzeile.
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

// TestVersendeUeberSMTP_SchweigenderServerHaengtNicht: Ein Server, der die Verbindung annimmt und
// dann schweigt (Firewall, überlasteter Relay), hält versendeUeberSMTP nicht unbegrenzt fest,
// denn die Frist eines Kontexts bricht kein blockierendes Lesen auf einem Socket ab. Am
// schwersten träfe es den Wächter für kritische Alarme, dessen Schleife sonst für immer stünde.
// Der Versand kehrt innerhalb der Frist der Sitzung mit einem Fehler zurück.
func TestVersendeUeberSMTP_SchweigenderServerHaengtNicht(t *testing.T) {
	// Ein Server, der Verbindungen annimmt und die Begrüßung 220 nie sendet.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listener: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() }) //nolint:errcheck
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Verbindung halten, nichts senden.
			defer conn.Close() //nolint:errcheck
		}
	}()

	// Kürzere Fristen als im Betrieb (10 s und 60 s), damit der Test nicht eine Minute wartet.
	fristen := smtpFristen{verbindung: 2 * time.Second, sitzung: 2 * time.Second}

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	konfig := SMTPKonfig{Host: host, Port: port, Absender: "test@schule.invalid"}

	fertig := make(chan error, 1)
	go func() {
		fertig <- versendeMitFristen(fristen, konfig, "test@schule.invalid", []string{"ziel@schule.invalid"}, []byte("Subject: t\r\n\r\nx"))
	}()

	select {
	case err := <-fertig:
		if err == nil {
			t.Fatal("Versand an einen schweigenden Server darf nicht als Erfolg enden")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("versendeUeberSMTP hängt an einem schweigenden Server — die Sitzungsfrist greift nicht")
	}
}
