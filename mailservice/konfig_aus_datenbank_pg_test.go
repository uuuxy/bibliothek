package mailservice

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Das Lesen der Konfiguration erbt die Werte der Anfrage, nicht ihr Ende. Endete es mit ihr,
// scheiterte das Lesen, und ladeSMTPKonfig wiche still auf die Umgebung aus: Der Rest eines
// Sammelversands ginge an einen anderen Server.
func TestKonfigAusDatenbank_NachDemEndeDerAnfrage(t *testing.T) {
	t.Setenv("SMTP_HOST", "umgebung.invalid")
	pool := pgtest.Pool(t)
	ctx := context.Background()
	if err := repository.NewMailEinstellungsAbfragen(pool).Schreibe(ctx, "gespeichert.example", "587", "", "",
		"bibliothek@schule.example"); err != nil {
		t.Fatalf("Konfiguration speichern: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			UPDATE mail_settings_config
			SET smtp_host = 'localhost', smtp_port = '1025', smtp_user = '',
			    smtp_password_encrypted = NULL, sender_email = 'noreply@bibliothek-schule.de'
			WHERE id = 1`); err != nil {
			t.Errorf("Mail-Zeile zurücksetzen: %v", err)
		}
	})

	beendet, beende := context.WithCancel(ctx)
	beende()
	konfig, err := KonfigAusDatenbank(pool)(beendet)
	if err != nil {
		t.Fatalf("KonfigAusDatenbank(pool)(beendete Anfrage) = %v, erwartet die Konfiguration", err)
	}
	if konfig.Host != "gespeichert.example" {
		t.Errorf("KonfigAusDatenbank(pool)(beendete Anfrage).Host = %q, erwartet %q aus der Datenbank",
			konfig.Host, "gespeichert.example")
	}
}
