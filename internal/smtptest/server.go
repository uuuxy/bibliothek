// Package smtptest stellt einen Mailserver für Tests bereit, der Nachrichten
// annimmt, ohne sie irgendwohin zuzustellen — und sich danach befragen lässt.
//
// Warum ein eigenes Päckchen: Der Versandweg lässt sich sonst nur in der E2E-Suite
// prüfen, und die läuft gegen einen Stack, dessen SMTP-Konfiguration auf den
// Schulserver zeigt. Ein Test des echten Versands würde dort echte Mails
// verschicken. Hier bekommt jeder Test seinen eigenen Server auf einem freien Port.
//
// Das Verhalten ist wählbar, weil die interessanten Fälle nicht der Normalfall sind:
// Ein Server, der die Nachricht annimmt und die Verbindung dann kappt, ohne sich zu
// verabschieden, ist im Schulnetz keine Seltenheit — und die Frage, ob das als
// „versendet" oder als „fehlgeschlagen" gilt, entscheidet darüber, ob eine bereits
// zugestellte Mahnung beim nächsten Lauf ein zweites Mal rausgeht.
package smtptest

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// Verhalten steuert, wie sich der Testserver verhält.
type Verhalten int

const (
	// Normal nimmt die Nachricht an und verabschiedet sich ordentlich.
	Normal Verhalten = iota
	// BrichtVorQuitAb nimmt die Nachricht an (250 nach dem Punkt) und kappt die
	// Verbindung, bevor der Client sein QUIT beantwortet bekommt.
	BrichtVorQuitAb
	// LehntNachrichtAb weist die Nachricht nach der Übertragung ab (z.B. Größenlimit).
	LehntNachrichtAb
)

// Sitzung hält fest, was der Server tatsächlich zu sehen bekam.
type Sitzung struct {
	Empfaenger []string
	Nachricht  string
}

// Starte nimmt genau eine Sitzung an und spricht das Minimum an SMTP, das ein
// Client ohne Authentifizierung braucht. Bewusst ohne STARTTLS/AUTH in der
// EHLO-Antwort: Damit bleibt der Ablauf der schlichte Klartext-Versand, den ein
// Relay im Schulnetz auch fährt, und der Test hängt nicht an einem Testzertifikat.
//
// Der Kanal liefert die Sitzung, sobald der Client fertig ist.
func Starte(t *testing.T, verhalten Verhalten) (host, port string, sitzungen <-chan Sitzung) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Test-SMTP konnte nicht lauschen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() }) //nolint:errcheck // Test-Listener

	ch := make(chan Sitzung, 1)
	go bediene(ln, verhalten, ch)

	_, p, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("Port des Test-SMTP unlesbar: %v", err)
	}
	return "127.0.0.1", p, ch
}

// GeschlossenerPort liefert Adresse und Port, auf dem sicher niemand lauscht —
// für den häufigsten Konfigurationsfehler: falscher Port, falscher Host.
func GeschlossenerPort(t *testing.T) (host, port string) {
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

func bediene(ln net.Listener, verhalten Verhalten, ch chan<- Sitzung) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }() //nolint:errcheck // Testverbindung

	leser := bufio.NewReader(conn)
	// Schreibfehler brauchen keine Behandlung: Bricht die Verbindung weg, scheitert
	// der Versand — und genau darüber urteilt der Test ohnehin anhand des Ergebnisses.
	schreibe := func(zeile string) {
		_, _ = conn.Write([]byte(zeile + "\r\n")) //nolint:errcheck
	}

	var sitzung Sitzung
	schreibe("220 smtptest ESMTP")
	for {
		zeile, err := leser.ReadString('\n')
		if err != nil {
			break
		}
		befehl := strings.ToUpper(strings.TrimSpace(zeile))

		switch {
		case strings.HasPrefix(befehl, "EHLO"), strings.HasPrefix(befehl, "HELO"):
			schreibe("250 smtptest")
		case strings.HasPrefix(befehl, "MAIL FROM"):
			schreibe("250 2.1.0 Ok")
		case strings.HasPrefix(befehl, "RCPT TO"):
			sitzung.Empfaenger = append(sitzung.Empfaenger, adresseAus(zeile))
			schreibe("250 2.1.5 Ok")
		case befehl == "DATA":
			schreibe("354 Ende mit <CR><LF>.<CR><LF>")
			sitzung.Nachricht = leseNachricht(leser)
			if verhalten == LehntNachrichtAb {
				schreibe("552 5.3.4 Nachricht zu groß")
				continue
			}
			schreibe("250 2.0.0 Ok: queued")
			if verhalten == BrichtVorQuitAb {
				// Angenommen — und jetzt weg, ohne Verabschiedung.
				ch <- sitzung
				return
			}
		case befehl == "QUIT":
			schreibe("221 2.0.0 Bye")
			ch <- sitzung
			return
		default:
			schreibe("250 Ok")
		}
	}
	ch <- sitzung
}

func leseNachricht(leser *bufio.Reader) string {
	var body strings.Builder
	for {
		zeile, err := leser.ReadString('\n')
		if err != nil {
			break
		}
		if strings.TrimRight(zeile, "\r\n") == "." {
			break
		}
		body.WriteString(zeile)
	}
	return body.String()
}

// adresseAus schält die Adresse aus "RCPT TO:<a@b.de>".
func adresseAus(zeile string) string {
	auf := strings.Index(zeile, "<")
	zu := strings.LastIndex(zeile, ">")
	if auf == -1 || zu <= auf {
		return strings.TrimSpace(zeile)
	}
	return zeile[auf+1 : zu]
}
