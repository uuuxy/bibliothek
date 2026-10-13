// Package smtptest stellt einen Mailserver für Tests bereit, der Nachrichten annimmt, ohne
// sie zuzustellen, und sich danach befragen lässt.
//
// Der Versandweg ließe sich sonst nur in der Browser-Suite prüfen, und die läuft gegen einen
// Stack, dessen Mailserver der der Schule ist: Ein Test des Versands verschickte dort echte
// Mails. Hier bekommt jeder Test seinen eigenen Server auf einem freien Port.
//
// Das Verhalten ist wählbar, weil die schwierigen Fälle nicht der gewöhnliche sind: Ein
// Server, der die Nachricht annimmt und die Verbindung dann kappt, ohne sich zu
// verabschieden, kommt im Schulnetz vor. Ob das als versendet oder als fehlgeschlagen gilt,
// entscheidet, ob eine schon zugestellte Mahnung beim nächsten Lauf ein zweites Mal hinausgeht.
package smtptest

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// Verhalten wählt, wie sich der Server des Tests verhält.
type Verhalten int

const (
	// Normal nimmt die Nachricht an und verabschiedet sich ordentlich.
	Normal Verhalten = iota
	// BrichtVorQuitAb nimmt die Nachricht an (250 nach dem Punkt) und kappt die Verbindung,
	// bevor der Client die Antwort auf sein QUIT bekommt.
	BrichtVorQuitAb
	// LehntNachrichtAb weist die Nachricht nach der Übertragung ab, etwa wegen ihrer Größe.
	LehntNachrichtAb
)

// Sitzung hält fest, was der Server zu sehen bekam.
type Sitzung struct {
	Empfaenger []string
	Nachricht  string
}

// Starte nimmt Sitzungen an und spricht so viel SMTP, wie ein Client ohne Anmeldung braucht.
// Die Antwort auf EHLO nennt weder STARTTLS noch AUTH: So bleibt es der Versand im Klartext,
// den ein Relay im Schulnetz auch fährt, und der Test hängt nicht an einem Zertifikat.
//
// Der Kanal liefert jede Sitzung, sobald ihr Client fertig ist. Der Server bedient jede
// Verbindung und nicht nur die erste: Sonst bliebe ein Test, der prüft, dass genau eine Mail
// hinausgeht, bei der zweiten stehen, statt fehlzuschlagen.
func Starte(t *testing.T, verhalten Verhalten) (host, port string, sitzungen <-chan Sitzung) {
	t.Helper()

	lauscher, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Test-SMTP konnte nicht lauschen: %v", err)
	}
	t.Cleanup(func() { _ = lauscher.Close() }) //nolint:errcheck // Test-Listener

	// Der Kanal ist gepuffert, damit der Server weiterarbeitet, auch wenn der Test die
	// Nachrichten später oder gar nicht abholt.
	kanal := make(chan Sitzung, 8)
	go lausche(lauscher, verhalten, kanal)

	_, p, err := net.SplitHostPort(lauscher.Addr().String())
	if err != nil {
		t.Fatalf("Port des Test-SMTP unlesbar: %v", err)
	}
	return "127.0.0.1", p, kanal
}

// GeschlossenerPort liefert Adresse und Port, auf dem niemand lauscht: für den häufigsten
// Fehler der Einstellung, den falschen Port oder Host.
func GeschlossenerPort(t *testing.T) (host, port string) {
	t.Helper()
	lauscher, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Port konnte nicht belegt werden: %v", err)
	}
	_, p, err := net.SplitHostPort(lauscher.Addr().String())
	if err != nil {
		t.Fatalf("Port unlesbar: %v", err)
	}
	if err := lauscher.Close(); err != nil {
		t.Fatalf("Port konnte nicht freigegeben werden: %v", err)
	}
	return "127.0.0.1", p
}

// lausche bedient jede eingehende Verbindung, bis der Lauscher am Ende des Tests geschlossen
// wird.
func lausche(lauscher net.Listener, verhalten Verhalten, kanal chan<- Sitzung) {
	for {
		verbindung, err := lauscher.Accept()
		if err != nil {
			return
		}
		go bediene(verbindung, verhalten, kanal)
	}
}

func bediene(verbindung net.Conn, verhalten Verhalten, kanal chan<- Sitzung) {
	defer func() { _ = verbindung.Close() }() //nolint:errcheck // Testverbindung

	leser := bufio.NewReader(verbindung)
	// Ein Fehler beim Schreiben braucht keine Behandlung: Bricht die Verbindung weg, scheitert
	// der Versand, und darüber urteilt der Test am Ergebnis.
	schreibe := func(zeile string) {
		_, _ = verbindung.Write([]byte(zeile + "\r\n")) //nolint:errcheck
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
				// Angenommen, und dann ohne Verabschiedung weg.
				kanal <- sitzung
				return
			}
		case befehl == "QUIT":
			schreibe("221 2.0.0 Bye")
			kanal <- sitzung
			return
		default:
			schreibe("250 Ok")
		}
	}
	kanal <- sitzung
}

func leseNachricht(leser *bufio.Reader) string {
	var rumpf strings.Builder
	for {
		zeile, err := leser.ReadString('\n')
		if err != nil {
			break
		}
		if strings.TrimRight(zeile, "\r\n") == "." {
			break
		}
		rumpf.WriteString(zeile)
	}
	return rumpf.String()
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
