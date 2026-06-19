package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMasterKey(t *testing.T) {
	t.Run("APP_ENCRYPTION_KEY is not set", func(t *testing.T) {
		t.Setenv("APP_ENCRYPTION_KEY", "")
		key, err := GetMasterKey()
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), "ist nicht gesetzt")
	})

	t.Run("APP_ENCRYPTION_KEY has valid 32 characters", func(t *testing.T) {
		keyStr := "12345678901234567890123456789012"
		t.Setenv("APP_ENCRYPTION_KEY", keyStr)
		key, err := GetMasterKey()
		assert.NoError(t, err)
		assert.Equal(t, []byte(keyStr), key)
	})

	t.Run("APP_ENCRYPTION_KEY has valid 64 hex characters", func(t *testing.T) {
		keyStr := "12345678901234567890123456789012"
		hexKey := hex.EncodeToString([]byte(keyStr))
		t.Setenv("APP_ENCRYPTION_KEY", hexKey)
		key, err := GetMasterKey()
		assert.NoError(t, err)
		assert.Equal(t, []byte(keyStr), key)
	})

	t.Run("APP_ENCRYPTION_KEY has invalid length", func(t *testing.T) {
		t.Setenv("APP_ENCRYPTION_KEY", "1234567890")
		key, err := GetMasterKey()
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), "muss genau 32 Zeichen")
	})

	t.Run("APP_ENCRYPTION_KEY has invalid hex format", func(t *testing.T) {
		// 64 characters but invalid hex
		invalidHex := "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"
		t.Setenv("APP_ENCRYPTION_KEY", invalidHex)
		key, err := GetMasterKey()
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), "ungültiges Hex-Format")
	})
}

func TestEncryptDecrypt(t *testing.T) {
	keyStr := "12345678901234567890123456789012"

	t.Run("Happy path", func(t *testing.T) {
		t.Setenv("APP_ENCRYPTION_KEY", keyStr)

		plaintext := []byte("secret message")

		ciphertext, err := Encrypt(plaintext)
		assert.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.False(t, bytes.Equal(plaintext, ciphertext))

		decrypted, err := Decrypt(ciphertext)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("Missing key during Encrypt", func(t *testing.T) {
		t.Setenv("APP_ENCRYPTION_KEY", "")
		ciphertext, err := Encrypt([]byte("secret message"))
		assert.Error(t, err)
		assert.Nil(t, ciphertext)
	})

	t.Run("Missing key during Decrypt", func(t *testing.T) {
		t.Setenv("APP_ENCRYPTION_KEY", "")
		plaintext, err := Decrypt([]byte("some ciphertext"))
		assert.Error(t, err)
		assert.Nil(t, plaintext)
	})

	t.Run("Ciphertext too short", func(t *testing.T) {
		t.Setenv("APP_ENCRYPTION_KEY", keyStr)

		// GCM Nonce is usually 12 bytes
		shortCiphertext := []byte("short")
		plaintext, err := Decrypt(shortCiphertext)
		assert.Error(t, err)
		assert.Nil(t, plaintext)
		assert.Contains(t, err.Error(), "ciphertext ist zu kurz")
	})

	t.Run("Tampered ciphertext", func(t *testing.T) {
		t.Setenv("APP_ENCRYPTION_KEY", keyStr)

		plaintext := []byte("secret message")
		ciphertext, err := Encrypt(plaintext)
		assert.NoError(t, err)

		// Tamper with the ciphertext (change the last byte)
		ciphertext[len(ciphertext)-1] ^= 0xFF

		decrypted, err := Decrypt(ciphertext)
		assert.Error(t, err)
		assert.Nil(t, decrypted)
		assert.Contains(t, err.Error(), "entschlüsselung fehlgeschlagen")
	})
}
