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
		envKey     string
		appEnvKey  string
		wantLen    int
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "Missing env vars",
			envKey:     "",
			appEnvKey:  "",
			wantErr:    true,
			wantErrMsg: "nicht gesetzt",
		},
		{
			name:      "32-byte string in ENCRYPTION_KEY",
			envKey:    "12345678901234567890123456789012",
			appEnvKey: "",
			wantLen:   32,
			wantErr:   false,
		},
		{
			name:      "32-byte string in APP_ENCRYPTION_KEY",
			envKey:    "",
			appEnvKey: "12345678901234567890123456789012",
			wantLen:   32,
			wantErr:   false,
		},
		{
			name:      "64-byte hex string in ENCRYPTION_KEY",
			envKey:    hex.EncodeToString([]byte("12345678901234567890123456789012")),
			appEnvKey: "",
			wantLen:   32,
			wantErr:   false,
		},
		{
			name:       "Invalid hex string",
			envKey:     "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
			appEnvKey:  "",
			wantErr:    true,
			wantErrMsg: "ungültiges Hex-Format",
		},
		{
			name:       "Invalid length string",
			envKey:     "short",
			appEnvKey:  "",
			wantErr:    true,
			wantErrMsg: "muss genau 32 Zeichen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv("ENCRYPTION_KEY", tt.envKey)
			} else {
				t.Setenv("ENCRYPTION_KEY", "")
			}

			if tt.appEnvKey != "" {
				t.Setenv("APP_ENCRYPTION_KEY", tt.appEnvKey)
			} else {
				t.Setenv("APP_ENCRYPTION_KEY", "")
			}

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

func TestEncryptDecrypt(t *testing.T) {
	// Set a valid 32-byte key for encryption/decryption tests
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")

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
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")

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
	t.Setenv("ENCRYPTION_KEY", "")
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
