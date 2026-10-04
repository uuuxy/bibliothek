package coverquelle

import (
	"crypto/sha256"
	"encoding/hex"
)

// ersatzbilder sind Bilder, mit denen eine Quelle antwortet, wenn sie zu einer ISBN kein Cover
// hat — mit Status 200 und in gewöhnlicher Größe, sodass weder der Status noch die Maße es
// zeigen. Als Cover abgelegt, stünde das Ersatzbild statt der Initiale des Titels da, und die
// nächste Quelle würde nie gefragt. Erkannt werden sie am Inhalt (SHA-256 der Antwort).
//
// Beide stammen von Google Books (books.google.com/books/content?vid=ISBN:…) und sind dort
// gemessen; TestErsatzbilder_LiveGegenGoogle misst auf Wunsch nach.
var ersatzbilder = map[string]bool{
	// „image not available": graues PNG, 128 × 170, 1.269 Byte.
	"e3f8c414b288cbdf4e6d1e00eb6d3826157d10a5b5628b9318f726ea490eca12": true,
	// Blaugrauer Einband ohne Aufdruck: JPEG, 128 × 184, 10.794 Byte.
	"a9af512c1e52ed9cafd06b8f212bf13940976703ee3a38573df558e28ce31a21": true,
}

func pruefsumme(antwort []byte) string {
	summe := sha256.Sum256(antwort)
	return hex.EncodeToString(summe[:])
}

// IstErsatzbild sagt, ob die Antwort einer Cover-Quelle eines der bekannten Ersatzbilder ist.
func IstErsatzbild(antwort []byte) bool {
	return ersatzbilder[pruefsumme(antwort)]
}

// MerkeErsatzbildFuerTest lässt einen Test ein eigenes Bild als Ersatzbild gelten: Die echten
// stehen als Prüfsumme im Programm, nicht als Datei im Repository. Der Rückgabewert nimmt
// die Angabe zurück; ein Bild, das schon in der Liste stand, bleibt dort.
func MerkeErsatzbildFuerTest(bild []byte) (vergiss func()) {
	summe := pruefsumme(bild)
	if ersatzbilder[summe] {
		return func() { /* Das Bild stand schon in der Liste; es bleibt dort. */ }
	}
	ersatzbilder[summe] = true
	return func() { delete(ersatzbilder, summe) }
}
