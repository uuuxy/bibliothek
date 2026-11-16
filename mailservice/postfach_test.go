package mailservice

// Diese Tests halten fest, unter welchem Absender eine Mail hinausgeht: Die Mails aus Mein Portal
// nehmen den eigenen Absender, sobald einer eingetragen ist; alle anderen, die Bestellmail an den
// Händler eingeschlossen, bleiben beim Postausgang.

import (
	"context"
	"strings"
	"testing"

	"bibliothek/internal/crypto"
	"bibliothek/internal/smtptest"
)

// Ohne eigene Adresse ändert sich für die Mails aus Mein Portal nichts.
func TestFuer_OhnePortalAbsenderBleibtEsBeimPostausgang(t *testing.T) {
	haupt := SMTPKonfig{Host: "smtp.schule.de", Port: "587", Benutzer: "bib", Passwort: "geheim", Absender: "bib@schule.de"}

	konfig, err := haupt.fuer(PostfachPortal)
	if err != nil {
		t.Fatalf("fuer: %v", err)
	}
	if konfig.Absender != "bib@schule.de" || konfig.Benutzer != "bib" || konfig.Passwort != "geheim" {
		t.Errorf("Portal ohne eigene Adresse = %+v, erwartet den Postausgang", konfig)
	}
}

func TestLadeSMTPKonfig_PortalAbsenderMitEigenemZugang(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", testSchluessel)
	haupt, err := crypto.Verschluessele([]byte("haupt-geheim"))
	if err != nil {
		t.Fatalf("Verschlüsseln: %v", err)
	}
	portal, err := crypto.Verschluessele([]byte("portal-geheim"))
	if err != nil {
		t.Fatalf("Verschlüsseln: %v", err)
	}

	mock := attrappe(t)
	mock.ExpectQuery(`SELECT smtp_host`).WillReturnRows(konfigZeileMitPortal(
		"smtp.schule.de", "587", "bib", haupt, "bib@schule.de", " portal@schule.de ", "portal", portal))

	konfig, err := ladeSMTPKonfig(context.Background(), mock)
	if err != nil {
		t.Fatalf("ladeSMTPKonfig: %v", err)
	}

	allgemein, err := konfig.fuer(PostfachAllgemein)
	if err != nil {
		t.Fatalf("fuer(PostfachAllgemein): %v", err)
	}
	if allgemein.Absender != "bib@schule.de" || allgemein.Benutzer != "bib" || allgemein.Passwort != "haupt-geheim" {
		t.Errorf("Postausgang = %+v — der Absender für Mein Portal ist hineingeraten", allgemein)
	}

	p, err := konfig.fuer(PostfachPortal)
	if err != nil {
		t.Fatalf("fuer(PostfachPortal): %v", err)
	}
	if p.Absender != "portal@schule.de" || p.Benutzer != "portal" || p.Passwort != "portal-geheim" {
		t.Errorf("Portal = Absender %q, Benutzer %q, Passwort %q — erwartet den eigenen Zugang",
			p.Absender, p.Benutzer, p.Passwort)
	}
	if p.adresse() != "smtp.schule.de:587" {
		t.Errorf("Portal-Server = %q, erwartet den des Postausgangs", p.adresse())
	}
}

// Ein leerer Benutzer heißt: Die Adresse geht mit dem Zugang des Postausgangs hinaus.
func TestLadeSMTPKonfig_PortalAbsenderOhneEigenenZugang(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", testSchluessel)
	haupt, err := crypto.Verschluessele([]byte("haupt-geheim"))
	if err != nil {
		t.Fatalf("Verschlüsseln: %v", err)
	}

	mock := attrappe(t)
	mock.ExpectQuery(`SELECT smtp_host`).WillReturnRows(konfigZeileMitPortal(
		"smtp.schule.de", "587", "bib", haupt, "bib@schule.de", "portal@schule.de", "", nil))

	konfig, err := ladeSMTPKonfig(context.Background(), mock)
	if err != nil {
		t.Fatalf("ladeSMTPKonfig: %v", err)
	}
	p, err := konfig.fuer(PostfachPortal)
	if err != nil {
		t.Fatalf("fuer(PostfachPortal): %v", err)
	}
	if p.Absender != "portal@schule.de" || p.Benutzer != "bib" || p.Passwort != "haupt-geheim" {
		t.Errorf("Portal = Absender %q, Benutzer %q, Passwort %q — erwartet die Adresse mit dem Zugang des Postausgangs",
			p.Absender, p.Benutzer, p.Passwort)
	}
}

