package repository

import (
	"bytes"
	"context"
	"testing"

	"bibliothek/internal/crypto"
	"bibliothek/internal/pgtest"
)

// Die Mail-Einstellungen (mail_settings_config, eine Zeile) tragen jeden Versand: Mahnungen,
// Bestellungen, Anmeldung. Geprüft wird der Rundweg, an dem ein Fehler still wäre: Das Passwort
// steht verschlüsselt in der Zeile und kommt beim Lesen unverändert heraus, und ein leeres
// Passwortfeld beim Speichern lässt das gespeicherte stehen, statt es zu löschen.

// testSchluessel ist ein Schlüssel nur für diesen Test (32 Zeichen, AES-256).
const testSchluessel = "mail-test-schluessel-32-zeichen!"

// setzeMailZeileZurueck stellt die eine Zeile auf ihre Vorgaben und nimmt das Passwort weg.
func setzeMailZeileZurueck(t *testing.T, ctx context.Context, repo *MailEinstellungsAbfragen) {
	t.Helper()
	if _, err := repo.pool.Exec(ctx, `
		UPDATE mail_settings_config
		SET smtp_host = 'localhost', smtp_port = '1025', smtp_user = '',
		    smtp_password_encrypted = NULL, sender_email = 'noreply@bibliothek-schule.de'
		WHERE id = 1`); err != nil {
		t.Fatalf("Mail-Zeile zurücksetzen: %v", err)
	}
}

func TestMailEinstellungen_PasswortImRundweg(t *testing.T) {
	t.Setenv(crypto.SchluesselVariable, testSchluessel)
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewMailEinstellungsAbfragen(pool)
	setzeMailZeileZurueck(t, ctx, repo)
	t.Cleanup(func() { setzeMailZeileZurueck(t, context.Background(), repo) })

	const passwort = "Geheim-SMTP-1!"
	if err := repo.Schreibe(ctx, "smtp.schule.example", "587", "bibliothek", passwort, "bibliothek@schule.example"); err != nil {
		t.Fatalf("speichern: %v", err)
	}

	stand, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if stand.SMTPHost != "smtp.schule.example" || stand.SMTPPort != "587" ||
		stand.SMTPBenutzer != "bibliothek" || stand.AbsenderEmail != "bibliothek@schule.example" {
		t.Errorf("gelesen: %+v, gespeichert war etwas anderes", stand)
	}
	// Verschlüsselt, nicht im Klartext: Die Zeile wandert in jede Sicherung.
	if bytes.Contains(stand.SMTPPasswortVerschluesselt, []byte(passwort)) {
		t.Fatal("das Passwort steht im Klartext in der Zeile")
	}
	klar, err := crypto.Entschluessele(stand.SMTPPasswortVerschluesselt)
	if err != nil {
		t.Fatalf("entschlüsseln: %v", err)
	}
	if string(klar) != passwort {
		t.Errorf("entschlüsselt %q, gespeichert war %q", klar, passwort)
	}
}

// Das Formular zeigt das gespeicherte Passwort nie an; ein leeres Feld heißt „nicht
// geändert". Löschte ein Speichern ohne Passwort das gespeicherte, fiele der Versand beim
// nächsten Mahnlauf aus, ohne dass jemand das Passwort angefasst hätte.
func TestMailEinstellungen_LeeresPasswortLaesstDasGespeicherteStehen(t *testing.T) {
	t.Setenv(crypto.SchluesselVariable, testSchluessel)
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewMailEinstellungsAbfragen(pool)
	setzeMailZeileZurueck(t, ctx, repo)
	t.Cleanup(func() { setzeMailZeileZurueck(t, context.Background(), repo) })

	if err := repo.Schreibe(ctx, "smtp.schule.example", "587", "bibliothek", "Erstes-Passwort", "a@schule.example"); err != nil {
		t.Fatalf("erstes Speichern: %v", err)
	}
	vorher, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}

	// Nur der Absender ändert sich, das Passwortfeld bleibt leer.
	if err := repo.Schreibe(ctx, "smtp.schule.example", "587", "bibliothek", "", "b@schule.example"); err != nil {
		t.Fatalf("zweites Speichern: %v", err)
	}
	nachher, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if nachher.AbsenderEmail != "b@schule.example" {
		t.Errorf("Absender %q, erwartet b@schule.example", nachher.AbsenderEmail)
	}
	if !bytes.Equal(nachher.SMTPPasswortVerschluesselt, vorher.SMTPPasswortVerschluesselt) {
		t.Fatal("ein Speichern ohne Passwort hat das gespeicherte verändert")
	}
	klar, err := crypto.Entschluessele(nachher.SMTPPasswortVerschluesselt)
	if err != nil || string(klar) != "Erstes-Passwort" {
		t.Errorf("nach dem zweiten Speichern entschlüsselt %q (Fehler %v), erwartet das erste Passwort", klar, err)
	}
}

// Ohne Schlüssel lässt sich nichts verschlüsseln. Das Speichern scheitert dann und fasst die
// Zeile nicht an, weder mit Klartext noch mit einem halben Stand.
func TestMailEinstellungen_OhneSchluesselWirdNichtsGespeichert(t *testing.T) {
	t.Setenv(crypto.SchluesselVariable, testSchluessel)
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewMailEinstellungsAbfragen(pool)
	setzeMailZeileZurueck(t, ctx, repo)
	t.Cleanup(func() { setzeMailZeileZurueck(t, context.Background(), repo) })

	t.Setenv(crypto.SchluesselVariable, "")
	if err := repo.Schreibe(ctx, "smtp.anders.example", "25", "x", "Passwort", "x@anders.example"); err == nil {
		t.Fatal("ohne Schlüssel gespeichert")
	}
	stand, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if stand.SMTPHost != "localhost" || stand.SMTPPasswortVerschluesselt != nil {
		t.Errorf("die Zeile wurde trotz Fehler angefasst: Host %q, Passwort gesetzt=%v",
			stand.SMTPHost, stand.SMTPPasswortVerschluesselt != nil)
	}
}
