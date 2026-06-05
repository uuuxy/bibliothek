package inventur

import (
	"os"
	"testing"
)

func TestNewBackupEmailConfigFromEnv(t *testing.T) {
	// Save existing environment variables to restore them later
	originalHost := os.Getenv("SMTP_HOST")
	originalPort := os.Getenv("SMTP_PORT")
	originalUser := os.Getenv("SMTP_USER")
	originalPassword := os.Getenv("SMTP_PASSWORD")
	originalTo := os.Getenv("BACKUP_EMAIL_TO")

	defer func() {
		os.Setenv("SMTP_HOST", originalHost)
		os.Setenv("SMTP_PORT", originalPort)
		os.Setenv("SMTP_USER", originalUser)
		os.Setenv("SMTP_PASSWORD", originalPassword)
		os.Setenv("BACKUP_EMAIL_TO", originalTo)
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		wantHost string
		wantPort string
		wantUser string
		wantPass string
		wantTo   string
		wantNil  bool
	}{
		{
			name:    "Missing Host",
			envVars: map[string]string{},
			wantNil: true,
		},
		{
			name: "Missing Port (defaults to 587)",
			envVars: map[string]string{
				"SMTP_HOST":       "smtp.example.com",
				"SMTP_USER":       "user1",
				"SMTP_PASSWORD":   "pass1",
				"BACKUP_EMAIL_TO": "to@example.com",
			},
			wantHost: "smtp.example.com",
			wantPort: "587",
			wantUser: "user1",
			wantPass: "pass1",
			wantTo:   "to@example.com",
			wantNil:  false,
		},
		{
			name: "Full Configuration",
			envVars: map[string]string{
				"SMTP_HOST":       "smtp.anotherexample.com",
				"SMTP_PORT":       "465",
				"SMTP_USER":       "user2",
				"SMTP_PASSWORD":   "pass2",
				"BACKUP_EMAIL_TO": "to2@example.com",
			},
			wantHost: "smtp.anotherexample.com",
			wantPort: "465",
			wantUser: "user2",
			wantPass: "pass2",
			wantTo:   "to2@example.com",
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear specific env vars
			os.Unsetenv("SMTP_HOST")
			os.Unsetenv("SMTP_PORT")
			os.Unsetenv("SMTP_USER")
			os.Unsetenv("SMTP_PASSWORD")
			os.Unsetenv("BACKUP_EMAIL_TO")

			// Set test-specific env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Run the function
			got := NewBackupEmailConfigFromEnv()

			// Assert results
			if tt.wantNil {
				if got != nil {
					t.Errorf("NewBackupEmailConfigFromEnv() = %v, want nil", got)
				}
				return // we are done if we wanted nil
			}

			if got == nil {
				t.Fatalf("NewBackupEmailConfigFromEnv() returned nil, want config")
			}

			if got.Host != tt.wantHost {
				t.Errorf("got Host = %q, want %q", got.Host, tt.wantHost)
			}
			if got.Port != tt.wantPort {
				t.Errorf("got Port = %q, want %q", got.Port, tt.wantPort)
			}
			if got.User != tt.wantUser {
				t.Errorf("got User = %q, want %q", got.User, tt.wantUser)
			}
			if got.Password != tt.wantPass {
				t.Errorf("got Password = %q, want %q", got.Password, tt.wantPass)
			}
			if got.To != tt.wantTo {
				t.Errorf("got To = %q, want %q", got.To, tt.wantTo)
			}
		})
	}
}
