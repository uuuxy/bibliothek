package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Der Prüfwert eines Passworts schließt die Sperre nach Inaktivität auf, wenn der Mailserver
// der Schule nicht erreichbar ist. Vor Argon2id steht ein HMAC mit einem Schlüssel, der nicht
// in der Datenbank liegt: Mit der Datenbank oder einer Sicherung allein lässt sich kein
// Passwort durchprobieren.
const (
	pruefwertKennung   = "argon2id"
	pruefwertSpeicher  = 19 * 1024 // KiB
	pruefwertRunden    = 2
	pruefwertFaeden    = 1
	pruefwertSalzBytes = 16
	pruefwertBytes     = 32
)

var errPruefwertUnlesbar = errors.New("prüfwert hat kein bekanntes format")

// pruefwertSchluessel leitet den Schlüssel der ersten Stufe aus dem Token-Geheimnis ab, damit
// der Prüfwert nicht denselben Schlüssel benutzt wie die Signatur der Tokens.
func pruefwertSchluessel(geheimnis []byte) []byte {
	mac := hmac.New(sha256.New, geheimnis)
	mac.Write([]byte("bibliothek/sitzung/passwort-pruefwert/v1"))
	return mac.Sum(nil)
}

func pruefwertVorstufe(schluessel []byte, passwort string) []byte {
	mac := hmac.New(sha256.New, schluessel)
	mac.Write([]byte(passwort))
	return mac.Sum(nil)
}

// bildePruefwert liefert den Prüfwert als Text: Kennung, Parameter, Salz und Wert, getrennt
// durch „$". Die Parameter stehen dabei, damit ein später geänderter Aufwand alte Werte
// weiter prüfen kann.
func bildePruefwert(schluessel []byte, passwort string) (string, error) {
	salz := make([]byte, pruefwertSalzBytes)
	if _, err := rand.Read(salz); err != nil {
		return "", fmt.Errorf("salz erzeugen: %w", err)
	}
	wert := argon2.IDKey(pruefwertVorstufe(schluessel, passwort), salz,
		pruefwertRunden, pruefwertSpeicher, pruefwertFaeden, pruefwertBytes)
	return strings.Join([]string{
		pruefwertKennung,
		strconv.Itoa(pruefwertSpeicher),
		strconv.Itoa(pruefwertRunden),
		strconv.Itoa(pruefwertFaeden),
		base64.RawStdEncoding.EncodeToString(salz),
		base64.RawStdEncoding.EncodeToString(wert),
	}, "$"), nil
}

// pruefwertPasst vergleicht ein Passwort in gleichbleibender Zeit mit einem Prüfwert.
func pruefwertPasst(schluessel []byte, pruefwert, passwort string) (bool, error) {
	teile := strings.Split(pruefwert, "$")
	if len(teile) != 6 || teile[0] != pruefwertKennung {
		return false, errPruefwertUnlesbar
	}
	speicher, err1 := strconv.ParseUint(teile[1], 10, 32)
	runden, err2 := strconv.ParseUint(teile[2], 10, 32)
	faeden, err3 := strconv.ParseUint(teile[3], 10, 8)
	salz, err4 := base64.RawStdEncoding.DecodeString(teile[4])
	soll, err5 := base64.RawStdEncoding.DecodeString(teile[5])
	if err := errors.Join(err1, err2, err3, err4, err5); err != nil {
		return false, fmt.Errorf("%w: %v", errPruefwertUnlesbar, err)
	}
	if speicher == 0 || runden == 0 || faeden == 0 || len(salz) == 0 || len(soll) == 0 {
		return false, errPruefwertUnlesbar
	}
	ist := argon2.IDKey(pruefwertVorstufe(schluessel, passwort), salz,
		uint32(runden), uint32(speicher), uint8(faeden), uint32(len(soll))) //nolint:gosec // G115: Grenzen stehen in ParseUint
	return subtle.ConstantTimeCompare(ist, soll) == 1, nil
}
