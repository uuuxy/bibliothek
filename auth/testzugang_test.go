package auth

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// testzugangPasswortBeispiel: 24 Zeichen, über der Mindestlänge.
const testzugangPasswortBeispiel = "xJ7-quell-fluss-marmor42"

// setzeTestzugang schaltet den Zugang für einen Test frei.
func setzeTestzugang(t *testing.T, email, passwort string) {
	t.Helper()
	t.Setenv(testzugangEmailEnv, email)
	t.Setenv(testzugangPasswortEnv, passwort)
}

// toterMailserver stellt IMAP auf einen Port, an dem nichts lauscht. Damit ist jeder
// Login, der trotzdem gelingt, BEWEISBAR ohne den Mailserver zustande gekommen — mit
// IMAP_HOST=mock wäre der Test wertlos, weil dann ohnehin jedes Passwort passt.
func toterMailserver(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "127.0.0.1")
	t.Setenv("IMAP_PORT", "1")
}

func TestPruefeTestzugangKonfiguration(t *testing.T) {
	faelle := []struct {
		name     string
		email    string
		passwort string
		domain   string
		wantErr  bool
	}{
		{name: "beide leer ist der Normalfall"},
		{name: "vollständig", email: "test@bibliothek.invalid", passwort: testzugangPasswortBeispiel},
		{name: "nur E-Mail: tote Tür", email: "test@bibliothek.invalid", wantErr: true},
		{name: "nur Passwort: wirkungslos", passwort: testzugangPasswortBeispiel, wantErr: true},
		{name: "keine Adresse", email: "kein-at-zeichen", passwort: testzugangPasswortBeispiel, wantErr: true},
		{name: "Passwort zu kurz", email: "test@bibliothek.invalid", passwort: "kurz", wantErr: true},
		{name: "Passwort ist die Adresse", email: "test@bibliothek.invalid", passwort: "test@bibliothek.invalid", wantErr: true},
		{
			name:     "Adresse auf der Schuldomain",
			email:    "test@philipp-reis-schule.de",
			passwort: testzugangPasswortBeispiel,
			domain:   "philipp-reis-schule.de",
			wantErr:  true,
		},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			setzeTestzugang(t, f.email, f.passwort)
			t.Setenv(selbstanmeldeDomainEnv, f.domain)
			err := PruefeTestzugangKonfiguration()
			if (err != nil) != f.wantErr {
				t.Fatalf("wantErr=%v, bekam %v", f.wantErr, err)
			}
		})
	}
}

// TestIstTestzugang_OhneUmgebungImmerFalsch ist die Ratsche gegen die schlimmste
// Regression dieses Bauteils: einen Sonderweg, der auch ohne gesetzte Variablen offen
// steht. Leer heißt zu — für JEDE Eingabe, auch die leere.
func TestIstTestzugang_OhneUmgebungImmerFalsch(t *testing.T) {
	setzeTestzugang(t, "", "")
	for _, f := range []struct{ email, passwort string }{
		{"", ""},
		{"test@bibliothek.invalid", testzugangPasswortBeispiel},
		{"admin@schule.de", "irgendwas"},
	} {
		if istTestzugang(f.email, f.passwort) {
			t.Errorf("ohne TESTZUGANG_* darf %q/%q nicht durchkommen", f.email, f.passwort)
		}
	}
}

func TestIstTestzugang_NurDasPaarPasst(t *testing.T) {
	setzeTestzugang(t, "test@bibliothek.invalid", testzugangPasswortBeispiel)

	if !istTestzugang("test@bibliothek.invalid", testzugangPasswortBeispiel) {
		t.Error("das konfigurierte Paar muss durchkommen")
	}
	// Die Adresse kommt aus einem Formular: Groß-/Kleinschreibung und Leerraum dürfen
	// nicht entscheiden, sonst scheitert der Tester an seiner eigenen Tastatur.
	if !istTestzugang("  Test@Bibliothek.INVALID ", testzugangPasswortBeispiel) {
		t.Error("Adresse muss unabhängig von Schreibweise und Leerraum passen")
	}
	// Das Passwort dagegen gilt zeichengenau.
	for _, f := range []struct{ name, email, passwort string }{
		{"falsches Passwort", "test@bibliothek.invalid", "xJ7-quell-fluss-marmor43"},
		{"Passwort in anderer Schreibweise", "test@bibliothek.invalid", "XJ7-QUELL-FLUSS-MARMOR42"},
		{"leeres Passwort", "test@bibliothek.invalid", ""},
		{"fremde Adresse mit dem Testpasswort", "admin@schule.de", testzugangPasswortBeispiel},
	} {
		if istTestzugang(f.email, f.passwort) {
			t.Errorf("%s darf nicht durchkommen", f.name)
		}
	}
}

// TestLoginHandler_TestzugangOhneMailserver ist der Beweis am LIVE-PFAD: nicht an
// istTestzugang, sondern am echten Handler, mit totem Mailserver.
func TestLoginHandler_TestzugangOhneMailserver(t *testing.T) {
	toterMailserver(t)
	setzeTestzugang(t, "test@bibliothek.invalid", testzugangPasswortBeispiel)
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("test@bibliothek.invalid").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email"}).
			AddRow("u-test", "", "admin", "Testzugang", "(extern)", true, "test@bibliothek.invalid"))
	// Jede Anmeldung über den Sonderweg MUSS im Protokoll stehen.
	// WithArgs ist Pflicht, nicht Zierde: Ohne die Argumente prüft pgxmock nur, DASS
	// geschrieben wurde — nicht, dass der Eintrag auf das richtige Konto und unter der
	// wiederfindbaren Aktion landet.
	mock.ExpectExec("INSERT INTO audit_logs").
		WithArgs("u-test", "testzugang_anmeldung", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	rec := doLogin(t, a, mock, `{"email":"test@bibliothek.invalid","password":"`+testzugangPasswortBeispiel+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200 ohne Mailserver, bekam %d: %s", rec.Code, rec.Body.String())
	}
	var resp LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort kein LoginResponse-JSON: %v", err)
	}
	if resp.Rolle != RoleAdmin || len(resp.Permissions) != 1 || resp.Permissions[0] != "*" {
		t.Errorf("Testzugang muss die Rechte seines Kontos bekommen: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Protokolleintrag fehlt: %v", err)
	}
}

// TestLoginHandler_TestzugangFalschesPasswortGehtZumMailserver: Die Gegenprobe. Ohne sie
// bewiese der Test oben nur, dass die Adresse durchkommt — nicht, dass das Passwort
// überhaupt geprüft wird. Mit falschem Passwort fällt die Anmeldung auf den regulären
// Weg zurück und scheitert dort am toten Mailserver (503, kein Cookie).
func TestLoginHandler_TestzugangFalschesPasswortGehtZumMailserver(t *testing.T) {
	toterMailserver(t)
	setzeTestzugang(t, "test@bibliothek.invalid", testzugangPasswortBeispiel)
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	rec := doLogin(t, a, mock, `{"email":"test@bibliothek.invalid","password":"falsch-aber-lang-genug-xx"}`)
	if rec.Code == http.StatusOK {
		t.Fatalf("falsches Passwort darf nicht anmelden, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("bei fehlgeschlagenem Login darf kein Cookie gesetzt werden")
	}
}
