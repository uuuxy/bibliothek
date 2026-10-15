package mailservice

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestSende_UngueltigerEmpfaenger(t *testing.T) {
	// Die Umgebung, mit der die erste Prüfung bestanden ist; t.Setenv stellt am Ende den vorigen
	// Wert wieder her.
	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_USER", "test")
	t.Setenv("SMTP_PASSWORD", "test")
	t.Setenv("SMTP_FROM", "test@example.com")

	tests := []struct {
		name    string
		to      string
		wantErr bool
	}{
		{
			name:    "Valid Email",
			to:      "valid@example.com",
			wantErr: true, // scheitert am fehlenden Server, nicht an der Prüfung der Adresse
		},
		{
			name:    "Header Injection",
			to:      "valid@example.com\r\nBcc: evil@example.com",
			wantErr: true, // scheitert an der Prüfung der Adresse
		},
		{
			name:    "Invalid Email Format",
			to:      "not an email",
			wantErr: true, // scheitert an der Prüfung der Adresse
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := Nachricht{
				An:      tt.to,
				Betreff: "Test Subject",
				Text:    "Test Body",
			}
			err := NewPostausgang(func() (SMTPKonfig, error) { return KonfigAusUmgebung(), nil }).Sende(req)
			if tt.wantErr {
				if err == nil {
					t.Error("kein Fehler, erwartet einen")
				}
				if tt.name != "Valid Email" {
					if got := err.Error(); !strings.Contains(got, "invalid recipient email address") {
						t.Errorf("err.Error() = %q, erwartet darin %q", got, "invalid recipient email address")
					}
					if errors.Is(err, ErrSMTPVersand) {
						t.Error("eine unbrauchbare Adresse ist kein Fehler des Zielservers — sonst antwortet der Handler mit 502 statt 400")
					}
				} else {
					// Marker statt Meldungstext: Daran entscheidet die Tür zwischen
					// 502 (Zielserver versagt, Meldung wird durchgereicht) und 500.
					if !errors.Is(err, ErrSMTPVersand) {
						t.Errorf("Fehler %v, erwartet %v", err, ErrSMTPVersand)
					}
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// Ein unlesbares SMTP-Passwort muss auffallen.
//
// LadeSMTPKonfig erzeugt diesen Fehler ausdrücklich, damit er nicht still bleibt („Ein
// unlesbares Passwort heißt, dass der APP_ENCRYPTION_KEY nicht mehr derselbe ist"). Meldete
// IstKonfiguriert nur „nicht konfiguriert", suchte der Admin nach einem Wechsel des Schlüssels
// in den Einstellungen, wo Host, Port und Benutzer stimmen.
func TestPostausgang_IstKonfiguriertLoggtUnlesbareKonfiguration(t *testing.T) {
	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })

	post := NewPostausgang(func() (SMTPKonfig, error) {
		return SMTPKonfig{}, errors.New("fehler beim Entschlüsseln des SMTP-Passworts")
	})
	if post.IstKonfiguriert() {
		t.Error("unlesbare Konfiguration gilt als konfiguriert")
	}
	if !strings.Contains(protokoll.String(), "nicht lesbar") {
		t.Errorf("kein Hinweis im Protokoll — der Grund bleibt unsichtbar: %q", protokoll.String())
	}
}
