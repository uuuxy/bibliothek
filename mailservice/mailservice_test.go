package mailservice

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"net/mail"
	"strings"
	"testing"
)

// angriffe sind die Varianten, mit denen man eine zusätzliche Kopfzeile in eine
// Mail schmuggelt: Ein CR oder LF beendet die laufende Kopfzeile, alles danach
// liest der Mailserver als neue — ein "Bcc:" etwa, das eine Mahnung an einen
// stillen Mitleser schickt.
var angriffe = []string{
	"opfer@example.com\r\nBcc: heimlich@angreifer.de",
	"opfer@example.com\nBcc: heimlich@angreifer.de",
	"\"opfer\r\nBcc: heimlich@angreifer.de\"@example.com",
	"opfer@example.com>\r\nBcc: x@y.de <a@b.de",
	"opfer@example.com\r\n\r\nGefaelschter Textkoerper",
}

// TestParseAddressWeistKopfzeilenSchmuggelAb belegt die erste Schranke: Schon
// mail.ParseAddress lässt keine der Varianten durch. Ohne diesen Nachweis wäre die
// Bewertung des CodeQL-Fundes ("Email content injection") reine Behauptung.
func TestParseAddressWeistKopfzeilenSchmuggelAb(t *testing.T) {
	for _, p := range angriffe {
		t.Run(strings.ReplaceAll(strings.ReplaceAll(p, "\r", "\\r"), "\n", "\\n"), func(t *testing.T) {
			adr, err := mail.ParseAddress(p)
			if err != nil {
				return // abgewiesen — genau richtig
			}
			if strings.ContainsAny(adr.Address, "\r\n") {
				t.Errorf("ParseAddress ließ einen Zeilenumbruch durch: %q", adr.Address)
			}
		})
	}
}

// TestHeaderWertWeistUmbruchAb belegt die zweite Schranke direkt an der
// Schreibstelle — sie muss auch dann greifen, wenn ParseAddress davor entfiele.
func TestHeaderWertWeistUmbruchAb(t *testing.T) {
	for _, p := range angriffe {
		if _, err := HeaderWert("To", p); err == nil {
			t.Errorf("HeaderWert ließ %q durch", p)
		} else if !errors.Is(err, ErrHeaderUmbruch) {
			t.Errorf("falscher Fehlertyp für %q: %v", p, err)
		}
	}

	if _, err := HeaderWert("To", "opfer@example.com"); err != nil {
		t.Errorf("gültige Adresse wurde abgewiesen: %v", err)
	}
}

// TestBaueTextNachrichtGlaettetBetreff: Ein Umbruch im Betreff einer
// Datenbank-Vorlage ist ein Tippfehler, kein Angriff — er wird geglättet statt
// abgewiesen, damit ein Mahnlauf nicht an einem versehentlichen Enter scheitert.
// Die Kopfzeile darf danach trotzdem nicht mehrzeilig sein.
func TestBaueTextNachrichtGlaettetBetreff(t *testing.T) {
	msg, err := baueTextNachricht(
		"absender@schule.de",
		"empfaenger@example.com",
		"Mahnung\r\nBcc: heimlich@angreifer.de",
		"Textkörper",
	)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	kopf, _, gefunden := strings.Cut(string(msg), "\r\n\r\n")
	if !gefunden {
		t.Fatal("Nachricht hat keinen Kopf-/Körper-Trenner")
	}

	// Entscheidend ist die ZEILE, nicht der Text: "Bcc:" mitten im Betreff ist ein
	// Wort, "Bcc:" am Zeilenanfang wäre eine Kopfzeile.
	zeilen := strings.Split(kopf, "\r\n")
	for _, z := range zeilen {
		if strings.HasPrefix(strings.ToLower(z), "bcc:") {
			t.Errorf("Bcc-Kopfzeile eingeschmuggelt:\n%s", kopf)
		}
	}
	if len(zeilen) != 4 {
		t.Errorf("Kopf sollte 4 Zeilen haben, hatte %d:\n%s", len(zeilen), kopf)
	}
}

// TestBaueTextNachrichtWeistAdressenUmbruchAb: Beim Empfänger wird NICHT geglättet
// — eine Adresse mit Umbruch ist ein Angriff oder ein Datenfehler, beides soll
// auffallen statt stillschweigend zurechtgebogen zu werden.
func TestBaueTextNachrichtWeistAdressenUmbruchAb(t *testing.T) {
	if _, err := baueTextNachricht("a@b.de", "opfer@example.com\r\nBcc: x@y.de", "Betreff", "Text"); err == nil {
		t.Error("Empfänger mit Zeilenumbruch wurde akzeptiert")
	}
	if _, err := baueTextNachricht("a@b.de\r\nBcc: x@y.de", "opfer@example.com", "Betreff", "Text"); err == nil {
		t.Error("Absender mit Zeilenumbruch wurde akzeptiert")
	}
}

// Der Zertifikatsfall ist der einzige SMTP-Fehler, bei dem die rohe Go-Meldung
// ("x509: certificate is valid for ...") in die falsche Richtung führt: Server und
// Zugangsdaten sind in Ordnung, nur der Hostname passt nicht zum Zertifikat.
func TestBeschreibeSMTPFehlerNenntZertifikatsnamen(t *testing.T) {
	hostErr := x509.HostnameError{
		Certificate: &x509.Certificate{DNSNames: []string{"srv1.example.de"}},
		Host:        "smtp.example.de",
	}

	got := BeschreibeSMTPFehler("smtp.example.de:587", hostErr).Error()

	for _, want := range []string{"Zertifikat", "srv1.example.de", "smtp.example.de"} {
		if !strings.Contains(got, want) {
			t.Errorf("Meldung nennt %q nicht: %s", want, got)
		}
	}
}

// Ohne DNS-SANs bleibt nur der Common Name — sonst stünde dort eine leere Liste.
func TestBeschreibeSMTPFehlerFaelltAufCommonNameZurueck(t *testing.T) {
	hostErr := x509.HostnameError{
		Certificate: &x509.Certificate{Subject: pkix.Name{CommonName: "srv1.example.de"}},
		Host:        "smtp.example.de",
	}

	if got := BeschreibeSMTPFehler("smtp.example.de:587", hostErr).Error(); !strings.Contains(got, "srv1.example.de") {
		t.Errorf("Common Name fehlt in der Meldung: %s", got)
	}
}

// Alle anderen Fehler müssen unverändert durchgereicht werden, damit errors.Is/As
// weiter funktioniert und die Originalmeldung im Formular ankommt.
func TestBeschreibeSMTPFehlerReichtAndereFehlerDurch(t *testing.T) {
	original := errors.New("535 5.7.8 Authentication credentials invalid")

	err := BeschreibeSMTPFehler("smtp.example.de:587", original)

	if !errors.Is(err, original) {
		t.Fatalf("ursprünglicher Fehler wurde nicht eingepackt: %v", err)
	}
	if !strings.Contains(err.Error(), original.Error()) {
		t.Errorf("Originalmeldung fehlt: %s", err.Error())
	}
}
