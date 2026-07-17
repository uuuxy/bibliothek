package inventur

import (
	"testing"
)

func TestNewBackupEmailConfigFromEnv(t *testing.T) {
	t.Run("EmptyHost_ReturnsNil", func(t *testing.T) {
		t.Setenv("SMTP_HOST", "")

		config := NewBackupEmailConfigFromEnv()
		if config != nil {
			t.Errorf("expected nil, got %v", config)
		}
	})

	t.Run("HostSetPortEmpty_DefaultsTo587", func(t *testing.T) {
		t.Setenv("SMTP_HOST", "smtp.example.com")
		t.Setenv("SMTP_PORT", "")
		t.Setenv("SMTP_USER", "user@example.com")
		t.Setenv("SMTP_PASSWORD", "secret")
		t.Setenv("BACKUP_EMAIL_TO", "backup@example.com")

		config := NewBackupEmailConfigFromEnv()
		if config == nil {
			t.Fatal("expected config, got nil")
		}

		if config.Host != "smtp.example.com" {
			t.Errorf("expected host 'smtp.example.com', got %q", config.Host)
		}
		if config.Port != "587" {
			t.Errorf("expected port '587', got %q", config.Port)
		}
		if config.User != "user@example.com" {
			t.Errorf("expected user 'user@example.com', got %q", config.User)
		}
		if config.Password != "secret" {
			t.Errorf("expected password 'secret', got %q", config.Password)
		}
		if config.To != "backup@example.com" {
			t.Errorf("expected to 'backup@example.com', got %q", config.To)
		}
	})

	t.Run("AllEnvVarsSet_ReturnsConfig", func(t *testing.T) {
		t.Setenv("SMTP_HOST", "smtp.example.com")
		t.Setenv("SMTP_PORT", "465")
		t.Setenv("SMTP_USER", "user@example.com")
		t.Setenv("SMTP_PASSWORD", "secret")
		t.Setenv("BACKUP_EMAIL_TO", "backup@example.com")

		config := NewBackupEmailConfigFromEnv()
		if config == nil {
			t.Fatal("expected config, got nil")
		}

		if config.Host != "smtp.example.com" {
			t.Errorf("expected host 'smtp.example.com', got %q", config.Host)
		}
		if config.Port != "465" {
			t.Errorf("expected port '465', got %q", config.Port)
		}
		if config.User != "user@example.com" {
			t.Errorf("expected user 'user@example.com', got %q", config.User)
		}
		if config.Password != "secret" {
			t.Errorf("expected password 'secret', got %q", config.Password)
		}
		if config.To != "backup@example.com" {
			t.Errorf("expected to 'backup@example.com', got %q", config.To)
		}
	})
}
