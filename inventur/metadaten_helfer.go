package inventur

import (
	"bibliothek/pkg/lmf"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Die Stufe steht nach dem Wort („Klasse 7", „Jahrgangsstufe 10") oder als Ordnungszahl
	// davor („7. Schuljahr"). Der Text ist kleingeschrieben.
	stufeNachWort = regexp.MustCompile(`\b(?:klasse|klassenstufe|jahrgangsstufe|schuljahr)\s+(\d{1,2})\b`)
	stufeVorWort  = regexp.MustCompile(`\b(\d{1,2})\.\s*(?:klasse|klassenstufe|jahrgangsstufe|schuljahr)\b`)
	// Eine zweite Zahl daneben nennt mehrere Jahrgänge („Klasse 7/8", „5./6. Schuljahr").
	zweiteZahlDanach = regexp.MustCompile(`^\s*(?:/|–|-|bis|und|u\.)\s*\d`)
	zweiteZahlDavor  = regexp.MustCompile(`\d\.?\s*(?:/|–|-|bis|und|u\.)\s*$`)
	// „ab Klasse 5" und „bis zur 10. Klasse" nennen eine Grenze, keinen einzelnen Jahrgang.
	grenzwortDavor = regexp.MustCompile(`\b(?:ab|bis)\s+(?:der\s+|zur\s+)?$`)
)

// automatischeKategorisierung liest aus Titel und Untertitel eines Buches das Fach und die
// Schulstufe, als Vorschlag der ISBN-Abfrage für die Maske.
//
// Das Fach kommt aus der einen Stichwortliste in pkg/lmf.
func automatischeKategorisierung(titel, untertitel string) (fach, klassenStufe string) {
	text := strings.ToLower(strings.TrimSpace(titel + " " + untertitel))
	return lmf.FachAusText(text), stufeAusText(text)
}

// stufeAusText liefert die Schulstufe nur, wenn der Text sie als solche nennt und genau einen
// Jahrgang meint. Eine bloße Zahl („Green Line 5"), „Band 2" oder „Level 9" zählt nicht: Sie
// ist oft der Band und nicht die Klasse, und die Maske trägt den Vorschlag in die
// Jahrgangsspanne ein, mit der Inventur und Portal rechnen.
func stufeAusText(text string) string {
	if m := stufeNachWort.FindStringSubmatchIndex(text); m != nil {
		if zweiteZahlDanach.MatchString(text[m[1]:]) || grenzwortDavor.MatchString(text[:m[0]]) {
			return ""
		}
		return schulstufe(text[m[2]:m[3]])
	}
	if m := stufeVorWort.FindStringSubmatchIndex(text); m != nil {
		davor := text[:m[2]]
		if zweiteZahlDavor.MatchString(davor) || grenzwortDavor.MatchString(davor) {
			return ""
		}
		return schulstufe(text[m[2]:m[3]])
	}
	return ""
}

// schulstufe lässt die Jahrgänge der Schule gelten, 5 bis 13 wie der Listenimport
// (parseKlassenStufe).
func schulstufe(zahl string) string {
	n, err := strconv.Atoi(zahl)
	if err != nil || n < 5 || n > 13 {
		return ""
	}
	return strconv.Itoa(n)
}
