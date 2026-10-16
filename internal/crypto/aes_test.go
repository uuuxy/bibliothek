package crypto

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestSchluesselAusUmgebung(t *testing.T) {
	faelle := []struct {
		name        string
		wert        string
		wantLaenge  int
		wantFehler  bool
		wantMeldung string
	}{
		{
			name:        "nicht gesetzt",
			wert:        "",
			wantFehler:  true,
			wantMeldung: "nicht gesetzt",
		},
		{
			name:       "32-Byte-Klartext",
			wert:       "12345678901234567890123456789012",
			wantLaenge: 32,
		},
		{
			name:       "64 Hex-Zeichen",
			wert:       hex.EncodeToString([]byte("12345678901234567890123456789012")),
			wantLaenge: 32,
		},
		{
			name:        "kaputtes Hex",
			wert:        "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
			wantFehler:  true,
			wantMeldung: "ungültiges Hex-Format",
		},
		{
			name:        "falsche Länge",
			wert:        "short",
			wantFehler:  true,
			wantMeldung: "muss genau 32 Zeichen",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			t.Setenv(SchluesselVariable, f.wert)

			schluessel, err := SchluesselAusUmgebung()
			if (err != nil) != f.wantFehler {
				t.Errorf("SchluesselAusUmgebung() Fehler = %v, erwartet Fehler: %v", err, f.wantFehler)
				return
			}
			if f.wantFehler && !strings.Contains(err.Error(), f.wantMeldung) {
				t.Errorf("SchluesselAusUmgebung() Fehler = %v, erwartet mit %q", err, f.wantMeldung)
			}
			if !f.wantFehler && len(schluessel) != f.wantLaenge {
				t.Errorf("SchluesselAusUmgebung() Länge = %v, erwartet %v", len(schluessel), f.wantLaenge)
			}
		})
	}
}

