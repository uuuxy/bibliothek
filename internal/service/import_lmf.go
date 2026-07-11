package service

import (
	"regexp"
	"strings"
)

// Littera kennzeichnet Lernmittelfreiheits-Bestand (Schulbücher) uneinheitlich:
// mal als Signatur-Präfix ("LMF Bio 7"), mal im Standort-Feld MAB 108a ("LMF",
// "LMF/Bibliothek"), in CSV-Exporten als Token in der Kategorie-Spalte
// ("Buch LMF Ma 6/Gri"). Diese Helfer erkennen das Token, entfernen es aus der
// Signatur (auf dem Rücken-Etikett steht die reine Fach-Signatur) und flaggen
// den Titel per Projekt-Konvention mit dem Präfix "LMF-" — daran erkennen
// Leihfristen (loan_rules), Statistik (stats.go) und die Massenverlängerung
// (ausleihe.go) den Schulbuch-Bestand.

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

// flaggeAlsSchulbuch stellt dem Titel das Projekt-Präfix "LMF-" voran.
// Bereits geflaggte Titel ("LMF-…" oder "LMF …") werden nicht doppelt markiert,
// sondern auf die Bindestrich-Schreibweise vereinheitlicht.
func flaggeAlsSchulbuch(titel string) string {
	lower := strings.ToLower(titel)
	if strings.HasPrefix(lower, "lmf-") {
		return titel
	}
	if strings.HasPrefix(lower, "lmf ") {
		return "LMF-" + strings.TrimSpace(titel[4:])
	}
	return "LMF-" + titel
}
