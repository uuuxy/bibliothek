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

// Ob ein Fehler am Mailserver „falsches Passwort" oder „Ausfall" heißt, entscheidet der Inhalt,
// nicht die Uhr: Nur eine Antwort des Servers (NO oder BAD) heißt falsche Zugangsdaten. Ein
// Server, der die Verbindung beim LOGIN schließt, ist ein Ausfall, auch vor Ablauf der Frist. Die
// Tests sprechen über TLS mit einem kleinen Mailserver, nicht mit einer Attrappe der eigenen
// Logik.

// miniIMAP startet einen Mailserver über TLS, der seine Begrüßung schickt und auf LOGIN mit
// `antwort` reagiert; "CLOSE" schließt stattdessen die Verbindung ohne Antwort.
func miniIMAP(t *testing.T, antwort string) (host, port string) {
	t.Helper()
	schluessel, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	vorlage := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "127.0.0.1"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, vorlage, vorlage, &schluessel.PublicKey, schluessel)
	if err != nil {
		t.Fatal(err)
	}
	zertifikat := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: schluessel}
	l, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{zertifikat}, MinVersion: tls.VersionTLS12})
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
			verbindung, err := l.Accept()
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
					eingang, err := r.ReadString('\n')
					if err != nil {
						return
					}
					tag := strings.Fields(eingang)[0]
					switch {
					case strings.Contains(strings.ToUpper(eingang), "LOGIN"):
						if antwort == "CLOSE" {
							return
						}
						if !schreibe(tag + " " + antwort + "\r\n") {
							return
						}
					case strings.Contains(strings.ToUpper(eingang), "LOGOUT"):
						schreibe("* BYE\r\n" + tag + " OK\r\n")
						return
					default:
						if !schreibe(tag + " BAD unbekannt\r\n") {
							return
						}
					}
				}
			}(verbindung)
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

func TestPruefeIMAPAnmeldung_NOIstFalschesPasswortKeinAusfall(t *testing.T) {
	mitMiniIMAP(t, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	err := pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "falsch")
	if err == nil {
		t.Fatal("erwartet Fehler bei NO")
	}
	if errors.Is(err, errMailserverNichtErreichbar) {
		t.Fatalf("NO vom Server ist ein falsches Passwort, kein Ausfall: %v", err)
	}
	if !errors.Is(err, errAnmeldungAbgelehnt) {
		t.Fatalf("erwartet errAnmeldungAbgelehnt, bekam %v", err)
	}
}

func TestPruefeIMAPAnmeldung_VerbindungsabbruchBeimLoginIstAusfall(t *testing.T) {
	mitMiniIMAP(t, "CLOSE")
	err := pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "egal")
	if !errors.Is(err, errMailserverNichtErreichbar) {
		t.Fatalf("Server schließt beim LOGIN ohne Antwort — das ist ein Ausfall, kein Passwortfehler: %v", err)
	}
}

func TestPruefeIMAPAnmeldung_OKMeldetErfolg(t *testing.T) {
	mitMiniIMAP(t, "OK LOGIN completed")
	if err := pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "richtig"); err != nil {
		t.Fatalf("OK vom Server muss Erfolg sein: %v", err)
	}
}

// Die Tür der Anmeldung muss dem Mailserver Luft lassen, sonst endet ihr Kontext zwischen
// „Passwort richtig" und dem Lesen des Kontos.
func TestAnmeldeFrist_LaesstIMAPLuft(t *testing.T) {
	if anmeldeFrist <= imapFrist+2*time.Second {
		t.Fatalf("anmeldeFrist (%v) muss deutlich über imapFrist (%v) liegen", anmeldeFrist, imapFrist)
	}
}