// Der frühere zweite Name ENCRYPTION_KEY wird nicht gelesen. Gälte er, ginge er an jeder
// Prüfung des Starts vorbei (Länge, Hex-Form, Beispielwerte unter ENFORCE_PROD_SECRETS), und
// verschlüsselt würde mit einem anderen Schlüssel als dem geprüften: Die Fotos der Schüler und
// das Passwort des Mailservers wären danach nicht mehr lesbar.
func TestSchluesselAusUmgebung_LiestDenFrueherenNamenNicht(t *testing.T) {
	t.Setenv(FruehereVariable, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv(SchluesselVariable, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	schluessel, err := SchluesselAusUmgebung()
	if err != nil {
		t.Fatalf("SchluesselAusUmgebung() Fehler = %v", err)
	}
	if string(schluessel) != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("SchluesselAusUmgebung() nahm %q: Der frühere Name %s gilt wieder", schluessel, FruehereVariable)
	}

	// Ohne den gültigen Namen trägt der frühere nichts.
	t.Setenv(SchluesselVariable, "")
	if _, err := SchluesselAusUmgebung(); err == nil {
		t.Fatalf("%s allein hat gereicht: Er wird noch gelesen", FruehereVariable)
	}
}

func TestVerschluesseleUndEntschluessele(t *testing.T) {
	t.Setenv(SchluesselVariable, "12345678901234567890123456789012")

	klartext := []byte("secret message")

	geheimtext, err := Verschluessele(klartext)
	if err != nil {
		t.Fatalf("Verschluessele() Fehler = %v", err)
	}
	if len(geheimtext) == 0 {
		t.Fatal("Verschluessele() lieferte nichts")
	}

	entschluesselt, err := Entschluessele(geheimtext)
	if err != nil {
		t.Fatalf("Entschluessele() Fehler = %v", err)
	}

	if !bytes.Equal(klartext, entschluesselt) {
		t.Errorf("Entschluessele() = %v, erwartet %v", string(entschluesselt), string(klartext))
	}
}

func TestEntschluessele_Fehler(t *testing.T) {
	t.Setenv(SchluesselVariable, "12345678901234567890123456789012")

	t.Run("zu kurz", func(t *testing.T) {
		zuKurz := []byte("short")
		_, err := Entschluessele(zuKurz)
		if err == nil || !strings.Contains(err.Error(), "ciphertext ist zu kurz") {
			t.Errorf("Entschluessele(zu kurz) Fehler = %v, erwartet 'ciphertext ist zu kurz'", err)
		}
	})

	t.Run("verändert", func(t *testing.T) {
		klartext := []byte("secret message")
		geheimtext, err := Verschluessele(klartext)
		if err != nil {
			t.Fatalf("Verschluessele() Fehler = %v", err)
		}

		// Ein Bit im Ciphertext kippen. Die Nonce von GCM belegt die ersten 12 Byte; Stelle
		// 15 liegt dahinter.
		geheimtext[15] ^= 1

		_, err = Entschluessele(geheimtext)
		if err == nil || !strings.Contains(err.Error(), "entschlüsselung fehlgeschlagen") {
			t.Errorf("Entschluessele(verändert) Fehler = %v, erwartet 'entschlüsselung fehlgeschlagen'", err)
		}
	})
}

func TestVerschluesseleUndEntschluessele_OhneSchluessel(t *testing.T) {
	t.Setenv(SchluesselVariable, "")
	t.Setenv("APP_ENCRYPTION_KEY", "")

	klartext := []byte("secret message")

	_, err := Verschluessele(klartext)
	if err == nil {
		t.Error("Verschluessele() ohne Schlüssel: erwartet einen Fehler")
	}

	_, err = Entschluessele([]byte("some-ciphertext"))
	if err == nil {
		t.Error("Entschluessele() ohne Schlüssel: erwartet einen Fehler")
	}
}

func TestVerschluesseleMitUndEntschluesseleMit(t *testing.T) {
	schluessel := []byte("12345678901234567890123456789012")
	klartext := []byte("secret message for explicit key")

	geheimtext, err := VerschluesseleMit(schluessel, klartext)
	if err != nil {
		t.Fatalf("VerschluesseleMit() Fehler = %v", err)
	}
	if len(geheimtext) == 0 {
		t.Fatal("VerschluesseleMit() lieferte nichts")
	}

	entschluesselt, err := EntschluesseleMit(schluessel, geheimtext)
	if err != nil {
		t.Fatalf("EntschluesseleMit() Fehler = %v", err)
	}

	if !bytes.Equal(klartext, entschluesselt) {
		t.Errorf("EntschluesseleMit() = %v, erwartet %v", string(entschluesselt), string(klartext))
	}
}

func TestVerschluesseleMitUndEntschluesseleMit_Fehler(t *testing.T) {
	gueltig := []byte("12345678901234567890123456789012")
	ungueltig := []byte("short")
	klartext := []byte("secret")

	t.Run("VerschluesseleMit mit ungültigem Schlüssel", func(t *testing.T) {
		_, err := VerschluesseleMit(ungueltig, klartext)
		if err == nil {
			t.Error("VerschluesseleMit() mit ungültigem Schlüssel: erwartet einen Fehler")
		}
	})

	t.Run("EntschluesseleMit mit ungültigem Schlüssel", func(t *testing.T) {
		_, err := EntschluesseleMit(ungueltig, []byte("some-ciphertext-long-enough-for-nonce"))
		if err == nil {
			t.Error("EntschluesseleMit() mit ungültigem Schlüssel: erwartet einen Fehler")
		}
	})

	t.Run("EntschluesseleMit zu kurz", func(t *testing.T) {
		zuKurz := []byte("short")
		_, err := EntschluesseleMit(gueltig, zuKurz)
		if err == nil || !strings.Contains(err.Error(), "ciphertext ist zu kurz") {
			t.Errorf("EntschluesseleMit(zu kurz) Fehler = %v, erwartet 'ciphertext ist zu kurz'", err)
		}
	})

	t.Run("EntschluesseleMit verändert", func(t *testing.T) {
		geheimtext, err := VerschluesseleMit(gueltig, klartext)
		if err != nil {
			t.Fatalf("VerschluesseleMit() Fehler = %v", err)
		}

		// Ein Bit im Ciphertext kippen. Die Nonce von GCM belegt die ersten 12 Byte; Stelle
		// 15 liegt dahinter.
		geheimtext[15] ^= 1

		_, err = EntschluesseleMit(gueltig, geheimtext)
		if err == nil || !strings.Contains(err.Error(), "entschlüsselung fehlgeschlagen") {
			t.Errorf("EntschluesseleMit(verändert) Fehler = %v, erwartet 'entschlüsselung fehlgeschlagen'", err)
		}
	})
}
