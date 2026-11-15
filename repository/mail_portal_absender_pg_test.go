package repository

import (
	"bytes"
	"context"
	"testing"

	"bibliothek/internal/crypto"
	"bibliothek/internal/pgtest"
)

// Der Absender der Mails aus Mein Portal steht in derselben Zeile wie der Postausgang. Geprüft
// wird der Rundweg wie dort: Das Passwort steht verschlüsselt in der Zeile, ein leeres
// Passwortfeld lässt das gespeicherte stehen, und der Postausgang bleibt dabei unberührt.
func TestMailEinstellungen_PortalAbsenderImRundweg(t *testing.T) {
	t.Setenv(crypto.SchluesselVariable, testSchluessel)
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewMailEinstellungsAbfragen(pool)
	setzeMailZeileZurueck(t, ctx, repo)
	t.Cleanup(func() { setzeMailZeileZurueck(t, context.Background(), repo) })

	if err := repo.Schreibe(ctx, "smtp.schule.example", "587", "bibliothek", "Haupt-Passwort", "bibliothek@schule.example"); err != nil {
		t.Fatalf("Postausgang speichern: %v", err)
	}
	vorher, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}

	const passwort = "Portal-Passwort-1!"
	if err := repo.SchreibePortalAbsender(ctx, "portal@schule.example", "portal", passwort); err != nil {
		t.Fatalf("Absender für Mein Portal speichern: %v", err)
	}
	stand, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if stand.PortalAbsenderEmail != "portal@schule.example" || stand.PortalSMTPBenutzer != "portal" {
		t.Errorf("gelesen: Adresse %q, Benutzer %q", stand.PortalAbsenderEmail, stand.PortalSMTPBenutzer)
	}
	if bytes.Contains(stand.PortalSMTPPasswortVerschluesselt, []byte(passwort)) {
		t.Fatal("das Passwort steht im Klartext in der Zeile")
	}
	klar, err := crypto.Entschluessele(stand.PortalSMTPPasswortVerschluesselt)
	if err != nil || string(klar) != passwort {
		t.Errorf("entschlüsselt %q (%v), gespeichert war %q", klar, err, passwort)
	}
	if stand.SMTPBenutzer != "bibliothek" || stand.AbsenderEmail != "bibliothek@schule.example" ||
		!bytes.Equal(stand.SMTPPasswortVerschluesselt, vorher.SMTPPasswortVerschluesselt) {
		t.Error("der Postausgang hat sich beim Speichern des Portal-Absenders verändert")
	}

	// Ein leeres Passwortfeld heißt „nicht geändert", auch hier.
	if err := repo.SchreibePortalAbsender(ctx, "anliegen@schule.example", "anliegen", ""); err != nil {
		t.Fatalf("erneut speichern: %v", err)
	}
	danach, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if danach.PortalAbsenderEmail != "anliegen@schule.example" || danach.PortalSMTPBenutzer != "anliegen" {
		t.Errorf("gelesen: Adresse %q, Benutzer %q", danach.PortalAbsenderEmail, danach.PortalSMTPBenutzer)
	}
	if !bytes.Equal(danach.PortalSMTPPasswortVerschluesselt, stand.PortalSMTPPasswortVerschluesselt) {
		t.Error("ein leeres Passwortfeld hat das gespeicherte Passwort verändert")
	}
}
