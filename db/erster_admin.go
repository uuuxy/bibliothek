package db

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
)

// Ohne Ausweisnummer: Sie gehört zur Leserzeile, die der Trigger trg_benutzer_hat_leserzeile mit
// anlegt. Die Nummer vergibt trg_aktives_konto_hat_ausweis mit dem Commit.
const sqlErsterAdmin = `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('System', 'Administrator', $1, 'admin', true)
		RETURNING id
	`

// LegeErstenAdminAn legt das erste Konto an, wenn es noch keines gibt: einen Admin mit der
// Adresse aus INITIAL_ADMIN_EMAIL. Ein Passwort braucht es nicht; angemeldet wird über den
// Mailserver der Schule (IMAP).
func (v *Verbindung) LegeErstenAdminAn(ctx context.Context) error {
	var vorhanden bool
	err := v.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer)").Scan(&vorhanden)
	if err != nil {
		return fmt.Errorf("failed to query benutzer count: %w", err)
	}

	if vorhanden {
		return nil // Es gibt schon Konten.
	}

	// Gespeichert wird die Adresse, wie die Anmeldung sie sucht und wie die übrigen Wege ein
	// Konto anlegen (repository.LegeKollegiumskontoAn): klein und ohne Leerraum.
	email := strings.ToLower(strings.TrimSpace(os.Getenv("INITIAL_ADMIN_EMAIL")))

	if email == "" {
		log.Println("Warnung: Keine Benutzer in der Datenbank und INITIAL_ADMIN_EMAIL nicht gesetzt. System startet ohne Admin-Zugang.")
		return nil
	}
	// Ein Wert ohne die Form einer Adresse ergäbe ein Konto, in das niemand kommt, und nach
	// diesem Start wirkt die Variable nicht mehr. Der Abbruch lässt die Tabelle leer.
	if adresse, err := mail.ParseAddress(email); err != nil || adresse.Address != email {
		return fmt.Errorf("INITIAL_ADMIN_EMAIL ist keine E-Mail-Adresse: %q", email)
	}

	var adminID string
	if err := v.Pool.QueryRow(ctx, sqlErsterAdmin, email).Scan(&adminID); err != nil {
		return fmt.Errorf("failed to insert initial admin: %w", err)
	}

	log.Printf("Erster Admin-Benutzer (%s) wurde erfolgreich initialisiert.", email)
	return nil
}
