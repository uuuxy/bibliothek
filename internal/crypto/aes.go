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

// SchluesselVariable ist der EINE Name, unter dem der Master-Schlüssel steht.
//
// Er hieß hier bis zum 06.08.2026 wahlweise ENCRYPTION_KEY ODER APP_ENCRYPTION_KEY,
// und zwar mit ENCRYPTION_KEY zuerst. Das war eine zweite Tür zu demselben Zustand,
// und die gefährlichere: Geprüft wird beim Start nur APP_ENCRYPTION_KEY — auf Länge,
// auf Hex-Form und (mit ENFORCE_PROD_SECRETS) gegen die bekannten Default-Werte. Ein
// gesetztes ENCRYPTION_KEY hätte diese Prüfungen allesamt umgangen und still gewonnen.
//
// Der Schaden wäre nicht sichtbar gewesen, sondern still: Verschlüsselt und
// entschlüsselt würde mit einem anderen Schlüssel als dem geprüften. Schülerfotos und
// das gespeicherte SMTP-Passwort sind dann nicht "falsch", sondern **weg** — bis der
// Ursprungswert wiederauftaucht. Im ausgelieferten Stack konnte das nicht passieren
// (docker-compose.yml reicht die Variable gar nicht durch); der Fall wäre ein
// Reststück in der Umgebung eines Servers, auf dem noch etwas anderes läuft.
const SchluesselVariable = "APP_ENCRYPTION_KEY"

// AltName ist der frühere Zweitname. Er wird NICHT mehr gelesen — main.go bricht den
// Start ab, wenn er abweichend gesetzt ist, damit ein Reststück in der Umgebung
// auffällt statt still ignoriert zu werden.
const AltName = "ENCRYPTION_KEY"

// GetMasterKey lädt den 32-Byte-Master-Schlüssel aus APP_ENCRYPTION_KEY.
// Unterstützt sowohl 32-stellige Klartext-Strings als auch 64-stellige Hex-kodierte Strings.
func GetMasterKey() ([]byte, error) {
	keyStr := os.Getenv(SchluesselVariable)
	if keyStr == "" {
		return nil, errors.New(SchluesselVariable + " ist nicht gesetzt")
	}

	if len(keyStr) == 64 {
		// Annahme: Hex-codierter 32-Byte Schlüssel
		decoded, err := hex.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("ungültiges Hex-Format im "+SchluesselVariable+": %w", err)
		}
		return decoded, nil
	}

	if len(keyStr) == 32 {
		return []byte(keyStr), nil
	}

	return nil, errors.New(SchluesselVariable + " muss genau 32 Zeichen (oder 64 Hex-Zeichen) lang sein für AES-256")
}

// SchluesselAus liest einen Master-Schlüssel aus einer beliebigen Zeichenkette, mit
// denselben Regeln wie GetMasterKey (32 Zeichen oder 64 Hex-Zeichen).
//
// Gebraucht wird das für den Schlüsselwechsel (cmd/rotate-encryption-key): Dort müssen
// ALTER und NEUER Schlüssel gleichzeitig im Speicher liegen — ein Umweg über die
// Umgebungsvariable wäre nicht nur umständlich, sondern schlicht nicht möglich.
func SchluesselAus(wert string) ([]byte, error) {
	if wert == "" {
		return nil, errors.New("schlüssel ist leer")
	}
	if len(wert) == 64 {
		decoded, err := hex.DecodeString(wert)
		if err != nil {
			return nil, fmt.Errorf("ungültiges Hex-Format: %w", err)
		}
		return decoded, nil
	}
	if len(wert) == 32 {
		return []byte(wert), nil
	}
	return nil, errors.New("schlüssel muss genau 32 Zeichen (oder 64 Hex-Zeichen) lang sein für AES-256")
}

// EncryptMit verschlüsselt mit einem ausdrücklich übergebenen Schlüssel.
func EncryptMit(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren des AES-Ciphers: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren von GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("fehler beim Generieren der Nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptMit entschlüsselt mit einem ausdrücklich übergebenen Schlüssel.
func DecryptMit(key, ciphertext []byte) ([]byte, error) {
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
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("entschlüsselung fehlgeschlagen (falscher Schlüssel oder manipulierte Daten?): %w", err)
	}
	return plaintext, nil
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
