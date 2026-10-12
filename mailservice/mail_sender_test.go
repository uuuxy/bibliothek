package mailservice

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendEmail_InvalidRecipient(t *testing.T) {
	// Setup required env variables to pass the first validation step.
	// t.Setenv restores the previous value automatically when the test ends.
	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_USER", "test")
	t.Setenv("SMTP_PASSWORD", "test")
	t.Setenv("SMTP_FROM", "test@example.com")

	tests := []struct {
		name    string
		to      string
		wantErr bool
	}{
		{
			name:    "Valid Email",
			to:      "valid@example.com",
			wantErr: true, // Will still fail because no actual SMTP server is listening, but we expect an SMTP failure, not a validation failure. Let's make it more precise below.
		},
		{
			name:    "Header Injection",
			to:      "valid@example.com\r\nBcc: evil@example.com",
			wantErr: true, // Should fail at validation.
		},
		{
			name:    "Invalid Email Format",
			to:      "not an email",
			wantErr: true, // Should fail at validation.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := MailRequest{
				To:      tt.to,
				Subject: "Test Subject",
				Body:    "Test Body",
			}
			err := SendEmail(req)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.name != "Valid Email" {
					assert.Contains(t, err.Error(), "invalid recipient email address")
					assert.False(t, errors.Is(err, ErrSMTPVersand),
						"eine unbrauchbare Adresse ist kein Fehler des Zielservers — sonst antwortet der Handler mit 502 statt 400")
				} else {
					// Marker statt Meldungstext: Daran entscheidet der Handler zwischen
					// 502 (Zielserver versagt, Meldung wird durchgereicht) und 500.
					assert.ErrorIs(t, err, ErrSMTPVersand)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Ein unlesbares SMTP-Passwort muss auffallen.
//
// LadeSMTPKonfig erzeugt diesen Fehler ausdrücklich, damit er NICHT still bleibt
// („Ein unlesbares Passwort heißt, dass der APP_ENCRYPTION_KEY nicht mehr derselbe
// ist"). smtpKonfiguriert verschluckte ihn bis zum 31.08.2026 und meldete schlicht
// „nicht konfiguriert" — nach einer Schlüsselrotation suchte der Admin in den
// Einstellungen, wo Host, Port und Benutzer korrekt standen.
func TestSmtpKonfiguriert_LoggtUnlesbareKonfiguration(t *testing.T) {
	alt := smtpKonfigLader
	t.Cleanup(func() { smtpKonfigLader = alt })

	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })

	smtpKonfigLader = func() (SMTPKonfig, error) {
		return SMTPKonfig{}, errors.New("fehler beim Entschlüsseln des SMTP-Passworts")
	}
	if SMTPKonfiguriert() {
		t.Error("unlesbare Konfiguration gilt als konfiguriert")
	}
	if !strings.Contains(protokoll.String(), "nicht lesbar") {
		t.Errorf("kein Hinweis im Protokoll — der Grund bleibt unsichtbar: %q", protokoll.String())
	}
}

// Die Naht für Tests setzt den vorigen Lader wieder ein: Sonst läse der nächste Test die
// Konfiguration eines fremden.
func TestTauscheKonfigLaderFuerTest_SetztDenVorigenWiederEin(t *testing.T) {
	lader := func(host string) func() (SMTPKonfig, error) {
		return func() (SMTPKonfig, error) { return SMTPKonfig{Host: host}, nil }
	}
	host := func() string {
		t.Helper()
		konfig, err := VersandKonfig()
		if err != nil {
			t.Fatalf("VersandKonfig: %v", err)
		}
		return konfig.Host
	}
	t.Cleanup(TauscheKonfigLaderFuerTest(lader("erster.invalid")))
	zurueck := TauscheKonfigLaderFuerTest(lader("zweiter.invalid"))
	if got := host(); got != "zweiter.invalid" {
		t.Fatalf("nach dem Tausch gilt %q, erwartet der zweite Lader", got)
	}
	zurueck()
	if got := host(); got != "erster.invalid" {
		t.Errorf("nach dem Zurücksetzen gilt %q, erwartet der vorige Lader", got)
	}
}
