// Package ausweis rechnet aus, bis wann ein Schülerausweis gilt.
//
// Warum das NICHT schueler.abgaenger_jahr benutzt, obwohl der Ausweis bisher genau das
// anzeigte: Die beiden Zahlen beantworten verschiedene Fragen.
//
//	abgaenger_jahr  Wann verlässt der Schüler die SCHULE?
//	                Steuert die DSGVO-Löschung (jobs/cron.go, PurgeAbgaenger) und die
//	                Stapel-Archivierung. Ein Gymnasialschüler bleibt bis 13.
//	Gültigkeit      Wie lange gilt die KARTE?
//	                Die Schule stellt Ausweise bis zum Ende der Mittelstufe aus
//	                (9 bzw. 10); für die Oberstufe gibt es einen neuen.
//
// Würde man die Ausweisgültigkeit aus abgaenger_jahr ableiten, hinge die
// Aufbewahrungsfrist einer Schülerakte daran, was auf eine Plastikkarte gedruckt wird.
// Zwei Zahlen, zwei Funktionen, kein gemeinsamer Schalter.
package ausweis

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SchuljahrEnde liefert das Jahr, in dem das zum Zeitpunkt jetzt laufende Schuljahr
// endet (Schuljahr 2026/27 → 2027).
//
// Die Grenze liegt im August — dieselbe Auslegung wie im Versetzungslauf und im
// Littera-Import (internal/littera/schreiber.go, StandardOptionen). Eine zweite
// Auslegung wäre hier fatal: Sie verschöbe jeden Ausweis um genau ein Jahr, und zwar
// nur zwischen August und Dezember.
func SchuljahrEnde(jetzt time.Time) int {
	jahr := jetzt.Year()
	if jetzt.Month() >= time.August {
		jahr++
	}
	return jahr
}

// klassenMuster zerlegt eine Klassenbezeichnung der Mittelstufe: führende Ziffern =
// Jahrgang (führende Null erlaubt), danach Zweigbuchstabe und Zug.
//
//	"05F1" → 5, "F1"      "5F1" → 5, "F1"
//	"7H1"  → 7, "H1"      "9R1" → 9, "R1"
//
// Dieselbe Auslegung wie im Versetzungslauf (api/student_promotion.go:
// substring(klasse from '^\d+')) und im Littera-Import (internal/littera/abgang.go).
// Drei Stellen, eine Konvention — liefen sie auseinander, bekäme derselbe Schüler je
// nach Weg ein anderes Datum.
var klassenMuster = regexp.MustCompile(`^(\d+)(.*)$`)

// Abschlussjahrgang nennt den Jahrgang, mit dessen Ende der Ausweis abläuft.
const (
	AbschlussHauptschule = 9
	AbschlussMittelstufe = 10
	AbschlussOberstufe   = 13
)

// Jahrgang der Oberstufenphasen in Hessen (G9, Stand August 2026):
// Einführungsphase E ist Jahrgang 11, die Qualifikationsphase umfasst 12 und 13 —
// Q1/Q2 liegen in Jahrgang 12, Q3/Q4 in Jahrgang 13. Das Abitur steht am Ende von Q4.
//
// Belegt am Bildungsgang der Schule selbst: Ein Realschulzweig, der mit Jahrgang 10
// endet, und eine Oberstufe, die mit 13 schließt, ergeben zusammen G9. Bei G8 läge die
// Oberstufe auf 10–12.
const (
	jahrgangEinfuehrungsphase = 11
	jahrgangQ12               = 12
	jahrgangQ34               = 13
)

