package repository

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/internal/crypto"
)

// MailEinstellungen sind die Zugangsdaten zum Mailserver, wie sie in der Datenbank stehen. Die
// Datenbank ist die einzige Quelle dafür; die Datei .env füllt sie nur beim ersten Start. Wer
// einem Mailproblem nachgeht, liest deshalb diese Zeile und nicht die Umgebungsvariablen.
type MailEinstellungen struct {
	SMTPHost                   string
	SMTPPort                   string
	SMTPBenutzer               string
	SMTPPasswortVerschluesselt []byte
	AbsenderEmail              string
}

// MailEinstellungsAbfragen liest und schreibt die Mail-Einstellungen.
type MailEinstellungsAbfragen struct {
	pool db.PgxPoolIface
}

// NewMailEinstellungsAbfragen liefert die Abfragen der Mail-Einstellungen am übergebenen Pool.
func NewMailEinstellungsAbfragen(pool db.PgxPoolIface) *MailEinstellungsAbfragen {
	return &MailEinstellungsAbfragen{pool: pool}
}

// Lade lädt die Mail-Einstellungen. Es gibt genau eine Zeile, die mit der ID 1.
func (a *MailEinstellungsAbfragen) Lade(ctx context.Context) (*MailEinstellungen, error) {
	var e MailEinstellungen
	err := a.pool.QueryRow(ctx, `
		SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email
		FROM mail_settings_config
		WHERE id = 1
	`).Scan(
		&e.SMTPHost,
		&e.SMTPPort,
		&e.SMTPBenutzer,
		&e.SMTPPasswortVerschluesselt,
		&e.AbsenderEmail,
	)

	if err != nil {
		return nil, fmt.Errorf("fehler beim Laden der Mail-Einstellungen: %w", err)
	}

	return &e, nil
}

// Schreibe schreibt die Mail-Einstellungen. Ein Passwort wird verschlüsselt gespeichert; ein
// leeres lässt das gespeicherte stehen, damit die Maske es nicht zurückschicken muss.
func (a *MailEinstellungsAbfragen) Schreibe(ctx context.Context, host, port, benutzer, passwort, absender string) error {
	if passwort != "" {
		verschluesselt, err := crypto.Verschluessele([]byte(passwort))
		if err != nil {
			return fmt.Errorf("fehler beim Verschlüsseln des Passworts: %w", err)
		}

		_, err = a.pool.Exec(ctx, `
			INSERT INTO mail_settings_config (id, smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email)
			VALUES (1, $1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				smtp_host = EXCLUDED.smtp_host,
				smtp_port = EXCLUDED.smtp_port,
				smtp_user = EXCLUDED.smtp_user,
				smtp_password_encrypted = EXCLUDED.smtp_password_encrypted,
				sender_email = EXCLUDED.sender_email
		`, host, port, benutzer, verschluesselt, absender)

		return err
	}

	// Ohne Passwort bleibt die Spalte aus der Anweisung heraus.
	_, err := a.pool.Exec(ctx, `
		INSERT INTO mail_settings_config (id, smtp_host, smtp_port, smtp_user, sender_email)
		VALUES (1, $1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			smtp_host = EXCLUDED.smtp_host,
			smtp_port = EXCLUDED.smtp_port,
			smtp_user = EXCLUDED.smtp_user,
			sender_email = EXCLUDED.sender_email
	`, host, port, benutzer, absender)

	return err
}
