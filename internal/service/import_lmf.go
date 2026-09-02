package service

import (
	"regexp"
	"strings"

	"bibliothek/pkg/lmf"
)

// Littera kennzeichnet Lernmittelfreiheits-Bestand (Schulbücher) uneinheitlich:
// mal als Signatur-Präfix ("LMF Bio 7"), mal im Standort-Feld MAB 108a ("LMF",
// "LMF/Bibliothek"), in CSV-Exporten als Token in der Kategorie-Spalte
// ("Buch LMF Ma 6/Gri"). Diese Helfer erkennen das Token — daraus wird seit
// Migration 093 das Feld buecher_titel.ist_lernmittel. Bis dahin verschob der Import
// das Wort in den Titel ("LMF-…"), weil nur der Titel gelesen wurde; das ist vorbei,
// Titel und Signatur bleiben, wie Littera sie liefert.

// lmfTokenRegex trifft "LMF" nur als eigenständiges Token an Wortgrenzen,
// damit Wörter wie "Filmfest" oder Signaturen wie "Elmf" nie anschlagen.
var lmfTokenRegex = regexp.MustCompile(`(?i)(^|[\s/])LMF([\s/]|$)`)

// hatLMFKennung meldet, ob ein Littera-Feldwert (Signatur, Kategorie oder
// Standort) den Bestand der Lernmittelfreiheit markiert.
func hatLMFKennung(wert string) bool {
	return lmfTokenRegex.MatchString(wert)
}

// entferneLMFToken schneidet das LMF-Token aus einem Feldwert heraus und
// normalisiert übrig bleibende Trenner: "LMF Bio 7" → "Bio 7",
// "Buch LMF Ma 6/Gri" → "Buch Ma 6/Gri", "LMF/Bibliothek" → "Bibliothek".
func entferneLMFToken(wert string) string {
	bereinigt := lmfTokenRegex.ReplaceAllString(wert, "$1")
	bereinigt = strings.Trim(bereinigt, " /")
	return strings.Join(strings.Fields(bereinigt), " ")
}

// zerlegeLMFTeil liest Fach und Jahrgang ab dem LMF-Token eines Feldwerts:
// "Buch LMF Ma 6/Gri" → Mathematik, 6. Ohne Token ok=false.
func zerlegeLMFTeil(wert string) (lmf.Zerlegung, bool) {
	loc := lmfTokenRegex.FindStringIndex(wert)
	if loc == nil {
		return lmf.Zerlegung{}, false
	}
	teil := strings.TrimLeft(wert[loc[0]:], " /")
	return lmf.Zerlege(teil)
}