// AblaufJahrgang liefert den Jahrgang, mit dessen Schuljahresende der Ausweis abläuft,
// sowie den Jahrgang, in dem der Schüler gerade steckt.
//
// ok=false bedeutet: Aus dieser Klassenbezeichnung lässt sich nichts ableiten. Dann
// darf NICHTS geraten werden — ein erfundenes Datum auf einem Ausweis fällt niemandem
// auf, bis die Karte an der Ausleihe abgewiesen wird. Der Druckdialog fragt in dem Fall
// nach.
//
//	5F1, 6G2 (Förderstufe)     → gilt bis Ende Jahrgang 10
//	7H1, 8H2 (Hauptschulzweig) → gilt bis Ende Jahrgang 9
//	7R1, 9R1 (Realschulzweig)  → gilt bis Ende Jahrgang 10
//	7G1, 8G2 (Gymnasialzweig)  → gilt bis Ende Jahrgang 10  ← Mittelstufe, NICHT 13
//	E1, E2, Q1…Q4 (Oberstufe)  → gilt bis Ende Jahrgang 13
//
// Der Gymnasialzweig endet hier bewusst mit 10 und nicht mit 13, obwohl der Schüler die
// Schule erst nach 13 verlässt: Die Schule stellt Ausweise für die Mittelstufe aus, für
// die Oberstufe gibt es einen neuen. Nebeneffekt, der dafür spricht: Das Passbild ist
// dann nicht sechs Jahre alt.
func AblaufJahrgang(klasse string) (ablauf int, aktuell int, ok bool) {
	k := strings.ToUpper(strings.TrimSpace(klasse))
	if k == "" {
		return 0, 0, false
	}

	// Oberstufe zuerst: E1/Q3 tragen keine führende Ziffer und fallen sonst durch das
	// Klassenmuster, weil das Ziffern am Anfang verlangt.
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

	abschluss, treffer := abschlussAusZweig(teile[2])
	if !treffer {
		return 0, 0, false
	}
	// Liegt der Jahrgang ÜBER dem Regelabschluss des Zweigs, gilt der Jahrgang.
	//
	// Der Fall ist nicht theoretisch: "10H1" ist das freiwillige 10. Hauptschuljahr auf
	// dem Weg zum Mittleren Abschluss. Der Hauptschulzweig endet regulär mit 9, ein
	// Zehntklässler darin ist also kein Datenfehler, sondern ein regulärer Bildungsweg.
	//
	// Vorher wurde hier abgelehnt, und der Ausweis eines 10H-Schülers hatte KEIN Datum —
	// ausgerechnet in dem Jahr, in dem er einen neuen braucht. Gerechnet wird jetzt bis
	// zum Ende des Jahrgangs, in dem er tatsächlich steckt. Das ist die Aussage, die
	// immer stimmt: Wer in Jahrgang X ist, beendet mindestens dieses Schuljahr.
	//
	// Nach unten bleibt es abgesichert — ein Jahrgang außerhalb 1..13 ist oben schon
	// aussortiert, ein Datum in der Vergangenheit kann so nicht entstehen.
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
		// Unbekannter Q-Zusatz: Die Phase steht fest, das Jahr nicht. Der spätere
		// Jahrgang ist die sichere Annahme — ein zu frühes Ablaufdatum macht einen
		// gültigen Ausweis ungültig, ein zu spätes fällt beim nächsten Druck auf.
		return jahrgangQ34, true
	}
	return 0, false
}

// abschlussAusZweig liest den Bildungsgang aus dem ersten Buchstaben hinter dem
// Jahrgang. Was dahinter steht, ist der Zug ("H1" → Hauptschulzweig, Zug 1).
func abschlussAusZweig(zweigUndZug string) (int, bool) {
	rest := strings.TrimSpace(zweigUndZug)
	if rest == "" {
		return 0, false
	}
	// []rune, damit ein Umlaut nicht zerschnitten wird.
	switch []rune(rest)[0] {
	case 'H':
		return AbschlussHauptschule, true
	case 'F', 'R', 'G':
		// Förderstufe (5/6) steht noch vor der Zweigwahl — angesetzt wird der längste
		// Weg der Mittelstufe, sonst läuft der Ausweis eines künftigen Realschülers zu
		// früh ab.
		return AbschlussMittelstufe, true
	}
	return 0, false
}

// GueltigBisJahr rechnet das Kalenderjahr aus, in dem der Ausweis abläuft.
//
// schuljahrEnde ist das Jahr, in dem das AKTUELLE Schuljahr endet (Schuljahr 2026/27 →
// 2027). Ein Siebtklässler im Hauptschulzweig verlässt die Mittelstufe nach Jahrgang 9,
// sein Ausweis gilt also zwei Schuljahre länger.
func GueltigBisJahr(klasse string, schuljahrEnde int) (int, bool) {
	ablauf, aktuell, ok := AblaufJahrgang(klasse)
	if !ok {
		return 0, false
	}
	return schuljahrEnde + (ablauf - aktuell), true
}
