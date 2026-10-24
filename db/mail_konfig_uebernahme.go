package db

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"bibliothek/internal/crypto"
)

// schemaVorgabeHost und schemaVorgabePort sind die Werte, mit denen schema.sql die Zeile
// anlegt. Sie zeigen, dass dort noch nie jemand etwas eingetragen hat; eine eigene Spalte dafür
// wäre mehr Schema für eine einmalige Frage.
const (
	schemaVorgabeHost = "localhost"
	schemaVorgabePort = "1025"
)

// UebernimmMailKonfig übernimmt SMTP_HOST/PORT/USER/PASSWORD/FROM aus der Umgebung in die
// gespeicherte Mail-Konfiguration, solange dort die unangetastete Vorgabe des Schemas steht.
//
// Versendet wird mit dem, was in der Datenbank steht. Eine Anlage, deren Zugangsdaten noch in
// der .env liegen, verschickte sonst nach dem Update an localhost; deshalb übernimmt der Start
// sie, bevor etwas versendet wird. Hat jemand die Einstellungen in der Oberfläche gespeichert,
// gilt seine Eingabe, und ein Update überschreibt sie nicht.
func (v *Verbindung) UebernimmMailKonfig(ctx context.Context) error {
	umgebungHost := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if umgebungHost == "" {
		return nil // nichts zu übernehmen
	}

	var host, port, benutzer string
	var passwortVerschluesselt []byte
	err := v.Pool.QueryRow(ctx, `
		SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted
		FROM mail_settings_config WHERE id = 1
	`).Scan(&host, &port, &benutzer, &passwortVerschluesselt)
	if err != nil {
		// Ohne lesbare Zeile fällt der Versand auf die Umgebung zurück
		// (mailservice.LadeSMTPKonfig); das ist kein Grund, den Start abzubrechen.
		slog.Error("mail-konfig: gespeicherte Konfiguration nicht lesbar, Übernahme übersprungen", "fehler", err)
		return nil
	}

	bereitsKonfiguriert := host != schemaVorgabeHost ||
		port != schemaVorgabePort ||
		benutzer != "" ||
		len(passwortVerschluesselt) > 0
	if bereitsKonfiguriert {
		return nil
	}

	passwort := os.Getenv("SMTP_PASSWORD")
	if passwort == "" {
		passwort = os.Getenv("SMTP_PASS")
	}
	var verschluesselt []byte
	if passwort != "" {
		verschluesselt, err = crypto.Verschluessele([]byte(passwort))
		if err != nil {
			return err
		}
	}

	umgebungPort := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if umgebungPort == "" {
		umgebungPort = "587"
	}
	absender := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if absender == "" {
		absender = strings.TrimSpace(os.Getenv("SMTP_USER"))
	}

	tag, err := v.Pool.Exec(ctx, `
		UPDATE mail_settings_config
		SET smtp_host = $1, smtp_port = $2, smtp_user = $3,
		    smtp_password_encrypted = COALESCE($4, smtp_password_encrypted),
		    sender_email = COALESCE(NULLIF($5, ''), sender_email)
		WHERE id = 1
	`, umgebungHost, umgebungPort, os.Getenv("SMTP_USER"), verschluesselt, absender)
	if err != nil {
		return err
	}
	// Die Zeile wurde oben gerade gelesen; keine geänderte Zeile heißt, sie verschwand
	// währenddessen. Dann darf nicht „übernommen" im Log stehen, und der Versand fällt wie beim
	// Lesefehler auf die Umgebung zurück.
	if tag.RowsAffected() == 0 {
		slog.Warn("mail-konfig: Konfigurationszeile verschwand während der Übernahme — nichts übernommen")
		return nil
	}

	slog.Info("mail-konfig: SMTP-Zugangsdaten aus der Umgebung übernommen — ab jetzt gilt die Einstellung in der Oberfläche", "host", umgebungHost, "port", umgebungPort)
	return nil
}
