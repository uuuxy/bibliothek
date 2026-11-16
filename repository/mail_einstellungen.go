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
	// Der zweite Absender neben dem Postausgang (Spalten portal_*). Eine leere Adresse heißt,
	// alle Mails gehen über den Postausgang; ein leerer Benutzer, der Zugang darüber gilt auch
	// für diese Adresse. PortalAbsenderFuer nennt die Arten von Mails, die über ihn gehen
	// (mailservice.Mailart).
	PortalAbsenderEmail              string
	PortalSMTPBenutzer               string
	PortalSMTPPasswortVerschluesselt []byte
	PortalAbsenderFuer               []string
	// Der Zugang zum Webmail für den Adressabgleich; eine leere Adresse heißt, es gibt keinen.
	WebmailAdresse                string
	WebmailBenutzer               string
	WebmailPasswortVerschluesselt []byte
}

// MailEinstellungsAbfragen liest und schreibt die Mail-Einstellungen.
type MailEinstellungsAbfragen struct {
	pool db.Pool
}

// NewMailEinstellungsAbfragen liefert die Abfragen der Mail-Einstellungen am übergebenen Pool.
func NewMailEinstellungsAbfragen(pool db.Pool) *MailEinstellungsAbfragen {
	return &MailEinstellungsAbfragen{pool: pool}
}

// Lade lädt die Mail-Einstellungen. Es gibt genau eine Zeile, die mit der ID 1.
func (a *MailEinstellungsAbfragen) Lade(ctx context.Context) (*MailEinstellungen, error) {
	var e MailEinstellungen
	err := a.pool.QueryRow(ctx, `
		SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted, sender_email,
		       portal_absender_email, portal_smtp_benutzer, portal_smtp_passwort_verschluesselt,
		       portal_absender_fuer, webmail_adresse, webmail_benutzer, webmail_passwort_verschluesselt
		FROM mail_settings_config
		WHERE id = 1
	`).Scan(
		&e.SMTPHost,
		&e.SMTPPort,
		&e.SMTPBenutzer,
		&e.SMTPPasswortVerschluesselt,
		&e.AbsenderEmail,
		&e.PortalAbsenderEmail,
		&e.PortalSMTPBenutzer,
		&e.PortalSMTPPasswortVerschluesselt,
		&e.PortalAbsenderFuer,
		&e.WebmailAdresse,
		&e.WebmailBenutzer,
		&e.WebmailPasswortVerschluesselt,
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

// SchreibePortalAbsender schreibt den zweiten Absender in die Zeile, die Schreibe angelegt hat.
// Ein leeres Passwort lässt das gespeicherte stehen, wie bei Schreibe; fuer nil lässt die Arten
// von Mails stehen, die über ihn gehen, eine leere Liste nimmt alle heraus.
func (a *MailEinstellungsAbfragen) SchreibePortalAbsender(ctx context.Context, absender, benutzer, passwort string, fuer []string) error {
	var verschluesselt []byte // nil wird NULL, und COALESCE behält das gespeicherte Passwort
	if passwort != "" {
		var err error
		verschluesselt, err = crypto.Verschluessele([]byte(passwort))
		if err != nil {
			return fmt.Errorf("fehler beim Verschlüsseln des Passworts: %w", err)
		}
	}
	tag, err := a.pool.Exec(ctx, `
		UPDATE mail_settings_config SET
			portal_absender_email = $1,
			portal_smtp_benutzer = $2,
			portal_smtp_passwort_verschluesselt = COALESCE($3, portal_smtp_passwort_verschluesselt),
			portal_absender_fuer = COALESCE($4::text[], portal_absender_fuer)
		WHERE id = 1
	`, absender, benutzer, verschluesselt, fuer)
	if err != nil {
		return fmt.Errorf("fehler beim Speichern des zweiten Absenders: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("zweiter Absender nicht gespeichert: Die Zeile der Mail-Einstellungen fehlt")
	}
	return nil
}

// SchreibeWebmailZugang schreibt den Zugang zum Webmail für den Adressabgleich. Ein leeres
// Passwort lässt das gespeicherte stehen, wie bei Schreibe; eine leere Adresse schaltet den
// Abgleich ab und nimmt das Passwort mit.
func (a *MailEinstellungsAbfragen) SchreibeWebmailZugang(ctx context.Context, adresse, benutzer, passwort string) error {
	var verschluesselt []byte // nil wird NULL, und COALESCE behält das gespeicherte Passwort
	if passwort != "" {
		var err error
		verschluesselt, err = crypto.Verschluessele([]byte(passwort))
		if err != nil {
			return fmt.Errorf("fehler beim Verschlüsseln des Passworts: %w", err)
		}
	}
	tag, err := a.pool.Exec(ctx, `
		UPDATE mail_settings_config SET
			webmail_adresse = $1::text,
			webmail_benutzer = $2,
			webmail_passwort_verschluesselt = CASE WHEN $1::text = '' THEN NULL
				ELSE COALESCE($3, webmail_passwort_verschluesselt) END
		WHERE id = 1
	`, adresse, benutzer, verschluesselt)
	if err != nil {
		return fmt.Errorf("fehler beim Speichern des Webmail-Zugangs: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("webmail-Zugang nicht gespeichert: Die Zeile der Mail-Einstellungen fehlt")
	}
	return nil
}
