package auth

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

// Prüfung 22.08.2026, A7: Ob ein IMAP-Fehler „falsches Passwort" oder „Ausfall" heißt,
// entschied bis dahin die UHR (Frist abgelaufen = Ausfall, sonst Passwort). Ein Server,
// der die Verbindung beim LOGIN schließt, zählte damit als Fehlversuch; ein Tarpit über
// 15 s machte jedes falsche Passwort zum 503. Jetzt entscheidet der INHALT: Nur eine
// Status-Antwort (NO/BAD) ist ein Zugangsdaten-Fehler. Diese Tests sprechen mit einem
// Mini-IMAP-Server über TLS — keine Mocks der eigenen Logik.

// miniIMAP startet einen TLS-Listener, der das Greeting schickt und auf LOGIN mit
// `antwort` reagiert; "CLOSE" schließt stattdessen die Verbindung ohne Antwort.
func miniIMAP(t *testing.T, antwort string) (host, port string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "127.0.0.1"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	l, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Logf("Listener schließen: %v", err)
		}
	})
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close() //nolint:errcheck
				schreibe := func(zeile string) bool {
					_, err := c.Write([]byte(zeile))
					return err == nil
				}
				if !schreibe("* OK Mini-IMAP bereit\r\n") {
					return
				}
				r := bufio.NewReader(c)
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					tag := strings.Fields(line)[0]
					switch {
					case strings.Contains(strings.ToUpper(line), "LOGIN"):
						if antwort == "CLOSE" {
							return
						}
						if !schreibe(tag + " " + antwort + "\r\n") {
							return
						}
					case strings.Contains(strings.ToUpper(line), "LOGOUT"):
						schreibe("* BYE\r\n" + tag + " OK\r\n")
						return
					default:
						if !schreibe(tag + " BAD unbekannt\r\n") {
							return
						}
					}
				}
			}(conn)
		}
	}()
	host, port, err = net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return host, port
}

func mitMiniIMAP(t *testing.T, antwort string) {
	t.Helper()
	host, port := miniIMAP(t, antwort)
	t.Setenv("APP_ENV", "production")
	t.Setenv("IMAP_HOST", host)
	t.Setenv("IMAP_PORT", port)
	alt := imapTLSAnpassung
	imapTLSAnpassung = func(c *tls.Config) { c.InsecureSkipVerify = true } //nolint:gosec // Testserver mit Wegwerf-Zertifikat
	t.Cleanup(func() { imapTLSAnpassung = alt })
}

func TestAuthenticateIMAP_NOIstFalschesPasswortKeinAusfall(t *testing.T) {
	mitMiniIMAP(t, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	err := AuthenticateIMAP(context.Background(), "wer@schule.de", "falsch")
	if err == nil {
		t.Fatal("erwartet Fehler bei NO")
	}
	if errors.Is(err, ErrMailserverNichtErreichbar) {
		t.Fatalf("NO vom Server ist ein falsches Passwort, kein Ausfall: %v", err)
	}
	if !errors.Is(err, errAnmeldungAbgelehnt) {
		t.Fatalf("erwartet errAnmeldungAbgelehnt, bekam %v", err)
	}
}

func TestAuthenticateIMAP_VerbindungsabbruchBeimLoginIstAusfall(t *testing.T) {
	mitMiniIMAP(t, "CLOSE")
	err := AuthenticateIMAP(context.Background(), "wer@schule.de", "egal")
	if !errors.Is(err, ErrMailserverNichtErreichbar) {
		t.Fatalf("Server schließt beim LOGIN ohne Antwort — das ist ein Ausfall, kein Passwortfehler: %v", err)
	}
}

func TestAuthenticateIMAP_OKMeldetErfolg(t *testing.T) {
	mitMiniIMAP(t, "OK LOGIN completed")
	if err := AuthenticateIMAP(context.Background(), "wer@schule.de", "richtig"); err != nil {
		t.Fatalf("OK vom Server muss Erfolg sein: %v", err)
	}
}

// Der Handler muss dem IMAP-Schritt Luft lassen — sonst stirbt der Kontext zwischen
// „Passwort richtig" und dem Benutzer-Lookup (so bis 22.08.2026: 10 s gegen 15 s).
func TestLoginHandlerFrist_LaesstIMAPLuft(t *testing.T) {
	if loginHandlerFrist <= imapFrist+2*time.Second {
		t.Fatalf("loginHandlerFrist (%v) muss deutlich über imapFrist (%v) liegen", loginHandlerFrist, imapFrist)
	}
}
