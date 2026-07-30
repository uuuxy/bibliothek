package db

// mail_konfig_bootstrap.go — Einmalige Übernahme der SMTP-Zugangsdaten aus der
// Umgebung in die Datenbank.
//
// Hintergrund: Die Oberfläche schreibt die Mail-Konfiguration nach
// mail_settings_config, jeder Versand las bis Juli 2026 aber die Umgebungsvariablen
// des Containers. Auf dem Schulserver steht deshalb in der Datenbank noch die
// Schema-Vorgabe (localhost:1025), während die echten Zugangsdaten in der .env
// liegen. Würde der Versand einfach auf die Datenbank umgestellt, gingen die
// Mahnungen ab dem nächsten Deploy an localhost — also wird die Umgebung beim Start
// übernommen, bevor irgendetwas versendet wird.
//
// Danach ist die Datenbank die Wahrheit: Was im Formular steht, gilt, und eine
// Änderung wirkt ohne Neustart des Containers.

import (
	"context"
	"log"
	"os"
	"strings"

	"bibliothek/internal/crypto"
)

// schemaVorgabeHost/Port sind die Werte, mit denen schema.sql bzw. Migration 027 die
// Zeile anlegt. Sie sind der Fingerabdruck für "hier hat noch nie jemand etwas
// eingetragen" — eine eigene Spalte dafür wäre mehr Schema für eine einmalige Frage.
const (
	schemaVorgabeHost = "localhost"
	schemaVorgabePort = "1025"
)

// InitMailKonfig übernimmt SMTP_HOST/PORT/USER/PASSWORD/FROM in die gespeicherte
// Mail-Konfiguration, solange dort noch die unangetastete Schema-Vorgabe steht.
//
// Bewusst nur dann: Sobald jemand die Einstellungen in der Oberfläche gespeichert hat,
// gewinnt seine Eingabe — sonst würde ein Deploy die Konfiguration der Schule jedes
// Mal wieder überschreiben.
func (db *Database) InitMailKonfig(ctx context.Context) error {
	umgebungHost := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if umgebungHost == "" {
		return nil // nichts zu übernehmen
	}

	var host, port, benutzer string
	var passwortVerschluesselt []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT smtp_host, smtp_port, smtp_user, smtp_password_encrypted
		FROM mail_settings_config WHERE id = 1
	`).Scan(&host, &port, &benutzer, &passwortVerschluesselt)
	if err != nil {
		// Keine Zeile: Migration 027 ist noch nicht gelaufen. Der Versand fällt dann auf
		// die Umgebung zurück (mailservice.LadeSMTPKonfig) — kein Grund, den Start
		// abzubrechen.
		log.Printf("mail-konfig: gespeicherte Konfiguration nicht lesbar, Übernahme übersprungen: %v", err)
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
		verschluesselt, err = crypto.Encrypt([]byte(passwort))
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

	_, err = db.Pool.Exec(ctx, `
		UPDATE mail_settings_config
		SET smtp_host = $1, smtp_port = $2, smtp_user = $3,
		    smtp_password_encrypted = COALESCE($4, smtp_password_encrypted),
		    sender_email = COALESCE(NULLIF($5, ''), sender_email)
		WHERE id = 1
	`, umgebungHost, umgebungPort, os.Getenv("SMTP_USER"), verschluesselt, absender)
	if err != nil {
		return err
	}

	log.Printf("mail-konfig: SMTP-Zugangsdaten aus der Umgebung übernommen (%s:%s) — ab jetzt gilt die Einstellung in der Oberfläche", umgebungHost, umgebungPort)
	return nil
}
