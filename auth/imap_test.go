package auth

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
)

// Die Attrappe des Mailservers beantwortet jede Anmeldung mit „Passwort stimmt". Zusammen mit
// IMAP_HOST=mock als Vorgabe in Compose hieße das: Wer die E-Mail-Adresse eines eingetragenen
// Kontos kennt, ist angemeldet. Die Tests halten fest, in welchen Umgebungen die Attrappe
// überhaupt erreichbar ist.
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
			// APP_ENV ungesetzt ist der gefährlichste Fall: Er sieht harmlos aus und ist der Zustand,
			// den Compose ohne weitere Angaben herstellt.
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

// Die Startmeldung nennt den Weg für den Server: Mailserver eintragen, APP_ENV lassen.
// Sonst liest sie sich wie die Anweisung, APP_ENV auf local zu stellen.
func TestPruefeIMAPKonfiguration_MeldungLaesstAppEnvStehen(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("IMAP_HOST", "mock")

	err := PruefeIMAPKonfiguration()
	if err == nil {
		t.Fatal("IMAP_HOST=mock in production: kein Fehler")
	}
	for _, muss := range []string{"Mailserver der Schule eintragen", "APP_ENV nicht ändern"} {
		if !strings.Contains(err.Error(), muss) {
			t.Errorf("Startmeldung %q nennt %q nicht", err.Error(), muss)
		}
	}
}

// Zweite Schranke: Auch wenn die Variable erst zur Laufzeit auf "mock" gesetzt wird, an der
// Prüfung beim Start vorbei, darf pruefeIMAPAnmeldung außerhalb der lokalen Entwicklung nicht
// blind „Passwort stimmt" melden.
func TestPruefeIMAPAnmeldung_AttrappeNurLokal(t *testing.T) {
	t.Run("lokal akzeptiert", func(t *testing.T) {
		t.Setenv("APP_ENV", "local")
		t.Setenv("IMAP_HOST", "mock")
		if err := (&Anmeldung{}).pruefeIMAPAnmeldung(context.Background(), "wer@example.test", "beliebig"); err != nil {
			t.Fatalf("lokaler Mock: unerwarteter Fehler %v", err)
		}
	})

	t.Run("Produktion lehnt ab", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("IMAP_HOST", "mock")
		if err := (&Anmeldung{}).pruefeIMAPAnmeldung(context.Background(), "wer@example.test", "beliebig"); err == nil {
			t.Fatal("Mock in Produktion hat die Anmeldung akzeptiert; want Ablehnung")
		}
	})

	// Ohne IMAP_HOST wird abgelehnt, ohne eine Verbindung aufzubauen; der Test darf keine
	// Netzverbindung auslösen, schon gar nicht zum echten Mailserver der Schule.
	t.Run("ohne Host keine Verbindung zur Schule", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("IMAP_HOST", "")
		if err := (&Anmeldung{}).pruefeIMAPAnmeldung(context.Background(), "wer@example.test", "beliebig"); err == nil {
			t.Fatal("leerer IMAP_HOST wurde akzeptiert; want Ablehnung ohne Netzzugriff")
		}
	})
}

// Ein nicht erreichbarer Server liefert errMailserverNichtErreichbar. Daran erkennt die Tür der
// Anmeldung den Ausfall (503) und unterscheidet ihn vom falschen Passwort (401 mit
// Fehlversuch).
func TestPruefeIMAPAnmeldung_ToterServerIstAusfall(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listener: %v", err)
	}
	host, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	t.Setenv("IMAP_HOST", host)
	t.Setenv("IMAP_PORT", port)

	pruefFehler := (&Anmeldung{}).pruefeIMAPAnmeldung(context.Background(), "wer@schule.de", "egal")
	if !errors.Is(pruefFehler, errMailserverNichtErreichbar) {
		t.Fatalf("erwartet ErrMailserverNichtErreichbar, bekam: %v", pruefFehler)
	}
}

// Die Adresse des Mailservers: Ein Port am Hostnamen wird verworfen, es gilt IMAP_PORT und
// ohne ihn 993. Der Name, gegen den das Zertifikat geprüft wird, ist der Host ohne Port.
func TestIMAPVerbindungsdaten(t *testing.T) {
	for _, f := range []struct{ host, port, adresse string }{
		{"mail.example.test", "", "mail.example.test:993"},
		{"mail.example.test:143", "", "mail.example.test:993"},
		{"mail.example.test", "1993", "mail.example.test:1993"},
	} {
		t.Setenv("IMAP_PORT", f.port)
		adresse, tlsKonfig := (&Anmeldung{}).imapVerbindungsdaten(f.host)
		if adresse != f.adresse {
			t.Errorf("Host %q, IMAP_PORT %q: Adresse %q, erwartet %q", f.host, f.port, adresse, f.adresse)
		}
		if tlsKonfig.ServerName != "mail.example.test" {
			t.Errorf("Host %q: Zertifikatsname %q, erwartet mail.example.test", f.host, tlsKonfig.ServerName)
		}
	}
}
