package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// GetMasterKey lädt den 32-Byte Master-Schlüssel aus der Umgebungsvariable ENCRYPTION_KEY oder APP_ENCRYPTION_KEY.
// Unterstützt sowohl 32-stellige Klartext-Strings als auch 64-stellige Hex-kodierte Strings.
func GetMasterKey() ([]byte, error) {
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		keyStr = os.Getenv("APP_ENCRYPTION_KEY")
	}
	if keyStr == "" {
		return nil, errors.New("ENCRYPTION_KEY (oder APP_ENCRYPTION_KEY) ist nicht gesetzt")
	}

	if len(keyStr) == 64 {
		// Annahme: Hex-codierter 32-Byte Schlüssel
		decoded, err := hex.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("ungültiges Hex-Format im APP_ENCRYPTION_KEY: %w", err)
		}
		return decoded, nil
	}

	if len(keyStr) == 32 {
		return []byte(keyStr), nil
	}

	return nil, errors.New("ENCRYPTION_KEY muss genau 32 Zeichen (oder 64 Hex-Zeichen) lang sein für AES-256")
}

// Encrypt verschlüsselt den übergebenen Plaintext mit AES-256-GCM.
func Encrypt(plaintext []byte) ([]byte, error) {
	key, err := GetMasterKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren des AES-Ciphers: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren von GCM: %w", err)
	}

	// Nonce (IV) mit Zufallswerten generieren
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("fehler beim Generieren der Nonce: %w", err)
	}

	// Seal hängt den Ciphertext an die Nonce an, damit beides zusammen gespeichert wird
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt entschlüsselt den übergebenen Ciphertext mit AES-256-GCM.
func Decrypt(ciphertext []byte) ([]byte, error) {
	key, err := GetMasterKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren des AES-Ciphers: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren von GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext ist zu kurz, Nonce fehlt")
	}

	// Nonce extrahieren und Ciphertext entschlüsseln
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("entschlüsselung fehlgeschlagen (falscher Schlüssel oder manipulierte Daten?): %w", err)
	}

	return plaintext, nil
}
