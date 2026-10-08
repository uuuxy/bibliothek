package inventur

import (
	"bibliothek/pkg/lmf"
	"regexp"
	"strconv"
	"strings"
)

// Die Wörter, die eine Zahl zur Schulstufe machen, und was zwei Zahlen verbindet.
const (
	einzahl  = `(?:klasse|klassenstufe|jahrgangsstufe|schuljahr)`
	mehrzahl = `(?:klassen|klassenstufen|jahrgangsstufen|schuljahre)`
	// Zwei Jahrgänge verbindet ein Zeichen ohne Leerzeichen („7/8", „7–10") oder ein Wort
	// („7 bis 9").
	verbindung = `(?:[/–-]|\s+(?:bis|und|u\.)\s+)`
	// Ein Strich mit Leerzeichen trennt sonst Titelteile („Band 5 - 9. Schuljahr"). Er
	// verbindet nur nach der Mehrzahl, die zwei Zahlen ankündigt („Klassen 5 - 7").
	strich = `\s+[–-]\s+`
	// Woran eine weitere Zahl neben der Angabe hängt, mit oder ohne Leerzeichen.
	nebenZahl = `\s*(?:/|–|-|bis|und|u\.|oder|bzw\.)\s*`
)

var (
	// Die Stufe steht nach dem Wort („Klasse 7", „Klasse 7/8", „Klassen 7–10") oder als
	// Ordnungszahl davor („7. Schuljahr", „5./6. Schuljahr"). Die Mehrzahl gilt nur mit zwei
	// Zahlen: „Klassenstufen 8, 9 und 10" ist eine Aufzählung. Der Text ist kleingeschrieben.
	stufeNachEinzahl   = regexp.MustCompile(`\b` + einzahl + `\s+(\d{1,2})\b(?:\.?` + verbindung + `(\d{1,2})\b)?`)
	stufenNachMehrzahl = regexp.MustCompile(`\b` + mehrzahl + `\s+(\d{1,2})\b\.?(?:` + verbindung + `|` + strich + `)(\d{1,2})\b`)
	stufeVorWort       = regexp.MustCompile(`\b(?:(\d{1,2})\.?` + verbindung + `)?(\d{1,2})\.\s*(?:` + mehrzahl + `|` + einzahl + `)\b`)
	// Eine weitere Zahl neben der Angabe lässt offen, welche Jahrgänge gemeint sind: eine dritte
	// („Klasse 5/6/7"), eine zweite Angabe („Klassen 10-12 oder 11-13") oder der Band („Band 5 -
	// 9. Schuljahr"). Eine Jahreszahl davor ist keine („Ausgabe 2027 - 5. Schuljahr").
	weitereZahlDanach = regexp.MustCompile(`^\.?` + nebenZahl + `\d`)
	weitereZahlDavor  = regexp.MustCompile(`\b\d{1,2}\.?` + nebenZahl + `$`)
	// „ab Klasse 5" und „bis zur 10. Klasse" nennen eine Grenze, keinen einzelnen Jahrgang.
	grenzwortDavor = regexp.MustCompile(`\b(?:ab|bis)\s+(?:der\s+|zur\s+)?$`)
)

// automatischeKategorisierung liest aus Titel und Untertitel eines Buches das Fach und die
// Schulstufe als Spanne „von … bis", als Vorschlag der ISBN-Abfrage für die Maske.
//
// Das Fach kommt aus der einen Stichwortliste in pkg/lmf.
func automatischeKategorisierung(titel, untertitel string) (fach string, stufeVon, stufeBis int) {
	text := strings.ToLower(strings.TrimSpace(titel + " " + untertitel))
	stufeVon, stufeBis = stufenAusText(text)
	return lmf.FachAusText(text), stufeVon, stufeBis
}

// stufenAusText liefert die Schulstufe als Spanne, wenn der Text sie als solche nennt: Ein
// Jahrgang ergibt zweimal dieselbe Zahl („Klasse 7"), zwei verbundene Jahrgänge die Spanne
// („Klassen 7–10", „5./6. Schuljahr"). 0 und 0 heißt, der Text nennt keine; ein Wert allein
// kommt nicht vor. Eine bloße Zahl („Green Line 5"), „Band 2" oder „Level 9" zählt nicht: Sie
// ist oft der Band und nicht die Klasse, und die Maske trägt den Vorschlag in die
// Jahrgangsspanne ein, mit der Inventur und Portal rechnen.
func stufenAusText(text string) (von, bis int) {
	m := frueherer(stufeNachEinzahl.FindStringSubmatchIndex(text), stufenNachMehrzahl.FindStringSubmatchIndex(text))
	if m != nil {
		if weitereZahlDanach.MatchString(text[m[1]:]) || grenzwortDavor.MatchString(text[:m[0]]) {
			return 0, 0
		}
		return spanne(gruppe(text, m, 1), gruppe(text, m, 2))
	}
	if m := stufeVorWort.FindStringSubmatchIndex(text); m != nil {
		davor := text[:m[0]]
		if weitereZahlDavor.MatchString(davor) || grenzwortDavor.MatchString(davor) {
			return 0, 0
		}
		return spanne(gruppe(text, m, 1), gruppe(text, m, 2))
	}
	return 0, 0
}

// frueherer liefert von zwei Treffern den, der im Text zuerst steht; nil heißt kein Treffer.
func frueherer(a, b []int) []int {
	if a == nil || (b != nil && b[0] < a[0]) {
		return b
	}
	return a
}

// gruppe liefert die n-te Gruppe eines Treffers, leer, wenn sie am Treffer nicht beteiligt war.
func gruppe(text string, m []int, n int) string {
	if m[2*n] < 0 {
		return ""
	}
	return text[m[2*n]:m[2*n+1]]
}

// spanne lässt zwei Jahrgänge der Schule in steigender Folge gelten. Fehlt eine der zwei
// Zahlen, nennt der Text einen einzelnen Jahrgang.
func spanne(erste, zweite string) (von, bis int) {
	if erste == "" {
		erste = zweite
	}
	if zweite == "" {
		zweite = erste
	}
	von, bis = schulstufe(erste), schulstufe(zweite)
	if von == 0 || bis == 0 || von > bis {
		return 0, 0
	}
	return von, bis
}

// schulstufe lässt die Jahrgänge der Schule gelten, 5 bis 13 wie der Listenimport
// (parseKlassenStufe); 0 heißt, die Zahl ist keiner.
func schulstufe(zahl string) int {
	n, err := strconv.Atoi(zahl)
	if err != nil || n < 5 || n > 13 {
		return 0
	}
	return n
}
