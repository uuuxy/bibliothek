package api

import (
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
				} else {
					assert.Contains(t, err.Error(), "SMTP send failed")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
