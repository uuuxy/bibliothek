// Package bestelllink bündelt, was den Bestätigungs-Link einer Bestellung ausmacht: das
// Geheimnis im Link, seinen Hash für die Datenbank, die Adresse und die Frist.
//
// Über den Link wählt der Lieferant seine Etiketten, druckt sie und bestätigt die Bestellung,
// ohne Anmeldung. Der Link ist deshalb das Geheimnis: Wer ihn hat, bestätigt. Gespeichert wird
// nur sein Hash; der Klartext steht allein in der Mail an den Lieferanten, wie bei einem Link
// zum Zurücksetzen eines Passworts.
//
// Das Paket steht unter pkg/, weil das Anlegen einer Bestellung und die Türen dasselbe
// brauchen.
package bestelllink

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// VorgabeTage ist die Frist eines Links, solange die Schule unter „Bestellwesen" nichts anderes
// einstellt. Drei Wochen decken den üblichen Vorgang: Der Händler bekommt die Bestellung, druckt
// die Etiketten, beklebt und bestätigt. Ohne Ablauf bliebe der Link in einem fremden Postfach
// gültig, auch nach einer Weiterleitung oder einem Wechsel des Mitarbeiters. Läuft einer ab,
// erzeugt die Bestellhistorie zur offenen Bestellung einen neuen, und der alte gilt nicht mehr.
const VorgabeTage = 21

// Tage liefert die Frist eines neuen Links aus dem eingestellten Wert. Unter einem Tag gilt die
// Vorgabe: Ein Link ohne Frist darf nicht entstehen.
func Tage(eingestellt int) int {
	if eingestellt < 1 {
		return VorgabeTage
	}
	return eingestellt
}

// 32 Byte aus crypto/rand sind 256 Bit Zufall; erraten lässt sich das auch ohne Begrenzung der
// Versuche nicht. In base64url sind es 43 Zeichen ohne Sonderzeichen, die ein Mailprogramm beim
// Verlinken zerlegen könnte.
const tokenBytes = 32

// NeuerToken liefert das Geheimnis für die Mail und seinen Hash für die Datenbank.
func NeuerToken() (token, hash string, err error) {
	roh := make([]byte, tokenBytes)
	if _, err := rand.Read(roh); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(roh)
	return token, Hash(token), nil
}

// Hash bildet ein Geheimnis auf den Wert ab, der in der Datenbank steht. SHA-256 ohne Salz
// genügt: Das Geheimnis ist selbst 256 Bit Zufall, und ein langsamer Hash schützt nur Eingaben,
// die sich erraten lassen. Er verteuerte jeden Aufruf der Seite.
func Hash(token string) string {
	summe := sha256.Sum256([]byte(token))
	return hex.EncodeToString(summe[:])
}

// Adresse baut den Link aus der öffentlichen Adresse der Einstellungen und dem Geheimnis. Fehlt
// eines von beiden, liefert sie "": Der Aufrufer verschickt dann keinen Link statt eines
// kaputten wie "/bestellung/abc".
func Adresse(basisAdresse, token string) string {
	basis := strings.TrimSpace(basisAdresse)
	if basis == "" || token == "" {
		return ""
	}
	basis = strings.TrimRight(basis, "/")
	// Ohne Schema baut ein Mailprogramm einen relativen Link, der beim Lieferanten ins Leere
	// zeigt. Wer in den Einstellungen "bibliothek.schule.de" einträgt, meint https.
	if !strings.HasPrefix(basis, "http://") && !strings.HasPrefix(basis, "https://") {
		basis = "https://" + basis
	}
	return basis + "/bestellung/" + token
}
