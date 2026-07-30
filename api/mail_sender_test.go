package api

import (
	"errors"
	"net/http"
	"testing"

	"bibliothek/mailservice"

	"github.com/stretchr/testify/assert"
)

// mailFehlerStatus ist die Weiche, an der die SMTP-Diagnose überlebt oder stirbt:
// apierrors ersetzt bei 500 jede Meldung durch "Ein interner Datenbankfehler ist
// aufgetreten" und reicht sie erst unterhalb von 500 durch.
func TestMailFehlerStatus(t *testing.T) {
	smtpFehler := mailservice.BeschreibeSMTPFehler("smtp.schule.de:587", errors.New("535 5.7.8 Zugangsdaten abgelehnt"))
	assert.Equal(t, http.StatusBadGateway, mailFehlerStatus(smtpFehler),
		"Zielserver versagt → 502, damit die Meldung beim Admin ankommt")

	assert.Equal(t, http.StatusInternalServerError, mailFehlerStatus(errors.New("vorlage nicht gefunden")),
		"eigene Störung bleibt 500 und wird bewusst neutralisiert")
}

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
					assert.False(t, errors.Is(err, mailservice.ErrSMTPVersand),
						"eine unbrauchbare Adresse ist kein Fehler des Zielservers — sonst antwortet der Handler mit 502 statt 400")
				} else {
					// Marker statt Meldungstext: Daran entscheidet der Handler zwischen
					// 502 (Zielserver versagt, Meldung wird durchgereicht) und 500.
					assert.ErrorIs(t, err, mailservice.ErrSMTPVersand)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