// Ein unlesbares Passwort des Portal-Absenders hält nur dessen Mails auf. Hielte es den ganzen
// Versand auf, bliebe auch die Bestellmail an den Händler liegen.
func TestLadeSMTPKonfig_UnlesbaresPortalPasswortHaeltNurDiePortalMailsAuf(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", testSchluessel)

	mock := attrappe(t)
	mock.ExpectQuery(`SELECT smtp_host`).WillReturnRows(konfigZeileMitPortal(
		"smtp.schule.de", "587", "", nil, "bib@schule.de", "portal@schule.de", "portal", []byte("kein-gueltiger-geheimtext")))

	konfig, err := ladeSMTPKonfig(context.Background(), mock)
	if err != nil {
		t.Fatalf("ladeSMTPKonfig scheitert am Passwort des Portal-Absenders: %v", err)
	}
	if _, err := konfig.fuer(PostfachAllgemein); err != nil {
		t.Errorf("Postausgang aufgehalten: %v", err)
	}
	if _, err := konfig.fuer(PostfachPortal); err == nil || !strings.Contains(err.Error(), "zweiten Absenders") {
		t.Errorf("fuer(PostfachPortal) = %v, erwartet den Fehler des Passworts des zweiten Absenders", err)
	}
}

// Der Weg einer echten Mail: Der Postausgang wählt den Absender nach der Art der Mail, wie die
// Einstellungen sie dem zweiten Absender geben. Ohne Art und ohne Haken gilt der Postausgang.
func TestPostausgang_SendeWaehltDenAbsenderNachDerArt(t *testing.T) {
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true") // Testserver ohne STARTTLS
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)
	post := NewPostausgang(func(context.Context) (SMTPKonfig, error) {
		return SMTPKonfig{Host: host, Port: port, Absender: "bib@schule.example",
			portal:     &portalAbsender{absender: "zweiter@schule.example"},
			portalFuer: map[Mailart]bool{MailartBestellung: true, MailartMahnliste: true}}, nil
	})

	faelle := []struct {
		art      Mailart
		absender string
	}{
		{"", "From: bib@schule.example"},
		{MailartPortal, "From: bib@schule.example"},
		{MailartAbgaenger, "From: bib@schule.example"},
		{MailartMahnliste, "From: zweiter@schule.example"},
		{MailartBestellung, "From: zweiter@schule.example"},
	}
	for _, f := range faelle {
		err := post.Sende(t.Context(), Nachricht{An: "lehrerin@schule.example", Betreff: "x", Text: "y", Art: f.art})
		if err != nil {
			t.Fatalf("Art %q: Versand fehlgeschlagen: %v", f.art, err)
		}
		sitzung := <-sitzungen
		if !strings.Contains(sitzung.Nachricht, f.absender) {
			t.Errorf("Art %q: erwartet %q in\n%s", f.art, f.absender, sitzung.Nachricht)
		}
	}
}

// Ab Werk nimmt der zweite Absender nur die Mails aus Mein Portal, wie die gespeicherte Zeile sagt.
func TestLadeSMTPKonfig_ZweiterAbsenderNimmtDieGespeichertenArten(t *testing.T) {
	mock := attrappe(t)
	mock.ExpectQuery(`SELECT smtp_host`).WillReturnRows(konfigZeileMitPortal(
		"smtp.schule.de", "587", "", nil, "bib@schule.de", "portal@schule.de", "", nil))
	konfig, err := ladeSMTPKonfig(context.Background(), mock)
	if err != nil {
		t.Fatalf("ladeSMTPKonfig: %v", err)
	}
	for _, art := range Mailarten {
		want := PostfachAllgemein
		if art == MailartPortal {
			want = PostfachPortal
		}
		if got := konfig.postfachFuer(art); got != want {
			t.Errorf("postfachFuer(%q) = %d, erwartet %d", art, got, want)
		}
	}
	if got := konfig.postfachFuer(""); got != PostfachAllgemein {
		t.Errorf("postfachFuer(ohne Art) = %d, erwartet den Postausgang", got)
	}
}
