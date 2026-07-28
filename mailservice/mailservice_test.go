package mailservice

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"strings"
	"testing"
)

// Der Zertifikatsfall ist der einzige SMTP-Fehler, bei dem die rohe Go-Meldung
// ("x509: certificate is valid for ...") in die falsche Richtung führt: Server und
// Zugangsdaten sind in Ordnung, nur der Hostname passt nicht zum Zertifikat.
func TestDescribeSMTPErrorNenntZertifikatsnamen(t *testing.T) {
	hostErr := x509.HostnameError{
		Certificate: &x509.Certificate{DNSNames: []string{"srv1.example.de"}},
		Host:        "smtp.example.de",
	}

	got := describeSMTPError("smtp.example.de:587", hostErr).Error()

	for _, want := range []string{"Zertifikat", "srv1.example.de", "smtp.example.de"} {
		if !strings.Contains(got, want) {
			t.Errorf("Meldung nennt %q nicht: %s", want, got)
		}
	}
}

// Ohne DNS-SANs bleibt nur der Common Name — sonst stünde dort eine leere Liste.
func TestDescribeSMTPErrorFaelltAufCommonNameZurueck(t *testing.T) {
	hostErr := x509.HostnameError{
		Certificate: &x509.Certificate{Subject: pkix.Name{CommonName: "srv1.example.de"}},
		Host:        "smtp.example.de",
	}

	if got := describeSMTPError("smtp.example.de:587", hostErr).Error(); !strings.Contains(got, "srv1.example.de") {
		t.Errorf("Common Name fehlt in der Meldung: %s", got)
	}
}

// Alle anderen Fehler müssen unverändert durchgereicht werden, damit errors.Is/As
// weiter funktioniert und die Originalmeldung im Formular ankommt.
func TestDescribeSMTPErrorReichtAndereFehlerDurch(t *testing.T) {
	original := errors.New("535 5.7.8 Authentication credentials invalid")

	err := describeSMTPError("smtp.example.de:587", original)

	if !errors.Is(err, original) {
		t.Fatalf("ursprünglicher Fehler wurde nicht eingepackt: %v", err)
	}
	if !strings.Contains(err.Error(), original.Error()) {
		t.Errorf("Originalmeldung fehlt: %s", err.Error())
	}
}
