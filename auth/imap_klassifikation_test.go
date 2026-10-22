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
// `antwort` reagiert; "CLOSE" schließt stattdessen die Verbindung ohne Antwort. logins erhält
// jede LOGIN-Zeile, wie sie ankam.
func miniIMAP(t *testing.T, antwort string) (host, port string, logins <-chan string) {
	t.Helper()
	eingegangen := make(chan string, 8)
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
						select {
						case eingegangen <- eingang:
						default:
						}
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
	return host, port, eingegangen
}

// mitMiniIMAP stellt den Testserver ein und lässt a sein Wegwerf-Zertifikat zu.
func mitMiniIMAP(t *testing.T, a *Anmeldung, antwort string) <-chan string {
	t.Helper()
	host, port, logins := miniIMAP(t, antwort)
	t.Setenv("APP_ENV", "production")
	t.Setenv("IMAP_HOST", host)
	t.Setenv("IMAP_PORT", port)
	a.imapTLS = func(c *tls.Config) { c.InsecureSkipVerify = true } //nolint:gosec // Testserver mit Wegwerf-Zertifikat
	return logins
}

func TestPruefeIMAPAnmeldung_NOIstFalschesPasswortKeinAusfall(t *testing.T) {
	a := &Anmeldung{}
	mitMiniIMAP(t, a, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	err := a.pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "falsch")
	if err == nil {
		t.Fatal("pruefeIMAPAnmeldung(…) bei NO = nil, erwartet einen Fehler")
	}
	if errors.Is(err, errMailserverNichtErreichbar) {
		t.Fatalf("NO vom Server ist ein falsches Passwort, kein Ausfall: %v", err)
	}
	if !errors.Is(err, errAnmeldungAbgelehnt) {
		t.Fatalf("a.pruefeIMAPAnmeldung(…) = %v, erwartet errAnmeldungAbgelehnt", err)
	}
}

func TestPruefeIMAPAnmeldung_VerbindungsabbruchBeimAnmeldenIstAusfall(t *testing.T) {
	a := &Anmeldung{}
	mitMiniIMAP(t, a, "CLOSE")
	err := a.pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "egal")
	if !errors.Is(err, errMailserverNichtErreichbar) {
		t.Fatalf("Server schließt beim LOGIN ohne Antwort — das ist ein Ausfall, kein Passwortfehler: %v", err)
	}
}

func TestPruefeIMAPAnmeldung_OKMeldetErfolg(t *testing.T) {
	a := &Anmeldung{}
	mitMiniIMAP(t, a, "OK LOGIN completed")
	if err := a.pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "richtig"); err != nil {
		t.Fatalf("OK vom Server muss Erfolg sein: %v", err)
	}
}

// Ein Name ohne „@" geht so an den Mailserver, wie er eingegeben wurde. Die Domain der Schule
// steht nicht im Code: Ihre eine Quelle ist SELBSTANMELDUNG_DOMAIN, und eine hier angehängte
// Domain fände danach ohnehin kein Konto, weil die Kontosuche die Eingabe nimmt.
func TestPruefeIMAPAnmeldung_HaengtKeineDomainAn(t *testing.T) {
	a := &Anmeldung{}
	logins := mitMiniIMAP(t, a, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	if err := a.pruefeIMAPAnmeldung(context.Background(), "nberger", "egal"); !errors.Is(err, errAnmeldungAbgelehnt) {
		t.Fatalf("a.pruefeIMAPAnmeldung(…) = %v, erwartet errAnmeldungAbgelehnt", err)
	}
	zeile := <-logins
	if felder := strings.Fields(zeile); len(felder) < 3 || felder[2] != "nberger" && felder[2] != `"nberger"` {
		t.Fatalf("LOGIN-Zeile = %q, erwartet den Namen „nberger“ ohne Domain", zeile)
	}
}

// Die Tür der Anmeldung muss dem Mailserver Luft lassen, sonst endet ihr Kontext zwischen
// „Passwort richtig" und dem Lesen des Kontos.
func TestAnmeldeFrist_LaesstIMAPLuft(t *testing.T) {
	if anmeldeFrist <= imapFrist+2*time.Second {
		t.Fatalf("anmeldeFrist (%v) muss deutlich über imapFrist (%v) liegen", anmeldeFrist, imapFrist)
	}
}
