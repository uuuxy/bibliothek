package jobs

// backup_krypto.go — Schlüsselableitung und Dateiformat der verschlüsselten Backups.
//
// Angreifermodell: Eine Backup-Datei enthält den GESAMTEN Bestand — Klarnamen,
// Adressen, Geburtsdaten und Ausleihhistorie Minderjähriger — und liegt zusätzlich
// außer Haus (S3). Wer eine Datei erlangt, kann Passphrase-Kandidaten OFFLINE
// durchprobieren, ohne dass irgendein Server das bremst oder bemerkt.
//
// Bis 21.08.2026 war der AES-Schlüssel `SHA256(passphrase)` — ein einziger
// Hash-Durchlauf, ohne Salt. Auf GPU sind das Milliarden Versuche/Sekunde: Eine
// gemerkte 32-Zeichen-Passphrase (Länge ≥ 32 war die einzige Prüfung, und Länge ist
// nicht Entropie) fällt damit in praktikabler Zeit, und ohne Salt ließe sich die
// Ableitung über alle Dateien hinweg vorberechnen.
//
// Jetzt leitet scrypt ab (N=2^15, r=8, p=1) — speicherhart, pro Datei mit eigenem
// 16-Byte-Salt. Das macht jeden Rateversuch um Größenordnungen teurer und die
// Vorberechnung wertlos. Für die seltene, legitime Entschlüsselung (Restore, Probe)
// kostet es einmalig ~100 ms — kein Faktor.
//
// Dateiformat, rückwärtskompatibel (der Grund, warum der Wechsel gefahrlos ist —
// im Gegensatz zu dem, was der alte Kommentar behauptete):
//
//	NEU:  "BKDF" 0x02 [16B Salt] [12B GCM-Nonce] [Ciphertext+Tag]   → scrypt
//	ALT:  [12B GCM-Nonce] [Ciphertext+Tag]                          → SHA-256
//
// Der Lesepfad erkennt das Magic und wählt die Ableitung; alte Backups bleiben also
// öffenbar. Neue werden ausschließlich im starken Format geschrieben.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/scrypt"
)

var backupMagic = []byte("BKDF")

const (
	backupFormatScrypt = 0x02
	backupSaltLaenge   = 16
	// scrypt-Parameter: N=2^15 ist der gängige „interaktive" Härtegrad (~100 ms,
	// ~32 MB). Höher ginge, doch dieselbe Ableitung läuft in der wöchentlichen
	// Restore-Probe — die Kosten müssen für den legitimen Fall tragbar bleiben.
	scryptN = 1 << 15
	scryptR = 8
	scryptP = 1
)

// leiteScryptAb erzeugt den 32-Byte-AES-Schlüssel aus Passphrase und Salt.
func leiteScryptAb(passphrase string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, 32)
}

// verschluesseleBackup verschlüsselt den komprimierten Dump im starken Format.
// Ausgabe: backupMagic ‖ 0x02 ‖ Salt ‖ Nonce ‖ Ciphertext+Tag.
func verschluesseleBackup(passphrase string, klartext []byte) ([]byte, error) {
	salt := make([]byte, backupSaltLaenge)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("salt erzeugen: %w", err)
	}
	key, err := leiteScryptAb(passphrase, salt)
	if err != nil {
		return nil, fmt.Errorf("scrypt-Ableitung: %w", err)
	}
	gcm, err := neuesGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce erzeugen: %w", err)
	}
	// Kopf = Magic ‖ Format ‖ Salt ‖ Nonce. Seal hängt Ciphertext+Tag an diesen Kopf
	// an (es fügt die Nonce NICHT selbst hinzu) — Ergebnis: der komplette Rahmen.
	kopf := append(append([]byte{}, backupMagic...), backupFormatScrypt)
	kopf = append(kopf, salt...)
	kopf = append(kopf, nonce...)
	return gcm.Seal(kopf, nonce, klartext, nil), nil
}

// entschluesseleBackupDaten öffnet beide Formate. Trägt die Datei das Magic, wird
// scrypt mit dem eingebetteten Salt verwendet; sonst der alte SHA-256-Weg.
func entschluesseleBackupDaten(passphrase string, daten []byte) ([]byte, error) {
	if istScryptFormat(daten) {
		rest := daten[len(backupMagic)+1:]
		if len(rest) < backupSaltLaenge {
			return nil, fmt.Errorf("backup zu kurz: Salt fehlt")
		}
		salt, hinterSalt := rest[:backupSaltLaenge], rest[backupSaltLaenge:]
		key, err := leiteScryptAb(passphrase, salt)
		if err != nil {
			return nil, fmt.Errorf("scrypt-Ableitung: %w", err)
		}
		return oeffneGCM(key, hinterSalt)
	}
	// Altes Format: Schlüssel = SHA-256(passphrase), Rest = Nonce ‖ Ciphertext.
	keyBytes := sha256.Sum256([]byte(passphrase))
	return oeffneGCM(keyBytes[:], daten)
}

// istScryptFormat prüft das Magic am Dateianfang. Die Kollision mit einer alten
// Datei, deren zufällige Nonce genau mit "BKDF"+0x02 beginnt, liegt bei 2^-40 pro
// Datei — praktisch ausgeschlossen.
func istScryptFormat(daten []byte) bool {
	if len(daten) < len(backupMagic)+1 {
		return false
	}
	for i, b := range backupMagic {
		if daten[i] != b {
			return false
		}
	}
	return daten[len(backupMagic)] == backupFormatScrypt
}

func neuesGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes-Cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// oeffneGCM erwartet daten = Nonce ‖ Ciphertext+Tag.
func oeffneGCM(key, daten []byte) ([]byte, error) {
	gcm, err := neuesGCM(key)
	if err != nil {
		return nil, err
	}
	if len(daten) < gcm.NonceSize() {
		return nil, fmt.Errorf("backup zu kurz: Nonce fehlt")
	}
	nonce, ct := daten[:gcm.NonceSize()], daten[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}
