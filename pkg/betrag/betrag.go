// Package betrag schreibt Geldbeträge in der deutschen Form: zwei Nachkommastellen mit Komma.
//
// Eine Stelle für Papier und Bildschirm: Briefe, Berichte und die Meldungen der Theke schreiben
// einen Betrag gleich.
package betrag

import (
	"fmt"
	"math"
	"strings"
)

// Text ist der Betrag ohne Währung: „12,50". Gerundet wird kaufmännisch auf den Cent.
func Text(betrag float64) string {
	cent := math.Round(betrag * 100)
	if cent == 0 {
		// Sonst stünde bei einem Betrag knapp unter null „-0,00" da.
		cent = 0
	}
	return strings.Replace(fmt.Sprintf("%.2f", cent/100), ".", ",", 1)
}

// Euro ist der Betrag mit Euro-Zeichen: „12,50 €".
func Euro(betrag float64) string {
	return Text(betrag) + " €"
}
