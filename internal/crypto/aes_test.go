package crypto

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestGetMasterKey(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		wantLen    int
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "nicht gesetzt",
			key:        "",
			wantErr:    true,
			wantErrMsg: "nicht gesetzt",
		},
		{
			name:    "32-Byte-Klartext",
			key:     "12345678901234567890123456789012",
			wantLen: 32,
		},
		{
			name:    "64 Hex-Zeichen",
			key:     hex.EncodeToString([]byte("12345678901234567890123456789012")),
			wantLen: 32,
		},
		{
			name:       "kaputtes Hex",
			key:        "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
			wantErr:    true,
			wantErrMsg: "ungültiges Hex-Format",
		},
		{
			name:       "falsche Länge",
			key:        "short",
			wantErr:    true,
			wantErrMsg: "muss genau 32 Zeichen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(SchluesselVariable, tt.key)

			key, err := GetMasterKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMasterKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("GetMasterKey() error = %v, wantErrMsg %v", err, tt.wantErrMsg)
			}
			if !tt.wantErr && len(key) != tt.wantLen {
				t.Errorf("GetMasterKey() len = %v, want %v", len(key), tt.wantLen)
			}
		})
	}
}

// Das Gate gegen die zweite Tür: ENCRYPTION_KEY wurde bis zum 06.08.2026 VORRANGIG
// gelesen und umging damit jede Startprüfung (Länge, Hex-Form, Default-Erkennung unter
// ENFORCE_PROD_SECRETS). Verschlüsselt worden wäre dann mit einem anderen Schlüssel als
// dem geprüften — Schülerfotos und das gespeicherte SMTP-Passwort wären still
// unlesbar geworden.
//
// Fällt dieser Test, ist der Zweitname zurück.
func TestGetMasterKeyIgnoriertDenAltnamen(t *testing.T) {
	t.Setenv(AltName, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv(SchluesselVariable, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	key, err := GetMasterKey()
	if err != nil {
		t.Fatalf("GetMasterKey() error = %v", err)
	}
	if string(key) != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("GetMasterKey() nahm %q — der Altname %s gewinnt wieder", key, AltName)
	}

	// Und ohne den gültigen Namen trägt der Altname gar nichts.
	t.Setenv(SchluesselVariable, "")
	if _, err := GetMasterKey(); err == nil {
		t.Fatalf("%s allein hat gereicht — er wird noch gelesen", AltName)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	// Set a valid 32-byte key for encryption/decryption tests
	t.Setenv(SchluesselVariable, "12345678901234567890123456789012")

	plaintext := []byte("secret message")

	// Test Encrypt
	ciphertext, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("Encrypt() returned empty ciphertext")
	}

	// Test Decrypt
	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypt() = %v, want %v", string(decrypted), string(plaintext))
	}
}

func TestDecryptErrors(t *testing.T) {
	// Set a valid 32-byte key
	t.Setenv(SchluesselVariable, "12345678901234567890123456789012")

	t.Run("Short ciphertext", func(t *testing.T) {
		shortCiphertext := []byte("short")
		_, err := Decrypt(shortCiphertext)
		if err == nil || !strings.Contains(err.Error(), "ciphertext ist zu kurz") {
			t.Errorf("Decrypt(short) error = %v, want 'ciphertext ist zu kurz'", err)
		}
	})

	t.Run("Tampered data", func(t *testing.T) {
		plaintext := []byte("secret message")
		ciphertext, err := Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}

		// Tamper with the ciphertext (flip a bit in the actual ciphertext, after the nonce)
		// Nonce size is 12 for GCM
		ciphertext[15] ^= 1

		_, err = Decrypt(ciphertext)
		if err == nil || !strings.Contains(err.Error(), "entschlüsselung fehlgeschlagen") {
			t.Errorf("Decrypt(tampered) error = %v, want 'entschlüsselung fehlgeschlagen'", err)
		}
	})
}

func TestEncryptDecryptNoKey(t *testing.T) {
	// Ensure no key is set
	t.Setenv(SchluesselVariable, "")
	t.Setenv("APP_ENCRYPTION_KEY", "")

	plaintext := []byte("secret message")

	_, err := Encrypt(plaintext)
	if err == nil {
		t.Error("Encrypt() expected error without key")
	}

	_, err = Decrypt([]byte("some-ciphertext"))
	if err == nil {
		t.Error("Decrypt() expected error without key")
	}
}

func TestDecryptMit(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	plaintext := []byte("secret message")

	ciphertext, err := EncryptMit(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptMit() error = %v", err)
	}

	decrypted, err := DecryptMit(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptMit() error = %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("DecryptMit() = %v, want %v", string(decrypted), string(plaintext))
	}
}

func TestDecryptMitErrors(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")
	invalidKey := []byte("short")

	t.Run("Invalid key length", func(t *testing.T) {
		_, err := DecryptMit(invalidKey, []byte("some-ciphertext"))
		if err == nil || !strings.Contains(err.Error(), "fehler beim Initialisieren des AES-Ciphers") {
			t.Errorf("DecryptMit(invalidKey) error = %v, want 'fehler beim Initialisieren des AES-Ciphers'", err)
		}
	})

	t.Run("Short ciphertext", func(t *testing.T) {
		shortCiphertext := []byte("short")
		_, err := DecryptMit(validKey, shortCiphertext)
		if err == nil || !strings.Contains(err.Error(), "ciphertext ist zu kurz, Nonce fehlt") {
			t.Errorf("DecryptMit(shortCiphertext) error = %v, want 'ciphertext ist zu kurz, Nonce fehlt'", err)
		}
	})

	t.Run("Tampered data", func(t *testing.T) {
		plaintext := []byte("secret message")
		ciphertext, err := EncryptMit(validKey, plaintext)
		if err != nil {
			t.Fatalf("EncryptMit() error = %v", err)
		}

		ciphertext[15] ^= 1

		_, err = DecryptMit(validKey, ciphertext)
		if err == nil || !strings.Contains(err.Error(), "entschlüsselung fehlgeschlagen") {
			t.Errorf("DecryptMit(tampered) error = %v, want 'entschlüsselung fehlgeschlagen'", err)
		}
	})
}
