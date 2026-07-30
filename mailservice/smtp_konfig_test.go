package mailservice

// Diese Tests halten die Antwort auf eine Frage fest, die vorher zwei verschiedene
// Antworten hatte: Womit wird verschickt? Die gespeicherte Konfiguration gewinnt, die
// Umgebung trägt nur, solange nichts gespeichert ist.

import (
	"context"
	"testing"

	"bibliothek/internal/crypto"

	"github.com/pashagolub/pgxmock/v4"
)

const testSchluessel = "0123456789abcdef0123456789abcdef" // 32 Zeichen für AES-256

func mockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	return mock
}

func konfigZeile(host, port, benutzer string, passwort []byte, absender string) *pgxmock.Rows {
	return pgxmock.NewRows([]string{"smtp_host", "smtp_port", "smtp_user", "smtp_password_encrypted", "sender_email"}).
		AddRow(host, port, benutzer, passwort, absender)
}

// Der Kernfall: Was der Admin gespeichert hat, gilt — auch wenn in der Umgebung noch
// etwas anderes steht (auf dem Schulserver steht dort die alte Konfiguration).
func TestLadeSMTPKonfigGespeichertesGewinnt(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", testSchluessel)
	t.Setenv("SMTP_HOST", "alt.example.de")
	t.Setenv("SMTP_PORT", "25")
	t.Setenv("SMTP_FROM", "alt@example.de")

	verschluesselt, err := crypto.Encrypt([]byte("geheim123"))
	if err != nil {
		t.Fatalf("Verschlüsseln: %v", err)
	}

	mock := mockPool(t)
	mock.ExpectQuery(`SELECT smtp_host`).
		WillReturnRows(konfigZeile("smtp.schule.de", "587", "bib", verschluesselt, "bib@schule.de"))

	konfig, err := LadeSMTPKonfig(context.Background(), mock)
	if err != nil {
		t.Fatalf("LadeSMTPKonfig: %v", err)
	}

	if konfig.Adresse() != "smtp.schule.de:587" {
		t.Errorf("Adresse = %q, want smtp.schule.de:587", konfig.Adresse())
	}
	if konfig.Absender != "bib@schule.de" {
		t.Errorf("Absender = %q — die Umgebung hat gewonnen", konfig.Absender)
	}
	if konfig.Passwort != "geheim123" {
		t.Errorf("Passwort wurde nicht entschlüsselt: %q", konfig.Passwort)
	}
	if konfig.Auth() == nil {
		t.Error("mit Benutzer und Passwort muss eine SMTP-Auth entstehen")
	}
}

// Solange nichts gespeichert ist, trägt die Umgebung weiter — sonst stünde eine
// frische Datenbank ohne Mailversand da.
func TestLadeSMTPKonfigFaelltAufUmgebungZurueck(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.umgebung.de")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "bib@umgebung.de")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_PASS", "")
	t.Setenv("SMTP_FROM", "")

	mock := mockPool(t)
	mock.ExpectQuery(`SELECT smtp_host`).
		WillReturnRows(konfigZeile("", "", "", nil, ""))

	konfig, err := LadeSMTPKonfig(context.Background(), mock)
	if err != nil {
		t.Fatalf("LadeSMTPKonfig: %v", err)
	}

	if konfig.Adresse() != "smtp.umgebung.de:587" {
		t.Errorf("Adresse = %q, want smtp.umgebung.de:587 (Port-Vorgabe)", konfig.Adresse())
	}
	// Ohne SMTP_FROM ist der Benutzer die beste verfügbare Absenderadresse.
	if konfig.Absender != "bib@umgebung.de" {
		t.Errorf("Absender = %q, want bib@umgebung.de", konfig.Absender)
	}
}

// Ein unlesbares Passwort heißt: Der APP_ENCRYPTION_KEY ist nicht mehr derselbe. Das
// muss auffallen und darf nicht still auf die Umgebung ausweichen — sonst verschickt
// die Anwendung über einen anderen Server als den, der im Formular steht.
func TestLadeSMTPKonfigMeldetUnlesbaresPasswort(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", testSchluessel)
	t.Setenv("SMTP_HOST", "smtp.umgebung.de")

	mock := mockPool(t)
	mock.ExpectQuery(`SELECT smtp_host`).
		WillReturnRows(konfigZeile("smtp.schule.de", "587", "bib", []byte("kein-gueltiger-geheimtext"), "bib@schule.de"))

	if _, err := LadeSMTPKonfig(context.Background(), mock); err == nil {
		t.Fatal("erwartet Fehler bei unlesbarem Passwort, bekam nil")
	}
}

// IstKonfiguriert entscheidet, ob ein Versand versucht oder übersprungen wird. Die
// Platzhalter stammen aus der Beispielkonfiguration — ein Verbindungsversuch gegen
// einen Host namens "Ihr SMTP-Host" hilft niemandem.
func TestIstKonfiguriert(t *testing.T) {
	faelle := map[string]struct {
		konfig SMTPKonfig
		want   bool
	}{
		"vollständig":       {SMTPKonfig{Host: "smtp.schule.de", Port: "587"}, true},
		"ohne Host":         {SMTPKonfig{Port: "587"}, false},
		"ohne Port":         {SMTPKonfig{Host: "smtp.schule.de"}, false},
		"Platzhalter-Host":  {SMTPKonfig{Host: "Ihr SMTP-Host", Port: "587"}, false},
		"Platzhalter-Pass":  {SMTPKonfig{Host: "smtp.schule.de", Port: "587", Passwort: "IhrPasswort"}, false},
		"Beispiel-Passwort": {SMTPKonfig{Host: "smtp.schule.de", Port: "587", Passwort: "secret"}, false},
	}

	for name, f := range faelle {
		t.Run(name, func(t *testing.T) {
			if got := f.konfig.IstKonfiguriert(); got != f.want {
				t.Errorf("IstKonfiguriert() = %v, want %v", got, f.want)
			}
		})
	}
}
