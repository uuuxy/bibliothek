package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"
)

// TokenGueltigkeitTage begrenzt, wie lange ein Bestätigungs-Link lebt.
//
// Ein Link ohne Ablauf bleibt für immer in einem fremden Postfach gültig — auch wenn der
// Mitarbeiter dort längst gewechselt hat oder die Mail weitergeleitet wurde. Der Link IST
// das Geheimnis: Wer ihn hat, bestätigt die Bestellung.
//
// Drei Wochen decken den realen Vorgang ab — der Händler bekommt die Bestellung, druckt
// die Etiketten, beklebt und bestätigt; das dauert Tage, nicht Monate. Vorher standen hier
// 180 Tage: ein halbes Jahr, in dem ein weitergeleiteter Link gültig blieb, ohne dass
// irgendjemand ihn noch auf dem Schirm hatte.
//
// Läuft er doch einmal ab, ist das kein Verlust: Die Bestellhistorie erzeugt zu jeder
// offenen Bestellung einen neuen Link ("Link erzeugen"), und der alte stirbt dabei.
const TokenGueltigkeitTage = 21

// TokenGueltigkeit ist dieselbe Frist als Zeitspanne — für Tests und Anzeige.
const TokenGueltigkeit = TokenGueltigkeitTage * 24 * time.Hour

// tokenBytes: 32 Byte aus crypto/rand = 256 Bit Entropie. Der Link IST das Geheimnis,
// deshalb muss er unerratbar sein — bei dieser Größe scheitert Raten an der Physik,
// nicht am Rate-Limiter. Base64-URL-kodiert ergibt das 43 Zeichen ohne Sonderzeichen,
// die ein Mailprogramm beim Verlinken zerlegen könnte.
const tokenBytes = 32

// neuerBestaetigungsToken liefert den Klartext-Token für die Mail und den Hash für die
// Datenbank. Der Klartext wird NIRGENDS gespeichert: Er existiert nur in dieser einen
// Mail an den Lieferanten — genau wie bei einem Passwort-Zurücksetzen-Link.
func neuerBestaetigungsToken() (token, hash string, err error) {
	roh := make([]byte, tokenBytes)
	if _, err := rand.Read(roh); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(roh)
	return token, hashBestaetigungsToken(token), nil
}

// hashBestaetigungsToken bildet den Token auf seinen Datenbank-Hash ab.
//
// SHA-256 ohne Salt/KDF ist hier richtig und nicht nachlässig: Der Token ist bereits
// 256 Bit Zufall. Wörterbuch- und Brute-Force-Angriffe, gegen die bcrypt & Co. schützen,
// setzen erratbare Eingaben voraus — die gibt es hier nicht. Ein langsamer Hash würde
// nur jeden Seitenaufruf verteuern.
func hashBestaetigungsToken(token string) string {
	summe := sha256.Sum256([]byte(token))
	return hex.EncodeToString(summe[:])
}

// bestaetigungsLink baut die Adresse, die in die Bestellmail geht. Fehlt die öffentliche
// Adresse in den Einstellungen, liefert die Funktion "" — der Aufrufer verschickt dann
// eine Mail ohne Link, statt einen kaputten wie "/bestellung/abc" zu erzeugen.
func bestaetigungsLink(basisAdresse, token string) string {
	basis := strings.TrimSpace(basisAdresse)
	if basis == "" || token == "" {
		return ""
	}
	basis = strings.TrimRight(basis, "/")
	// Ohne Schema baut jedes Mailprogramm einen relativen Link, der beim Lieferanten ins
	// Leere zeigt. Wer in den Einstellungen "bibliothek.schule.de" einträgt, meint https.
	if !strings.HasPrefix(basis, "http://") && !strings.HasPrefix(basis, "https://") {
		basis = "https://" + basis
	}
	return basis + "/bestellung/" + token
}
