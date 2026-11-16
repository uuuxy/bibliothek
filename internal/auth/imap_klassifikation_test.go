package auth

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"log/slog"
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
// `antwort` reagiert; "CLOSE" schließt stattdessen die Verbindung ohne Antwort, "SCHWEIGT"
// antwortet nicht. befehle erhält jede Zeile mit LOGIN oder LOGOUT, wie sie ankam, und
// verbindungZu, wenn der Client die Verbindung schließt.
const verbindungZu = "(Verbindung vom Client geschlossen)"

func miniIMAP(t *testing.T, antwort string) (host, port string, befehle <-chan string) {
	t.Helper()
	eingegangen := make(chan string, 8)
	melde := func(zeile string) {
		select {
		case eingegangen <- zeile:
		default:
		}
	}
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
						melde(verbindungZu) // der Client hat die Verbindung geschlossen
						return
					}
					tag := strings.Fields(eingang)[0]
					switch {
					case strings.Contains(strings.ToUpper(eingang), "LOGIN"):
						melde(eingang)
						if antwort == "CLOSE" {
							return
						}
						if antwort == "SCHWEIGT" {
							continue
						}
						if !schreibe(tag + " " + antwort + "\r\n") {
							return
						}
					case strings.Contains(strings.ToUpper(eingang), "LOGOUT"):
						melde(eingang)
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
	host, port, befehle := miniIMAP(t, antwort)
	t.Setenv("APP_ENV", "production")
	t.Setenv("IMAP_HOST", host)
	t.Setenv("IMAP_PORT", port)
	a.imapTLS = func(c *tls.Config) { c.InsecureSkipVerify = true } //nolint:gosec // Testserver mit Wegwerf-Zertifikat
	return befehle
}

func TestPruefeIMAPAnmeldung_NOIstFalschesPasswortKeinAusfall(t *testing.T) {
	a := &Anmeldung{}
	mitMiniIMAP(t, a, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	err := a.pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "falsch")
	if err == nil {
		t.Fatal("pruefeIMAPAnmeldung(…) bei NO = nil, erwartet einen Fehler")
	}
	if errors.Is(err, errAnmeldeserverNichtErreichbar) {
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
	if !errors.Is(err, errAnmeldeserverNichtErreichbar) {
		t.Fatalf("Server schließt beim LOGIN ohne Antwort — das ist ein Ausfall, kein Passwortfehler: %v", err)
	}
}

// Nach dem Erfolg meldet sich die Prüfung beim Mailserver ab. Ohne LOGOUT bliebe je Anmeldung
// eine Verbindung offen, bis der Server sie von sich aus schließt.
func TestPruefeIMAPAnmeldung_OKMeldetErfolg(t *testing.T) {
	a := &Anmeldung{}
	befehle := mitMiniIMAP(t, a, "OK LOGIN completed")
	if err := a.pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "richtig"); err != nil {
		t.Fatalf("OK vom Server muss Erfolg sein: %v", err)
	}
	<-befehle // LOGIN
	select {
	case zeile := <-befehle:
		if !strings.Contains(strings.ToUpper(zeile), "LOGOUT") {
			t.Errorf("nach dem LOGIN kam %q, erwartet LOGOUT", zeile)
		}
	case <-time.After(2 * time.Second):
		t.Error("nach der erfolgreichen Anmeldung kein LOGOUT: die Verbindung zum Mailserver bliebe offen")
	}
}

// Ein Name ohne „@" geht so an den Mailserver, wie er eingegeben wurde. Die Domain der Schule
// steht nicht im Code: Ihre eine Quelle ist SELBSTANMELDUNG_DOMAIN, und eine hier angehängte
// Domain fände danach ohnehin kein Konto, weil die Kontosuche die Eingabe nimmt.
func TestPruefeIMAPAnmeldung_HaengtKeineDomainAn(t *testing.T) {
	a := &Anmeldung{}
	befehle := mitMiniIMAP(t, a, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	if err := a.pruefeIMAPAnmeldung(context.Background(), "nberger", "egal"); !errors.Is(err, errAnmeldungAbgelehnt) {
		t.Fatalf("a.pruefeIMAPAnmeldung(…) = %v, erwartet errAnmeldungAbgelehnt", err)
	}
	zeile := <-befehle
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

// Ein gescheitertes LOGIN schließt die Verbindung einmal und meldet sich nicht mehr ab. Ein
// LOGOUT über die geschlossene Verbindung schrieb bei jedem falschen Passwort „imap: Logout
// fehlgeschlagen … use of closed network connection“, die abgelaufene Frist dazu „close failed“.
// Geschlossen wird trotzdem: Sonst bliebe je Fehlversuch eine Verbindung zum Mailserver offen.
func TestPruefeIMAPAnmeldung_FehlschlagSchliesstOhneWarnung(t *testing.T) {
	for _, fall := range []struct {
		name, antwort   string
		serverSchliesst bool
	}{
		{"falsches Passwort", "NO [AUTHENTICATIONFAILED] Authentication failed.", false},
		{"Server bricht ab", "CLOSE", true},
		{"Server schweigt bis zur Frist", "SCHWEIGT", false},
	} {
		t.Run(fall.name, func(t *testing.T) {
			a := &Anmeldung{}
			befehle := mitMiniIMAP(t, a, fall.antwort)
			var protokoll bytes.Buffer
			vorher := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&protokoll, nil)))
			t.Cleanup(func() { slog.SetDefault(vorher) })
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()

			if err := a.pruefeIMAPAnmeldung(ctx, "wer@schule.de", "falsch"); err == nil {
				t.Fatal("pruefeIMAPAnmeldung(…) = nil, erwartet einen Fehler")
			}
			for _, warnung := range []string{"Logout fehlgeschlagen", "close failed"} {
				if strings.Contains(protokoll.String(), warnung) {
					t.Errorf("Protokoll enthält %q: %s", warnung, protokoll.String())
				}
			}
			if fall.serverSchliesst {
				return
			}
			frist := time.After(2 * time.Second)
			for {
				select {
				case zeile := <-befehle:
					if zeile == verbindungZu {
						return
					}
				case <-frist:
					t.Fatal("die Verbindung zum Mailserver bleibt nach dem Fehlschlag offen")
				}
			}
		})
	}
}
