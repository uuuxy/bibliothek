package crypto

import (
	"bytes"
	"strings"
	"testing"
)

// Der Wechsel des Schlüssels hängt daran, dass VerschluesseleMit und EntschluesseleMit sich
// verhalten wie Verschluessele und Entschluessele, nur mit ausdrücklich übergebenem Schlüssel.
// Sonst hinterließe cmd/rotate-encryption-key einen Bestand, den das Programm nicht mehr lesen
// kann, und das zeigte sich erst, wenn jemand ein Foto öffnet.

func TestSchluesselAus(t *testing.T) {
	faelle := map[string]struct {
		wert       string
		laenge     int
		fehler     bool
		begruend   string
		wantErrMsg string
	}{
		"32 Zeichen Klartext": {"12345678901234567890123456789012", 32, false, "", ""},
		"64 Zeichen Hex":      {"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff", 32, false, "", ""},
		"leer":                {"", 0, true, "leerer Schlüssel muss abgelehnt werden", "schlüssel ist leer"},
		"zu kurz":             {"kurz", 0, true, "zu kurzer Schlüssel muss abgelehnt werden", "schlüssel muss genau 32 Zeichen (oder 64 Hex-Zeichen) lang sein"},
		"64 Zeichen kein Hex": {"zzzz2233445566778899aabbccddeeff00112233445566778899aabbccddeeff", 0, true, "ungültiges Hex muss abgelehnt werden", "ungültiges Hex-Format"},
	}

	for name, f := range faelle {
		t.Run(name, func(t *testing.T) {
			schluessel, err := SchluesselAus(f.wert)
			if f.fehler {
				if err == nil {
					t.Fatalf("%s", f.begruend)
				}
				if f.wantErrMsg != "" {
					if !strings.Contains(err.Error(), f.wantErrMsg) {
						t.Errorf("SchluesselAus() error = %v, wantErrMsg %v", err, f.wantErrMsg)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if len(schluessel) != f.laenge {
				t.Errorf("Schlüssellänge %d, erwartet %d", len(schluessel), f.laenge)
			}
		})
	}
}

func TestVerschluesseleMitUndEntschluesseleMitSindGegenlaeufig(t *testing.T) {
	schluessel, err := SchluesselAus("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("Schlüssel: %v", err)
	}

	klartext := []byte("ein Schülerfoto oder ein SMTP-Passwort")
	geheimtext, err := VerschluesseleMit(schluessel, klartext)
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}
	if bytes.Contains(geheimtext, klartext) {
		t.Fatal("der Klartext steht unverändert im Ciphertext")
	}

	zurueck, err := EntschluesseleMit(schluessel, geheimtext)
	if err != nil {
		t.Fatalf("entschlüsseln: %v", err)
	}
	if !bytes.Equal(zurueck, klartext) {
		t.Errorf("zurück %q, erwartet %q", zurueck, klartext)
	}
}

// Was mit dem alten Schlüssel geschrieben wurde, liest der neue nicht. Deshalb fasst
// rotate-encryption-key jeden Datensatz an, und ein bloßer Tausch der Umgebungsvariable wäre
// Datenverlust.
func TestNeuerSchluesselLiestAltenBestandNicht(t *testing.T) {
	alt, err := SchluesselAus("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("alter Schlüssel: %v", err)
	}
	neu, err := SchluesselAus("abcdefghijklmnopqrstuvwxyz012345")
	if err != nil {
		t.Fatalf("neuer Schlüssel: %v", err)
	}

	geheimtext, err := VerschluesseleMit(alt, []byte("geheim"))
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}

	if _, err := EntschluesseleMit(neu, geheimtext); err == nil {
		t.Fatal("der neue Schlüssel konnte alten Bestand lesen; dann wäre AES-GCM kaputt")
	}

	// Nach dem Umschlüsseln liest ihn der neue.
	klartext, err := EntschluesseleMit(alt, geheimtext)
	if err != nil {
		t.Fatalf("mit altem Schlüssel entschlüsseln: %v", err)
	}
	umgeschluesselt, err := VerschluesseleMit(neu, klartext)
	if err != nil {
		t.Fatalf("neu verschlüsseln: %v", err)
	}
	zurueck, err := EntschluesseleMit(neu, umgeschluesselt)
	if err != nil {
		t.Fatalf("mit neuem Schlüssel entschlüsseln: %v", err)
	}
	if string(zurueck) != "geheim" {
		t.Errorf("zurück %q, erwartet %q", zurueck, "geheim")
	}
}

// VerschluesseleMit liefert zu gleichem Klartext verschiedene Ciphertexte, weil jeder Aufruf
// eine frische Nonce nimmt.
func TestVerschluesseleMitNimmtFrischeNonce(t *testing.T) {
	schluessel, err := SchluesselAus("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("Schlüssel: %v", err)
	}

	a, err := VerschluesseleMit(schluessel, []byte("gleich"))
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}
	b, err := VerschluesseleMit(schluessel, []byte("gleich"))
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}
	if bytes.Equal(a, b) {
		t.Error("zwei Verschlüsselungen desselben Klartexts sind gleich: Die Nonce wird wiederverwendet")
	}
}
