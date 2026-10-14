package service

import (
	"regexp"
	"strings"

	"bibliothek/pkg/lmf"
)

// Littera kennzeichnet Schulbücher der Lernmittelfreiheit uneinheitlich: als Vorsilbe der
// Signatur ("LMF Bio 7"), im Standort, Feld 108a ("LMF", "LMF/Bibliothek"), oder in Exporten
// als Token in der Spalte Kategorie ("Buch LMF Ma 6/Gri"). Diese Helfer erkennen das Token;
// daraus wird das Feld buecher_titel.ist_lernmittel, und Titel und Signatur bleiben, wie
// Littera sie liefert.

// lmfTokenMuster trifft "LMF" nur als eigenständiges Token an Wortgrenzen,
// damit Wörter wie "Filmfest" oder Signaturen wie "Elmf" nie anschlagen.
var lmfTokenMuster = regexp.MustCompile(lmf.VermerkMuster)

// hatLMFKennung meldet, ob ein Littera-Feldwert (Signatur, Kategorie oder
// Standort) den Bestand der Lernmittelfreiheit markiert.
func hatLMFKennung(wert string) bool {
	return lmfTokenMuster.MatchString(wert)
}

// entferneLMFToken schneidet das LMF-Token aus einem Feldwert heraus und
// normalisiert übrig bleibende Trenner: "LMF Bio 7" → "Bio 7",
// "Buch LMF Ma 6/Gri" → "Buch Ma 6/Gri", "LMF/Bibliothek" → "Bibliothek".
func entferneLMFToken(wert string) string {
	bereinigt := lmfTokenMuster.ReplaceAllString(wert, "$1")
	bereinigt = strings.Trim(bereinigt, " /")
	return strings.Join(strings.Fields(bereinigt), " ")
}

// zerlegeLMFTeil liest Fach und Jahrgang ab dem LMF-Token eines Feldwerts:
// "Buch LMF Ma 6/Gri" → Mathematik, 6. Ohne Token ok=false.
func zerlegeLMFTeil(wert string) (lmf.Zerlegung, bool) {
	stelle := lmfTokenMuster.FindStringIndex(wert)
	if stelle == nil {
		return lmf.Zerlegung{}, false
	}
	teil := strings.TrimLeft(wert[stelle[0]:], " /")
	return lmf.Zerlege(teil)
}
