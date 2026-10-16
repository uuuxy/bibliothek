package bereitschaft

import (
	"os"
	"strings"
	"testing"
)

// Die Vorgabe ist die sichere Richtung: Nur eine ausdrückliche „false"-Zeile oder eine
// Spielwiesen-Umgebung ohne gesetzte Variable schaltet die Absicherung aus.
func TestIstGeheimnisSchutzScharf(t *testing.T) {
	faelle := []struct {
		umgebung, roh string
		will          bool
	}{
		{"production", "", true},
		{"production", "true", true},
		{"production", "false", false},
		{"production", "ja", true},
		{"", "", true},
		{"local", "", false},
		{"development", "", false},
		{"test", "", false},
		{"local", "true", true},
		{"local", "false", false},
	}
	for _, f := range faelle {
		if got := IstGeheimnisSchutzScharf(f.umgebung, f.roh); got != f.will {
			t.Errorf("IstGeheimnisSchutzScharf(%q, %q) = %v; erwartet %v", f.umgebung, f.roh, got, f.will)
		}
	}
}

// Die Liste der Beispiel-Geheimnisse teilt sich die Selbstprüfung mit main.go. Fehlt ein
// Wert, startet der Server mit einem öffentlich bekannten Schlüssel.
func TestIstBeispielGeheimnis(t *testing.T) {
	for _, wert := range []string{
		"super-secret-default-key-at-least-32-bytes",
		"super-secure-aes-key-32-chars-ok",
		"supergeheim_lokal",
		// Die Platzhalter aus .env.example, die nach cp .env.example .env stehen bleiben:
		"super-secret-key-that-is-at-least-32-bytes-long",
		"your-32-character-encryption-key",
	} {
		if !IstBeispielGeheimnis(wert) {
			t.Errorf("%q wird nicht als Beispiel-Geheimnis erkannt", wert)
		}
	}
	for _, wert := range []string{"", "ein-eigenes-langes-geheimnis-mit-genug-zeichen"} {
		if IstBeispielGeheimnis(wert) {
			t.Errorf("%q gilt fälschlich als Beispiel-Geheimnis", wert)
		}
	}
}

// Der Wächter kennt die Platzhalter aus .env.example. Der Test liest sie aus der Datei, nicht
// aus einer Kopie: Ändert jemand die Beispielwerte, ohne den Wächter nachzuziehen, wird er
// rot, statt dass der Wächter blind wird.
func TestIstBeispielGeheimnis_KenntEnvExample(t *testing.T) {
	inhalt, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatalf(".env.example lesen: %v", err)
	}
	gesehen := 0
	for _, zeile := range strings.Split(string(inhalt), "\n") {
		for _, schluessel := range []string{"JWT_SECRET=", "APP_ENCRYPTION_KEY=", "BACKUP_ENCRYPTION_KEY="} {
			if !strings.HasPrefix(zeile, schluessel) {
				continue
			}
			wert := strings.TrimSpace(strings.TrimPrefix(zeile, schluessel))
			if wert == "" {
				continue // leer = kein kopierbares Geheimnis
			}
			gesehen++
			if !IstBeispielGeheimnis(wert) {
				t.Errorf(".env.example %s%q wird vom Wächter nicht als Beispiel erkannt — cp .env.example .env startet damit in Produktion", schluessel, wert)
			}
		}
	}
	if gesehen < 2 {
		t.Fatalf("nur %d Beispielwerte in .env.example gefunden — die Probe läuft ins Leere", gesehen)
	}
}
