// Package backupkrypto leitet den Schlüssel der verschlüsselten Backups ab und kennt ihr
// Dateiformat.
//
// Es hängt nur an der Standardbibliothek und golang.org/x/crypto/scrypt. So lassen sich die
// Werkzeuge, die nur ver- oder entschlüsseln (cmd/encrypt-backup, cmd/restore-backup), ohne
// cgo bauen; das Paket jobs zieht die WebP-Bibliothek mit und braucht cgo.
//
// Eine Backup-Datei enthält den ganzen Bestand (Namen, Adressen, Geburtsdaten und die
// Ausleihen Minderjähriger) und liegt auch außer Haus. Wer eine Datei erlangt, kann
// Passphrasen ohne Bremse durchprobieren. Der Schlüssel wird deshalb mit scrypt abgeleitet
// (N=2^15, r=8, p=1), je Datei mit eigenem Salt von 16 Byte: Das verteuert jeden Versuch um
// Größenordnungen und macht eine Vorberechnung wertlos. Eine Entschlüsselung kostet rund
// 100 ms.
//
// Das Dateiformat trägt eine Kennung und eine Fassung, damit sich ein späterer Wechsel der
// Ableitung unterscheiden lässt:
//
//	"BKDF" 0x02 [16B Salt] [12B GCM-Nonce] [Ciphertext+Tag]   → scrypt
//
// Eine Datei ohne diese Kennung wird abgelehnt. Die frühere Ableitung mit einem Durchlauf
// SHA-256 ohne Salt liest das Programm nicht mehr, weil sie als Ableitung aus einer Passphrase
// schwach ist.
package backupkrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/scrypt"
)

var kennung = []byte("BKDF")

const (
	formatScrypt = 0x02
	saltLaenge   = 16
	// N=2^15 ist die übliche Härte für eine Ableitung, auf die jemand wartet (rund 100 ms und
	// 32 MB). Dieselbe Ableitung läuft in der wöchentlichen Restore-Probe und muss dort
	// tragbar bleiben.
	scryptN = 1 << 15
	scryptR = 8
	scryptP = 1
)

// leiteScryptAb liefert den AES-Schlüssel von 32 Byte aus Passphrase und Salt.
func leiteScryptAb(passphrase string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, 32)
}

// Verschluessele verschlüsselt den gepackten Dump im Format des Pakets: Kennung, Fassung,
// Salt, Nonce, Ciphertext mit Tag. Werkzeuge und Tests erzeugen damit gültige Backups, ohne
// das Format nachzubauen.
func Verschluessele(passphrase string, klartext []byte) ([]byte, error) {
	salt := make([]byte, saltLaenge)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("salt erzeugen: %w", err)
	}
	schluessel, err := leiteScryptAb(passphrase, salt)
	if err != nil {
		return nil, fmt.Errorf("scrypt-Ableitung: %w", err)
	}
	gcm, err := newGCM(schluessel)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce erzeugen: %w", err)
	}
	// Der Kopf ist Kennung, Fassung, Salt und Nonce. Seal hängt Ciphertext und Tag an ihn an;
	// die Nonce fügt es nicht selbst hinzu.
	kopf := append(append([]byte{}, kennung...), formatScrypt)
	kopf = append(kopf, salt...)
	kopf = append(kopf, nonce...)
	return gcm.Seal(kopf, nonce, klartext, nil), nil
}

// Entschluessele öffnet eine Datei im Format des Pakets. Eine Datei ohne die Kennung wird
// abgelehnt.
func Entschluessele(passphrase string, daten []byte) ([]byte, error) {
	if !IstScryptFormat(daten) {
		return nil, fmt.Errorf("kein gültiges Backup: scrypt-Format-Kennung fehlt (Dateien von vor der scrypt-Umstellung werden nicht unterstützt)")
	}
	rest := daten[len(kennung)+1:]
	if len(rest) < saltLaenge {
		return nil, fmt.Errorf("backup zu kurz: Salt fehlt")
	}
	salt, hinterSalt := rest[:saltLaenge], rest[saltLaenge:]
	schluessel, err := leiteScryptAb(passphrase, salt)
	if err != nil {
		return nil, fmt.Errorf("scrypt-Ableitung: %w", err)
	}
	return oeffneGCM(schluessel, hinterSalt)
}

// IstScryptFormat prüft Kennung und Fassung am Anfang der Datei. Dass eine Datei im früheren
// Format zufällig so beginnt, hat die Wahrscheinlichkeit 2^-40.
func IstScryptFormat(daten []byte) bool {
	if len(daten) < len(kennung)+1 {
		return false
	}
	for i, b := range kennung {
		if daten[i] != b {
			return false
		}
	}
	return daten[len(kennung)] == formatScrypt
}

// newGCM liefert AES-GCM unter dem Schlüssel.
func newGCM(schluessel []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(schluessel)
	if err != nil {
		return nil, fmt.Errorf("aes-Cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// oeffneGCM entschlüsselt daten, die mit der Nonce beginnen; dahinter folgen Ciphertext und
// Tag.
func oeffneGCM(schluessel, daten []byte) ([]byte, error) {
	gcm, err := newGCM(schluessel)
	if err != nil {
		return nil, err
	}
	if len(daten) < gcm.NonceSize() {
		return nil, fmt.Errorf("backup zu kurz: Nonce fehlt")
	}
	nonce, geheimtext := daten[:gcm.NonceSize()], daten[gcm.NonceSize():]
	return gcm.Open(nil, nonce, geheimtext, nil)
}
