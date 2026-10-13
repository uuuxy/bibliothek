// Package crypto verschlüsselt und entschlüsselt mit AES-256-GCM unter dem Schlüssel aus
// APP_ENCRYPTION_KEY: die Fotos der Schüler und das Passwort des Mailservers. Ein Wechsel des
// Schlüssels arbeitet mit zwei ausdrücklich übergebenen Schlüsseln.
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

// SchluesselVariable ist der eine Name der Umgebungsvariable, unter dem der Schlüssel steht.
// Der Start prüft nur diese Variable auf Länge, Hex-Form und bekannte Beispielwerte. Ein
// zweiter Name ginge an den Prüfungen vorbei, und mit einem anderen Schlüssel als dem
// geprüften wären die Fotos und das Passwort des Mailservers nicht mehr zu entschlüsseln.
const SchluesselVariable = "APP_ENCRYPTION_KEY"

// FruehereVariable ist der frühere zweite Name. Das Programm liest ihn nicht; main bricht den
// Start ab, wenn er gesetzt ist und abweicht, damit ein Rest in der Umgebung auffällt.
const FruehereVariable = "ENCRYPTION_KEY"

// SchluesselAusUmgebung liefert den Schlüssel von 32 Byte aus APP_ENCRYPTION_KEY: 32 Zeichen
// Klartext oder 64 Zeichen Hex.
func SchluesselAusUmgebung() ([]byte, error) {
	wert := os.Getenv(SchluesselVariable)
	if wert == "" {
		return nil, errors.New(SchluesselVariable + " ist nicht gesetzt")
	}

	if len(wert) == 64 {
		// 64 Zeichen gelten als Schlüssel von 32 Byte in Hex.
		roh, err := hex.DecodeString(wert)
		if err != nil {
			return nil, fmt.Errorf("ungültiges Hex-Format im "+SchluesselVariable+": %w", err)
		}
		return roh, nil
	}

	if len(wert) == 32 {
		return []byte(wert), nil
	}

	return nil, errors.New(SchluesselVariable + " muss genau 32 Zeichen (oder 64 Hex-Zeichen) lang sein für AES-256")
}

// SchluesselAus liest einen Schlüssel aus einer Zeichenkette, nach denselben Regeln wie
// SchluesselAusUmgebung. Der Wechsel des Schlüssels (cmd/rotate-encryption-key) braucht den
// alten und den neuen zugleich im Speicher; über die Umgebungsvariable ginge das nicht.
func SchluesselAus(wert string) ([]byte, error) {
	if wert == "" {
		return nil, errors.New("schlüssel ist leer")
	}
	if len(wert) == 64 {
		roh, err := hex.DecodeString(wert)
		if err != nil {
			return nil, fmt.Errorf("ungültiges Hex-Format: %w", err)
		}
		return roh, nil
	}
	if len(wert) == 32 {
		return []byte(wert), nil
	}
	return nil, errors.New("schlüssel muss genau 32 Zeichen (oder 64 Hex-Zeichen) lang sein für AES-256")
}

// VerschluesseleMit verschlüsselt mit einem ausdrücklich übergebenen Schlüssel.
func VerschluesseleMit(schluessel, klartext []byte) ([]byte, error) {
	block, err := aes.NewCipher(schluessel)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren des AES-Ciphers: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren von GCM: %w", err)
	}
	// Jeder Aufruf nimmt eine frische Nonce aus dem Zufall.
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("fehler beim Generieren der Nonce: %w", err)
	}
	// Seal hängt den Ciphertext an die Nonce an, damit beides zusammen gespeichert wird.
	return gcm.Seal(nonce, nonce, klartext, nil), nil
}

// EntschluesseleMit entschlüsselt mit einem ausdrücklich übergebenen Schlüssel.
func EntschluesseleMit(schluessel, geheimtext []byte) ([]byte, error) {
	block, err := aes.NewCipher(schluessel)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren des AES-Ciphers: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Initialisieren von GCM: %w", err)
	}
	nonceLaenge := gcm.NonceSize()
	if len(geheimtext) < nonceLaenge {
		return nil, errors.New("ciphertext ist zu kurz, Nonce fehlt")
	}
	// Die Nonce steht vor dem Ciphertext.
	nonce, rest := geheimtext[:nonceLaenge], geheimtext[nonceLaenge:]
	klartext, err := gcm.Open(nil, nonce, rest, nil)
	if err != nil {
		return nil, fmt.Errorf("entschlüsselung fehlgeschlagen (falscher Schlüssel oder manipulierte Daten?): %w", err)
	}
	return klartext, nil
}

// Verschluessele verschlüsselt den Klartext mit AES-256-GCM unter dem Schlüssel der Umgebung.
func Verschluessele(klartext []byte) ([]byte, error) {
	schluessel, err := SchluesselAusUmgebung()
	if err != nil {
		return nil, err
	}
	return VerschluesseleMit(schluessel, klartext)
}

// Entschluessele entschlüsselt mit AES-256-GCM unter dem Schlüssel der Umgebung.
func Entschluessele(geheimtext []byte) ([]byte, error) {
	schluessel, err := SchluesselAusUmgebung()
	if err != nil {
		return nil, err
	}
	return EntschluesseleMit(schluessel, geheimtext)
}
