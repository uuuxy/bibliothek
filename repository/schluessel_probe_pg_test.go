package repository

import (
	"context"
	"testing"

	"bibliothek/internal/crypto"
	"bibliothek/internal/pgtest"
)

// Der Wächter „passt der Schlüssel zum Bestand?" — mit einem echten verschlüsselten Wert
// und einem echten zweiten Schlüssel. Nachgestellt wird die Wiederherstellung auf einem
// neuen Server: Die Daten kommen aus der Sicherung, die .env ist neu.
func TestPruefeSchluesselGegenBestand(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	const (
		schluesselDerAnlage = "0123456789abcdef0123456789abcdef"
		neuerSchluessel     = "fedcba9876543210fedcba9876543210"
	)
	setzePasswort := func(wert []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO mail_settings_config (id, smtp_password_encrypted) VALUES (1, $1)
			ON CONFLICT (id) DO UPDATE SET smtp_password_encrypted = EXCLUDED.smtp_password_encrypted`,
			wert); err != nil {
			t.Fatalf("SMTP-Passwort ablegen: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `TRUNCATE schueler_fotos`); err != nil {
		t.Fatalf("Fotos räumen: %v", err)
	}
	t.Cleanup(func() { setzePasswort(nil) })

	// Nichts verschlüsselt abgelegt: nichts zu prüfen, nichts zu verfehlen.
	setzePasswort(nil)
	t.Setenv(crypto.SchluesselVariable, schluesselDerAnlage)
	probe, err := PruefeSchluesselGegenBestand(ctx, pool)
	if err != nil {
		t.Fatalf("Probe ohne Bestand: %v", err)
	}
	if probe.Geprueft != 0 || len(probe.NichtLesbar) != 0 {
		t.Errorf("leerer Bestand: geprüft=%d, nicht lesbar=%v — erwartet 0 und nichts", probe.Geprueft, probe.NichtLesbar)
	}

	// Mit dem Schlüssel der Anlage abgelegt und gelesen: passt.
	verschluesselt, err := crypto.Encrypt([]byte("smtp-geheim"))
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}
	setzePasswort(verschluesselt)
	probe, err = PruefeSchluesselGegenBestand(ctx, pool)
	if err != nil {
		t.Fatalf("Probe mit passendem Schlüssel: %v", err)
	}
	if probe.Geprueft != 1 || len(probe.NichtLesbar) != 0 {
		t.Errorf("passender Schlüssel: geprüft=%d, nicht lesbar=%v — erwartet 1 und nichts", probe.Geprueft, probe.NichtLesbar)
	}

	// Neue .env, alter Bestand: Der Wächter nennt die Spalte.
	t.Setenv(crypto.SchluesselVariable, neuerSchluessel)
	probe, err = PruefeSchluesselGegenBestand(ctx, pool)
	if err != nil {
		t.Fatalf("Probe mit fremdem Schlüssel: %v", err)
	}
	if len(probe.NichtLesbar) != 1 || probe.NichtLesbar[0] != "SMTP-Passwort" {
		t.Errorf("fremder Schlüssel: nicht lesbar=%v — erwartet [SMTP-Passwort]", probe.NichtLesbar)
	}
}
