package docs

import (
	"strings"
	"testing"
)

// vorlageWert liest einen Wert aus .env.example; fehlt die Zeile, ist das ein Fehler des
// Gates und kein leerer Wert.
func vorlageWert(t *testing.T, name string) string {
	t.Helper()
	for _, zeile := range strings.Split(lies(t, "../.env.example"), "\n") {
		if wert, ok := strings.CutPrefix(zeile, name+"="); ok {
			return strings.TrimSpace(wert)
		}
	}
	t.Fatalf(".env.example ohne Zeile %s= — Gate nachziehen", name)
	return ""
}

// Wer die Vorlage kopiert, darf nicht einen Handgriff von der offenen Anmeldung entfernt
// sein: mock in der Vorlage startet in production nicht, und APP_ENV=local behöbe das,
// indem es jedes Passwort gelten lässt.
//
// Blindheit: nur die Zeilen IMAP_HOST= und APP_ENV= in .env.example. Eine andere Vorlage
// (Compose-Datei, Anleitung im Text) und andere Vorgaben sieht der Test nicht.
func TestEnvVorlage_TraegtKeineMockAnmeldung(t *testing.T) {
	if wert := vorlageWert(t, "IMAP_HOST"); strings.EqualFold(wert, "mock") {
		t.Errorf(".env.example setzt IMAP_HOST=%s — auf einem Server nimmt mock jedes Passwort an", wert)
	}
	if wert := vorlageWert(t, "APP_ENV"); wert != "production" {
		t.Errorf(".env.example setzt APP_ENV=%s, erwartet production", wert)
	}
}
