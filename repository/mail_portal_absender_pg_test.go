package repository

import (
	"bytes"
	"context"
	"slices"
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
	if err := repo.SchreibePortalAbsender(ctx, "portal@schule.example", "portal", passwort, nil); err != nil {
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
	if err := repo.SchreibePortalAbsender(ctx, "anliegen@schule.example", "anliegen", "", nil); err != nil {
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

// Die Arten von Mails des zweiten Absenders: ab Werk nur portal, nil lässt sie stehen, eine leere
// Liste nimmt alle heraus, und eine Art, die das Programm nicht kennt, lehnt die Datenbank ab.
func TestSchreibePortalAbsender_Arten(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewMailEinstellungsAbfragen(pool)
	setzeMailZeileZurueck(t, ctx, repo)
	t.Cleanup(func() { setzeMailZeileZurueck(t, context.Background(), repo) })
	arten := func() []string {
		t.Helper()
		stand, err := repo.Lade(ctx)
		if err != nil {
			t.Fatalf("lesen: %v", err)
		}
		return stand.PortalAbsenderFuer
	}

	if got := arten(); !slices.Equal(got, []string{"portal"}) {
		t.Errorf("ab Werk %q, erwartet [portal]", got)
	}
	for _, f := range []struct {
		fuer, want []string
	}{
		{[]string{"mahnliste", "bestellung"}, []string{"mahnliste", "bestellung"}},
		{nil, []string{"mahnliste", "bestellung"}},
		{[]string{}, []string{}},
	} {
		if err := repo.SchreibePortalAbsender(ctx, "zweiter@schule.example", "", "", f.fuer); err != nil {
			t.Fatalf("fuer %q: %v", f.fuer, err)
		}
		if got := arten(); !slices.Equal(got, f.want) {
			t.Errorf("nach fuer %q: %q, erwartet %q", f.fuer, got, f.want)
		}
	}
	if err := repo.SchreibePortalAbsender(ctx, "zweiter@schule.example", "", "", []string{"alarm"}); err == nil {
		t.Error("eine unbekannte Art wurde gespeichert, erwartet die Ablehnung der Datenbank")
	}
}
