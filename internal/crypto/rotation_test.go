package crypto

import (
	"bytes"
	"strings"
	"testing"
)

// Der Schlüsselwechsel steht und fällt damit, dass EncryptMit/DecryptMit sich exakt
// so verhalten wie Encrypt/Decrypt — nur mit ausdrücklich übergebenem Schlüssel. Wäre
// das nicht so, produzierte cmd/rotate-encryption-key einen Bestand, den die Anwendung
// hinterher nicht mehr lesen kann. Und zwar erst dann sichtbar, wenn jemand ein
// Schülerfoto öffnet.

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
			key, err := SchluesselAus(f.wert)
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
			if len(key) != f.laenge {
				t.Errorf("Schlüssellänge %d, erwartet %d", len(key), f.laenge)
			}
		})
	}
}

func TestEncryptMitUndDecryptMitSindGegenlaeufig(t *testing.T) {
	key, err := SchluesselAus("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("Schlüssel: %v", err)
	}

	klartext := []byte("ein Schülerfoto oder ein SMTP-Passwort")
	ct, err := EncryptMit(key, klartext)
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}
	if bytes.Contains(ct, klartext) {
		t.Fatal("der Klartext steht unverändert im Ciphertext")
	}

	zurueck, err := DecryptMit(key, ct)
	if err != nil {
		t.Fatalf("entschlüsseln: %v", err)
	}
	if !bytes.Equal(zurueck, klartext) {
		t.Errorf("zurück %q, erwartet %q", zurueck, klartext)
	}
}

// Der Kern des Wechsels: Was mit dem alten Schlüssel geschrieben wurde, liest der neue
// NICHT. Genau deshalb muss rotate-encryption-key jeden Datensatz anfassen — und genau
// deshalb ist ein bloßer Tausch der Umgebungsvariable Datenverlust.
func TestNeuerSchluesselLiestAltenBestandNicht(t *testing.T) {
	alt, err := SchluesselAus("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("alter Schlüssel: %v", err)
	}
	neu, err := SchluesselAus("abcdefghijklmnopqrstuvwxyz012345")
	if err != nil {
		t.Fatalf("neuer Schlüssel: %v", err)
	}

	ct, err := EncryptMit(alt, []byte("geheim"))
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}

	if _, err := DecryptMit(neu, ct); err == nil {
		t.Fatal("der neue Schlüssel konnte alten Bestand lesen — dann wäre AES-GCM kaputt")
	}

	// Und nach der Umschlüsselung liest ihn der neue sehr wohl.
	klartext, err := DecryptMit(alt, ct)
	if err != nil {
		t.Fatalf("mit altem Schlüssel entschlüsseln: %v", err)
	}
	umgeschluesselt, err := EncryptMit(neu, klartext)
	if err != nil {
		t.Fatalf("neu verschlüsseln: %v", err)
	}
	zurueck, err := DecryptMit(neu, umgeschluesselt)
	if err != nil {
		t.Fatalf("mit neuem Schlüssel entschlüsseln: %v", err)
	}
	if string(zurueck) != "geheim" {
		t.Errorf("zurück %q, erwartet %q", zurueck, "geheim")
	}
}

// EncryptMit muss bei gleichem Klartext verschiedene Ciphertexte liefern (frische Nonce).
func TestEncryptMitNutztFrischeNonce(t *testing.T) {
	key, err := SchluesselAus("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("Schlüssel: %v", err)
	}

	a, err := EncryptMit(key, []byte("gleich"))
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}
	b, err := EncryptMit(key, []byte("gleich"))
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}
	if bytes.Equal(a, b) {
		t.Error("zwei Verschlüsselungen desselben Klartexts sind identisch — die Nonce wird wiederverwendet")
	}
}
