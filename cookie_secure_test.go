package main

import (
	"os"
	"testing"
)

// Die Richtung des Zweifelsfalls ist hier das Sicherheitsmerkmal: Fehlt die
// Variable, galt bisher false — also OHNE Secure-Flag. Über HTTPS funktioniert
// dann trotzdem alles, weshalb es niemandem auffällt; das Cookie geht aber auch
// über eine einfache HTTP-Verbindung mit, sobald jemand eine erzwingt.
//
// CodeQL meldet die fünf Set-Cookie-Stellen weiterhin (es kann eine Variable nicht
// auflösen). Der Fund ist damit nicht "falsch" — er zeigt auf eine Entscheidung,
// die vorher an der falschen Stelle stillschweigend getroffen wurde.
func TestErmittleCookieSecure(t *testing.T) {
	faelle := []struct {
		name         string
		appEnv       string
		cookieSecure string
		setzen       bool
		erwartet     bool
	}{
		{"Produktion ohne Variable → sicher", "production", "", false, true},
		{"Produktion mit leerem Wert → sicher", "production", "   ", true, true},
		{"leeres APP_ENV ohne Variable → sicher", "", "", false, true},
		{"Produktion explizit true", "production", "true", true, true},
		{"Produktion explizit false bleibt false", "production", "false", true, false},
		{"lokal ohne Variable → unsicher erlaubt", "local", "", false, false},
		{"lokal explizit true", "local", "true", true, true},
		{"development ohne Variable → unsicher erlaubt", "development", "", false, false},
		{"Großschreibung wird erkannt", "PRODUCTION", "", false, true},
		{"Leerzeichen um den Wert", "production", " false ", true, false},
		{"numerisch 1", "production", "1", true, true},
		{"numerisch 0", "production", "0", true, false},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			t.Setenv("APP_ENV", f.appEnv)
			// t.Setenv merkt sich den ursprünglichen Zustand und stellt ihn nach dem
			// Test wieder her — auch dann, wenn wir die Variable danach entfernen.
			t.Setenv("COOKIE_SECURE", f.cookieSecure)
			if !f.setzen {
				if err := os.Unsetenv("COOKIE_SECURE"); err != nil {
					t.Fatalf("Umgebungsvariable konnte nicht entfernt werden: %v", err)
				}
			}

			if got := ermittleCookieSecure(); got != f.erwartet {
				t.Errorf("APP_ENV=%q COOKIE_SECURE=%q (gesetzt=%v): erwartet %v, war %v",
					f.appEnv, f.cookieSecure, f.setzen, f.erwartet, got)
			}
		})
	}
}
