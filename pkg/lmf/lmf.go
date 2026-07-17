// Package lmf kapselt die Erkennung von Lernmittel-Titeln (Schulbücher der
// Lernmittelfreiheit) an EINER Stelle. Vorher war die Prüfung an mehreren Stellen als
// strings.HasPrefix(…, "lmf-") bzw. LIKE 'lmf-%' dupliziert. Ein manuell angelegtes
// Buch "LMF - Deutsch 5" (Leerzeichen um den Bindestrich) fiel damit durchs Raster,
// wurde als Freihand-Titel gewertet und liess den Schüler ins Ausleihlimit laufen —
// die Schule konnte keine Schulbücher mehr ausgeben.
package lmf

import (
	"regexp"
	"strings"
)

// prefix matcht "lmf" am Titelanfang, gefolgt von einem Trenner (Leerzeichen oder
// Bindestrich). Deckt "LMF-Deutsch", "LMF - Deutsch" und "LMF Deutsch" ab, aber
// bewusst NICHT "LMFP-Roman" oder "lmfao" — nach dem Kürzel muss ein Trenner stehen.
// Der Bindestrich steht am Ende der Zeichenklasse, damit er literal (keine Range) ist.
var prefix = regexp.MustCompile(`(?i)^lmf[ -]`)

// IstTitel meldet, ob ein Buchtitel ein Lernmittel (Schulbuch) kennzeichnet.
func IstTitel(titel string) bool {
	return prefix.MatchString(strings.TrimSpace(titel))
}

// SQLBedingung liefert ein SQL-Fragment, das für die gegebene Titelspalte prüft, ob es
// ein LMF-Titel ist — robust gegen die Schreibvarianten "lmf-", "lmf -", "lmf ",
// konsistent zu IstTitel. Die Spalte ist entwickler-definiert (nie nutzergesteuert),
// daher ist die Einbettung sicher.
func SQLBedingung(titelSpalte string) string {
	return "LOWER(" + titelSpalte + ") ~ '^lmf[ -]'"
}
