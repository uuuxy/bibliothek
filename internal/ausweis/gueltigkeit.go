// Package ausweis rechnet aus, bis wann ein Schülerausweis gilt.
//
// Die Gültigkeit kommt nicht aus schueler.abgaenger_jahr, weil die zwei Zahlen verschiedene
// Fragen beantworten:
//
//	abgaenger_jahr  Wann verlässt der Schüler die Schule?
//	                Daran hängen die Löschung nach der DSGVO und die Archivierung im
//	                Stapel. Ein Schüler des Gymnasialzweigs bleibt bis 13.
//	Gültigkeit      Wie lange gilt die Karte?
//	                Die Schule stellt Ausweise bis zum Ende der Mittelstufe aus (9 oder
//	                10); für die Oberstufe gibt es einen neuen.
//
// Käme die Gültigkeit aus abgaenger_jahr, hinge die Aufbewahrungsfrist einer Schülerakte
// daran, was auf einer Karte steht.
package ausweis

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SchuljahrEnde liefert das Jahr, in dem das laufende Schuljahr endet (Schuljahr 2026/27:
// 2027).
//
// Die Grenze liegt im August wie im Versetzungslauf und im Import aus Littera. Eine zweite
// Auslegung verschöbe jeden Ausweis um ein Jahr, und zwar nur zwischen August und Dezember.
func SchuljahrEnde(jetzt time.Time) int {
	jahr := jetzt.Year()
	if jetzt.Month() >= time.August {
		jahr++
	}
	return jahr
}

// klassenMuster zerlegt die Bezeichnung einer Klasse der Mittelstufe: Die Ziffern am Anfang
// sind der Jahrgang (eine führende Null ist erlaubt), danach folgen der Buchstabe des Zweigs
// und der Zug.
//
//	"05F1" → 5, "F1"      "5F1" → 5, "F1"
//	"7H1"  → 7, "H1"      "9R1" → 9, "R1"
//
// Der Versetzungslauf in repository und der Import aus Littera lesen die Klasse genauso.
// Liefen die drei auseinander, bekäme derselbe Schüler je nach Weg ein anderes Datum.
var klassenMuster = regexp.MustCompile(`^(\d+)(.*)$`)

// Der Jahrgang, mit dessen Ende ein Ausweis abläuft, je nach Zweig.
const (
	AbschlussHauptschule = 9
	AbschlussMittelstufe = 10
	AbschlussOberstufe   = 13
)

// Die Jahrgänge der Phasen der Oberstufe in Hessen (G9): Die Einführungsphase E ist Jahrgang
// 11, Q1 und Q2 liegen in Jahrgang 12, Q3 und Q4 in Jahrgang 13; das Abitur steht am Ende von
// Q4.
//
// Belegt ist das am Bildungsgang der Schule: Ein Realschulzweig, der mit Jahrgang 10 endet,
// und eine Oberstufe, die mit 13 schließt, ergeben G9. Bei G8 läge die Oberstufe auf 10 bis
// 12.
const (
	jahrgangEinfuehrungsphase = 11
	jahrgangQ12               = 12
	jahrgangQ34               = 13
)

// AblaufJahrgang liefert den Jahrgang, mit dessen Schuljahresende der Ausweis abläuft, und
// den Jahrgang, in dem der Schüler gerade ist.
//
// ok ist false, wenn sich aus der Bezeichnung der Klasse nichts ableiten lässt. Geraten wird
// dann nicht: Ein erfundenes Datum auf einem Ausweis fiele erst auf, wenn die Karte an der
// Ausleihe abgewiesen wird. Der Druckdialog fragt in dem Fall nach.
//
//	5F1, 6G2 (Förderstufe)     → gilt bis Ende Jahrgang 10
//	7H1, 8H2 (Hauptschulzweig) → gilt bis Ende Jahrgang 9
//	7R1, 9R1 (Realschulzweig)  → gilt bis Ende Jahrgang 10
//	7G1, 8G2 (Gymnasialzweig)  → gilt bis Ende Jahrgang 10, nicht 13
//	E1, E2, Q1…Q4 (Oberstufe)  → gilt bis Ende Jahrgang 13
//	ET, 12T, 13T (Oberstufe)   → gilt bis Ende Jahrgang 13, Schreibweise dieser Schule
//
// Der Gymnasialzweig endet hier mit 10, obwohl der Schüler die Schule erst nach 13 verlässt:
// Die Schule stellt Ausweise für die Mittelstufe aus, für die Oberstufe gibt es einen neuen.
// Das Passbild ist so auch nicht sechs Jahre alt.
func AblaufJahrgang(klasse string) (ablauf int, aktuell int, ok bool) {
	k := strings.ToUpper(strings.TrimSpace(klasse))
	if k == "" {
		return 0, 0, false
	}

	// Die Oberstufe zuerst: E1 und Q3 beginnen nicht mit einer Ziffer, und das Klassenmuster
	// verlangt eine.
	if jahrgang, treffer := oberstufenJahrgang(k); treffer {
		return AbschlussOberstufe, jahrgang, true
	}

	teile := klassenMuster.FindStringSubmatch(k)
	if teile == nil {
		return 0, 0, false
	}
	jahrgang, err := strconv.Atoi(teile[1])
	if err != nil || jahrgang < 1 || jahrgang > 13 {
		return 0, 0, false
	}

	abschluss := abschlussAusZweig(teile[2])
	// Liegt der Jahrgang über dem Abschluss des Zweigs, gilt der Jahrgang: „10H1" ist das
	// freiwillige zehnte Jahr im Hauptschulzweig auf dem Weg zum Mittleren Abschluss, kein
	// Datenfehler. Gerechnet wird bis zum Ende des Jahrgangs, in dem der Schüler ist; wer in
	// Jahrgang X ist, beendet mindestens dieses Schuljahr. Ein Datum in der Vergangenheit
	// entsteht so nicht, weil ein Jahrgang außerhalb von 1 bis 13 oben schon abgewiesen ist.
	if jahrgang > abschluss {
		abschluss = jahrgang
	}
	return abschluss, jahrgang, true
}

// oberstufenJahrgang erkennt die Phasen der gymnasialen Oberstufe.
func oberstufenJahrgang(k string) (int, bool) {
	switch {
	case strings.HasPrefix(k, "E"):
		return jahrgangEinfuehrungsphase, true
	case k == "Q1", k == "Q2":
		return jahrgangQ12, true
	case k == "Q3", k == "Q4":
		return jahrgangQ34, true
	case strings.HasPrefix(k, "Q"):
		// Ein unbekannter Zusatz hinter Q: Die Phase steht fest, das Jahr nicht. Angenommen
		// wird der spätere Jahrgang: Ein zu frühes Ablaufdatum macht einen gültigen Ausweis
		// ungültig, ein zu spätes fällt beim nächsten Druck auf.
		return jahrgangQ34, true
	}
	return 0, false
}

// abschlussAusZweig liest den Bildungsgang aus dem ersten Buchstaben hinter dem Jahrgang; was
// danach steht, ist der Zug („H1": Hauptschulzweig, Zug 1).
//
// Ist der Zweig nicht erkennbar, weil kein Buchstabe folgt („10") oder ein unbekannter
// („10a", „7X1"), gilt der längste Weg der Mittelstufe, Jahrgang 10. So ist es auch in der
// Förderstufe, in der der Zweig noch nicht feststeht. Der Fehler geht damit nur in eine
// Richtung: Der Ausweis eines Hauptschülers gilt ein Jahr länger als nötig, und mit dem
// Abgang wird das Konto ohnehin gesperrt. Ein zu früh ablaufender Ausweis würde an der
// Ausleihe abgewiesen, während der Schüler noch zur Schule geht.
//
// Ohne diese Annahme bräuchte fast jeder Ausweis ein Datum von Hand, wo die Klassen keinen
// Buchstaben für den Zweig tragen (5a bis 10d).
func abschlussAusZweig(zweigUndZug string) int {
	rest := strings.TrimSpace(zweigUndZug)
	if rest != "" {
		// []rune, damit ein Umlaut nicht zerschnitten wird.
		switch []rune(rest)[0] {
		case 'H':
			return AbschlussHauptschule
		case 'T':
			// Die Oberstufe dieser Schule heißt ET, 12T und 13T. 12T und 13T beginnen mit
			// einer Ziffer und kommen deshalb nicht über die Erkennung von E und Q, sondern
			// hierher. Als unbekannter Zweig gälte 12T bis Ende Jahrgang 12 und liefe ein
			// Jahr zu früh ab.
			return AbschlussOberstufe
		}
	}
	return AbschlussMittelstufe
}

// GueltigBisJahr liefert das Kalenderjahr, in dem der Ausweis abläuft.
//
// schuljahrEnde ist das Jahr, in dem das laufende Schuljahr endet (Schuljahr 2026/27: 2027).
// Ein Siebtklässler im Hauptschulzweig verlässt die Mittelstufe nach Jahrgang 9; sein Ausweis
// gilt also zwei Schuljahre länger.
func GueltigBisJahr(klasse string, schuljahrEnde int) (int, bool) {
	ablauf, aktuell, ok := AblaufJahrgang(klasse)
	if !ok {
		return 0, false
	}
	return schuljahrEnde + (ablauf - aktuell), true
}
