package auth

import (
	"strings"
	"testing"
)

// Der IMAP-Mock beantwortet jede Anmeldung mit "Passwort stimmt". Kombiniert mit
// dem Compose-Default IMAP_HOST=mock hieß das: Wer die E-Mail eines eingetragenen
// Benutzers kannte, war angemeldet. Diese Tests halten fest, in welchen Umgebungen
// der Mock überhaupt noch erreichbar ist.
func TestPruefeIMAPKonfiguration(t *testing.T) {
	faelle := []struct {
		name       string
		appEnv     string
		imapHost   string
		willFehler bool
		fehlerEnth string
	}{
		{name: "echter Host in Produktion", appEnv: "production", imapHost: "imap.example.test"},
		{name: "Mock lokal erlaubt", appEnv: "local", imapHost: "mock"},
		{name: "Mock in Entwicklung erlaubt", appEnv: "development", imapHost: "mock"},
		{name: "Mock im Test erlaubt", appEnv: "test", imapHost: "mock"},
		{
			name: "Mock in Produktion verboten", appEnv: "production", imapHost: "mock",
			willFehler: true, fehlerEnth: "jedes Passwort",
		},
		{
			// APP_ENV ungesetzt ist der gefährlichste Fall: Er sieht harmlos aus und
			// war vorher exakt der Compose-Default-Zustand.
			name: "Mock ohne APP_ENV verboten", appEnv: "", imapHost: "mock",
			willFehler: true, fehlerEnth: "jedes Passwort",
		},
		{
			name: "leerer Host verboten", appEnv: "production", imapHost: "",
			willFehler: true, fehlerEnth: "nicht gesetzt",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			t.Setenv("APP_ENV", f.appEnv)
			t.Setenv("IMAP_HOST", f.imapHost)

			err := PruefeIMAPKonfiguration()
			if f.willFehler {
				if err == nil {
					t.Fatalf("APP_ENV=%q IMAP_HOST=%q: kein Fehler; want Fehler", f.appEnv, f.imapHost)
				}
				if !strings.Contains(err.Error(), f.fehlerEnth) {
					t.Errorf("Fehlertext %q enthält %q nicht", err.Error(), f.fehlerEnth)
				}
				return
			}
			if err != nil {
				t.Fatalf("APP_ENV=%q IMAP_HOST=%q: unerwarteter Fehler %v", f.appEnv, f.imapHost, err)
			}
		})
	}
}

// Zweite Schranke: Selbst wenn die Variable erst zur Laufzeit auf "mock" gesetzt
// wird (also am Startup-Check vorbei), darf AuthenticateIMAP außerhalb der
// lokalen Entwicklung nicht mehr blind "Passwort korrekt" melden.
func TestAuthenticateIMAP_MockNurLokal(t *testing.T) {
	t.Run("lokal akzeptiert", func(t *testing.T) {
		t.Setenv("APP_ENV", "local")
		t.Setenv("IMAP_HOST", "mock")
		if err := AuthenticateIMAP("wer@example.test", "beliebig"); err != nil {
			t.Fatalf("lokaler Mock: unerwarteter Fehler %v", err)
		}
	})

	t.Run("Produktion lehnt ab", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("IMAP_HOST", "mock")
		if err := AuthenticateIMAP("wer@example.test", "beliebig"); err == nil {
			t.Fatal("Mock in Produktion hat die Anmeldung akzeptiert; want Ablehnung")
		}
	})

	// Ohne IMAP_HOST wurde vorher imap.philipp-reis-schule.de kontaktiert. Der Test
	// darf deshalb keine Netzverbindung auslösen — er belegt, dass sofort abgelehnt
	// wird, statt eine echte Verbindung zur Schule aufzubauen.
	t.Run("ohne Host keine Verbindung zur Schule", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("IMAP_HOST", "")
		if err := AuthenticateIMAP("wer@example.test", "beliebig"); err == nil {
			t.Fatal("leerer IMAP_HOST wurde akzeptiert; want Ablehnung ohne Netzzugriff")
		}
	})
}
