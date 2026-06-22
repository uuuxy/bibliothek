package repository

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/internal/crypto"
)

type MailSettings struct {
	SMTPHost              string
	SMTPPort              string
	SMTPUser              string
	SMTPPasswordEncrypted []byte
	SenderEmail           string
}

type MailSettingsRepository struct {
	pool db.PgxPoolIface
}

func NewMailSettingsRepository(pool db.PgxPoolIface) *MailSettingsRepository {
	return &MailSettingsRepository{pool: pool}
}

// GetConfig lädt die Konfiguration aus der Datenbank (immer ID 1).
func (r *MailSettingsRepository) GetConfig(ctx context.Context) (*MailSettings, error) {
	var settings MailSettings
	err := r.pool.QueryRow(ctx, `
		SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email
		FROM mail_settings_config
		WHERE id = 1
	`).Scan(
		&settings.SMTPHost,
		&settings.SMTPPort,
		&settings.SMTPUser,
		&settings.SMTPPasswordEncrypted,
		&settings.SenderEmail,
	)

	if err != nil {
		return nil, fmt.Errorf("fehler beim Laden der Mail-Einstellungen: %w", err)
	}

	return &settings, nil
}

// UpdateConfig aktualisiert die Mail-Konfiguration.
// Falls smtpPassword nicht leer ist, wird es verschlüsselt und überschrieben.
// Ist es leer, bleibt das vorhandene Passwort unangetastet.
func (r *MailSettingsRepository) UpdateConfig(ctx context.Context, host, port, user, smtpPassword, sender string) error {
	if smtpPassword != "" {
		// Passwort verschlüsseln
		encrypted, err := crypto.Encrypt([]byte(smtpPassword))
		if err != nil {
			return fmt.Errorf("fehler beim Verschlüsseln des Passworts: %w", err)
		}

		_, err = r.pool.Exec(ctx, `
			INSERT INTO mail_settings_config (id, smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email)
			VALUES (1, $1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				smtp_host = EXCLUDED.smtp_host,
				smtp_port = EXCLUDED.smtp_port,
				smtp_user = EXCLUDED.smtp_user,
				smtp_password_encrypted = EXCLUDED.smtp_password_encrypted,
				sender_email = EXCLUDED.sender_email
		`, host, port, user, encrypted, sender)
		
		return err
	}

	// Falls kein Passwort mitgegeben wurde, belassen wir das bestehende Password
	_, err := r.pool.Exec(ctx, `
		INSERT INTO mail_settings_config (id, smtp_host, smtp_port, smtp_user, sender_email)
		VALUES (1, $1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			smtp_host = EXCLUDED.smtp_host,
			smtp_port = EXCLUDED.smtp_port,
			smtp_user = EXCLUDED.smtp_user,
			sender_email = EXCLUDED.sender_email
	`, host, port, user, sender)

	return err
}
