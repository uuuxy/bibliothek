package db

// Die Übernahme entscheidet, ob auf dem Schulserver nach dem Umstieg noch Mahnungen
// ankommen: In der Datenbank steht dort die Schema-Vorgabe (localhost:1025), die
// echten Zugangsdaten liegen in der .env. Genau einmal übernehmen — und eine vom
// Admin gespeicherte Konfiguration niemals überschreiben.

import (
	"context"
	"testing"

	"bibliothek/internal/crypto"

	"github.com/pashagolub/pgxmock/v4"
)

const testSchluessel = "0123456789abcdef0123456789abcdef"

func bootstrapPool(t *testing.T) (*Database, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	return &Database{Pool: mock}, mock
}

func gespeicherteZeile(host, port, benutzer string, passwort []byte) *pgxmock.Rows {
	return pgxmock.NewRows([]string{"smtp_host", "smtp_port", "smtp_user", "smtp_password_encrypted"}).
		AddRow(host, port, benutzer, passwort)
}

func TestInitMailKonfigUebernimmtUmgebungBeiUnangetasteterVorgabe(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", testSchluessel)
	t.Setenv("SMTP_HOST", "smtp.schule.de")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "bibliothek")
	t.Setenv("SMTP_PASSWORD", "geheim123")
	t.Setenv("SMTP_FROM", "bibliothek@schule.de")

	database, mock := bootstrapPool(t)
	mock.ExpectQuery(`SELECT smtp_host`).
		WillReturnRows(gespeicherteZeile(schemaVorgabeHost, schemaVorgabePort, "", nil))
	mock.ExpectExec(`UPDATE mail_settings_config`).
		WithArgs("smtp.schule.de", "587", "bibliothek", pgxmock.AnyArg(), "bibliothek@schule.de").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if err := database.InitMailKonfig(context.Background()); err != nil {
		t.Fatalf("InitMailKonfig: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Übernahme ist nicht passiert: %v", err)
	}
}

// Sobald in der Oberfläche gespeichert wurde, gewinnt diese Eingabe — sonst würde
// jeder Deploy die Konfiguration der Schule wieder überschreiben.
func TestInitMailKonfigLaesstGespeicherteKonfigurationInRuhe(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", testSchluessel)
	t.Setenv("SMTP_HOST", "smtp.aus-der-umgebung.de")

	verschluesselt, err := crypto.Encrypt([]byte("egal"))
	if err != nil {
		t.Fatalf("Verschlüsseln: %v", err)
	}

	for name, zeile := range map[string]*pgxmock.Rows{
		"anderer Host":     gespeicherteZeile("smtp.schule.de", "587", "", nil),
		"anderer Port":     gespeicherteZeile(schemaVorgabeHost, "2525", "", nil),
		"Benutzer gesetzt": gespeicherteZeile(schemaVorgabeHost, schemaVorgabePort, "bib", nil),
		"Passwort gesetzt": gespeicherteZeile(schemaVorgabeHost, schemaVorgabePort, "", verschluesselt),
	} {
		t.Run(name, func(t *testing.T) {
			database, mock := bootstrapPool(t)
			mock.ExpectQuery(`SELECT smtp_host`).WillReturnRows(zeile)
			// Kein ExpectExec: Ein UPDATE wäre ein unerwarteter Aufruf und ließe den
			// Test scheitern — genau die Zusicherung, um die es hier geht.

			if err := database.InitMailKonfig(context.Background()); err != nil {
				t.Fatalf("InitMailKonfig: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Erwartungen nicht erfüllt: %v", err)
			}
		})
	}
}

// Ohne SMTP_HOST in der Umgebung gibt es nichts zu übernehmen — und die Datenbank
// wird nicht einmal gelesen.
func TestInitMailKonfigOhneUmgebungTutNichts(t *testing.T) {
	t.Setenv("SMTP_HOST", "")

	database, mock := bootstrapPool(t)

	if err := database.InitMailKonfig(context.Background()); err != nil {
		t.Fatalf("InitMailKonfig: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Datenbank wurde unnötig berührt: %v", err)
	}
}
