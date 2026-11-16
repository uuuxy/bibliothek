package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"strings"
)

// Ohne Ausweisnummer: Sie gehört zur Leserzeile, die der Trigger trg_benutzer_hat_leserzeile mit
// anlegt. Die Nummer vergibt trg_aktives_konto_hat_ausweis mit dem Commit.
const sqlErsterAdmin = `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, anmeldeverfahren, ldap_kennung)
		VALUES ('System', 'Administrator', $1, 'admin', true, $2, NULLIF($3, ''))
		RETURNING id
	`

// LegeErstenAdminAn legt das erste Konto an, wenn es noch keines gibt: einen Admin mit der
// Adresse aus INITIAL_ADMIN_EMAIL. Ein Passwort braucht es nicht; es prüft der Mailserver der
// Schule (IMAP) oder, mit INITIAL_ADMIN_LDAP_KENNUNG, der Verzeichnisdienst (LDAP).
func (v *Verbindung) LegeErstenAdminAn(ctx context.Context) error {
	var anzahl int
	err := v.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM benutzer").Scan(&anzahl)
	if err != nil {
		return fmt.Errorf("failed to query benutzer count: %w", err)
	}

	if anzahl > 0 {
		return nil // Es gibt schon Konten.
	}

	// Gespeichert wird die Adresse, wie die Anmeldung sie sucht und wie die übrigen Wege ein
	// Konto anlegen (repository.LegeKollegiumskontoAn): klein und ohne Leerraum.
	email := strings.ToLower(strings.TrimSpace(os.Getenv("INITIAL_ADMIN_EMAIL")))

	// Ohne Admin füllte die erste Selbstanmeldung die Tabelle mit einem inaktiven Konto, das
	// niemand freischalten kann, und die Variable wirkte danach nicht mehr.
	if email == "" {
		return errors.New("INITIAL_ADMIN_EMAIL ist nicht gesetzt — der erste Start braucht die Adresse des ersten Admins in der .env")
	}
	// Ein Wert ohne die Form einer Adresse ergäbe ein Konto, in das niemand kommt, und nach
	// diesem Start wirkt die Variable nicht mehr. Der Abbruch lässt die Tabelle leer.
	if adresse, err := mail.ParseAddress(email); err != nil || adresse.Address != email {
		return fmt.Errorf("INITIAL_ADMIN_EMAIL ist keine E-Mail-Adresse: %q", email)
	}

	kennung := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_LDAP_KENNUNG"))
	verfahren := "imap"
	if kennung != "" {
		verfahren = "ldap"
	}
	// Ohne Mailserver käme ein IMAP-Admin nicht hinein, und nach diesem Start wirken die
	// Variablen nicht mehr.
	if verfahren == "imap" && strings.TrimSpace(os.Getenv("IMAP_HOST")) == "" {
		return errors.New("IMAP_HOST ist nicht gesetzt — ohne Mailserver braucht der erste Admin INITIAL_ADMIN_LDAP_KENNUNG")
	}

	var adminID string
	if err := v.Pool.QueryRow(ctx, sqlErsterAdmin, email, verfahren, kennung).Scan(&adminID); err != nil {
		return fmt.Errorf("failed to insert initial admin: %w", err)
	}

	slog.Info("Erster Admin-Benutzer wurde erfolgreich initialisiert.", "email", email, "anmeldeverfahren", verfahren)
	return nil
}
