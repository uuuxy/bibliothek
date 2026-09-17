package api

import (
	"os"
	"testing"
)

// TestWerkzeug_KameraBild ist kein Gate, sondern das Werkzeug hinter
// scripts/kamera_probe.sh: Es schreibt den Strichcode, den die künstliche Kamera zeigen
// soll — erzeugt vom GENERATOR DER ANWENDUNG, damit die Probe dasselbe sieht wie ein
// gedrucktes Etikett und nicht ein sauberes Bild aus einer fremden Bibliothek.
//
// Ohne KAMERA_ZIEL tut es nichts; in der CI läuft es deshalb als übersprungener Test mit.
func TestWerkzeug_KameraBild(t *testing.T) {
	ziel := os.Getenv("KAMERA_ZIEL")
	if ziel == "" {
		t.Skip("KAMERA_ZIEL nicht gesetzt — das Werkzeug läuft nur über scripts/kamera_probe.sh")
	}
	png, err := GenerateBarcodePNG(os.Getenv("KAMERA_INHALT"), os.Getenv("KAMERA_QR") == "1", 900, 260)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ziel, png, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("geschrieben: %s", ziel)
}
