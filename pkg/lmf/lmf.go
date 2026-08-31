// Package lmf kapselt die Erkennung von Lernmittel-Titeln (Schulbücher der
// Lernmittelfreiheit) an EINER Stelle. Vorher war die Prüfung an mehreren Stellen als
// strings.HasPrefix(…, "lmf-") bzw. LIKE 'lmf-%' dupliziert. Ein manuell angelegtes
// Buch "LMF - Deutsch 5" (Leerzeichen um den Bindestrich) fiel damit durchs Raster,
// wurde als Freihand-Titel gewertet und liess den Schüler ins Ausleihlimit laufen —
// die Schule konnte keine Schulbücher mehr ausgeben.
//
// Geprüft werden TITEL UND SIGNATUR, nicht nur der Titel (bis 05.08.2026 der einzige
// Blick). Zwei Wege schreiben das Merkmal an verschiedene Stellen: Der Sammelimport
// (internal/service/import_lmf.go) schreibt "LMF-" in den Titel und entfernt das
// Token bewusst aus der Signatur. Die manuelle Neuanlage über die Admin-Oberfläche
// (BuchEingabefelder.svelte, seit 09.07.2026) macht es umgekehrt: Der Auto-Vorschlag
// setzt "LMF <Kürzel>" NUR in die Signatur, der Titel bleibt der Klartext-Buchtitel
// ("Mathematik Neue Wege 9"). Eine Prüfung, die nur den Titel liest, übersieht seither
// jedes von Hand angelegte Schulbuch — mit genau der Folge, die diese Konsolidierung
// ursprünglich beheben sollte: Ausleihlimit greift fälschlich, Schuljahresfrist bleibt
// aus, die Bestellwarnung sieht den Titel nie unter die Meldebestand-Schwelle fallen.
package lmf

import (
	"regexp"
	"strings"
)

// prefix matcht "lmf" am Anfang von Titel oder Signatur, gefolgt von einem Trenner
// (Leerzeichen oder Bindestrich). Deckt "LMF-Deutsch", "LMF - Deutsch" und
// "LMF Deutsch" ab, aber bewusst NICHT "LMFP-Roman" oder "lmfao" — nach dem Kürzel
// muss ein Trenner stehen. Der Bindestrich steht am Ende der Zeichenklasse, damit er
// literal (keine Range) ist.
var prefix = regexp.MustCompile(`(?i)^lmf[ -]`)

// IstSchulbuch meldet, ob Titel ODER Signatur eines Buchs es als Lernmittel
// kennzeichnet.
func IstSchulbuch(titel, signatur string) bool {
	return prefix.MatchString(strings.TrimSpace(titel)) || prefix.MatchString(strings.TrimSpace(signatur))
}

// SQLBedingung liefert ein SQL-Fragment, das für die gegebenen Titel- und
// Signaturspalten prüft, ob eine der beiden ein LMF-Kennzeichen trägt — robust gegen
// die Schreibvarianten "lmf-", "lmf -", "lmf ", konsistent zu IstSchulbuch. Die
// Spalten sind entwickler-definiert (nie nutzergesteuert), daher ist die Einbettung
// sicher.
func SQLBedingung(titelSpalte, signaturSpalte string) string {
	// btrim wie strings.TrimSpace in IstSchulbuch (31.08.2026): Ohne das Trimmen war ein
	// Titel mit führendem Leerzeichen (Import, Copy-Paste aus Excel) für Go ein
	// Lernmittel — Ausleihlimit, Schuljahresfrist — und für das SQL keins. Genau dieses
	// Buch erschien dann im öffentlichen Katalog und als „Buch des Monats" auf dem
	// Flurbildschirm, obwohl repository.OeffentlichSichtbar es aussortieren soll.
	// btrim entfernt Leerzeichen, Tabs, Zeilenumbrüche und Wagenrückläufe.
	const nackt = ", E' \\t\\n\\r'"
	return "(LOWER(btrim(" + titelSpalte + nackt + ")) ~ '^lmf[ -]' OR LOWER(btrim(COALESCE(" +
		signaturSpalte + ", '')" + nackt + ")) ~ '^lmf[ -]')"
}
