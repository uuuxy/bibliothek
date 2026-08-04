package littera

import (
	"regexp"
	"strconv"
	"strings"
)

// klassenMuster zerlegt eine Littera-Klassenbezeichnung: führende Ziffern = Jahrgang,
// Rest = Zweig und Zug ("07H1" → 7, "H1"; "12T3" → 12, "T3"). Dieselbe Auslegung wie im
// Versetzungslauf (api/student_promotion.go: substring(klasse from '^\d+')).
var klassenMuster = regexp.MustCompile(`^(\d+)(.*)$`)

// AbschlussJahrgang liefert den Jahrgang, nach dem ein Zweig die Schule verlässt.
//
// Die Regel ist NICHT hier erfunden: Der Versetzungslauf markiert seit jeher genau
// 9H, 10R und 13 als Abgänger (api/student_promotion.go, is_graduating). Diese Funktion
// leitet daraus dieselbe Aussage für den Import ab — zwei Wege, dieselbe Konvention.
// Liefen sie auseinander, hätte ein importierter Schüler ein anderes Abgangsjahr als
// derselbe Schüler nach dem ersten Schuljahreswechsel.
//
//	H  Hauptschulzweig    → 9
//	R  Realschulzweig     → 10
//	G  Gymnasialzweig     → 13
//	T  Sekundarstufe II   → 13
//	F  Förderstufe        → 13 (siehe unten)
//
// Die Förderstufe (Jahrgang 5–6) ist ein Sonderfall: Von dort geht es erst danach in
// einen der drei Zweige, das Abgangsjahr steht also noch gar nicht fest. Angesetzt wird
// bewusst der LÄNGSTE Weg. Das Abgangsjahr steuert die Stapel-Archivierung; ein zu
// früher Wert würde einen Schüler aus dem Bestand nehmen, der noch zur Schule geht.
// Ein zu später Wert kostet nichts — der Versetzungslauf zieht ihn jedes Jahr nach.
func AbschlussJahrgang(zweigUndZug string) (int, bool) {
	rest := strings.ToUpper(strings.TrimSpace(zweigUndZug))
	if rest == "" {
		return 0, false
	}
	// Nur der ERSTE Buchstabe ist der Zweig; was dahinter steht, ist der Zug
	// ("H1" → Hauptschulzweig, Zug 1). []rune, damit ein Umlaut nicht zerschnitten wird.
	switch []rune(rest)[0] {
	case 'H':
		return 9, true
	case 'R':
		return 10, true
	case 'G', 'T', 'F':
		return 13, true
	}
	return 0, false
}

// AbgaengerJahr rechnet aus, in welchem Kalenderjahr ein Schüler die Schule verlässt.
//
// schuljahrEnde ist das Jahr, in dem das AKTUELLE Schuljahr endet (Schuljahr 2026/27 →
// 2027). Ein Siebtklässler im Hauptschulzweig verlässt die Schule nach Jahrgang 9, also
// zwei Schuljahre später.
//
// Zweiter Rückgabewert ist false, wenn die Klassenbezeichnung nichts hergibt (kein
// Jahrgang, unbekannter Zweig). Dann darf der Import KEINEN Wert erfinden:
// schueler.abgaenger_jahr ist NOT NULL, aber eine geratene Jahreszahl archiviert
// irgendwann still den falschen Schüler.
func AbgaengerJahr(klasse string, schuljahrEnde int) (int, bool) {
	treffer := klassenMuster.FindStringSubmatch(strings.TrimSpace(klasse))
	if treffer == nil {
		return 0, false
	}
	jahrgang, err := strconv.Atoi(treffer[1])
	if err != nil || jahrgang <= 0 {
		return 0, false
	}
	abschluss, ok := AbschlussJahrgang(treffer[2])
	if !ok {
		return 0, false
	}

	verbleibend := abschluss - jahrgang
	if verbleibend < 0 {
		// Jemand steht über seinem Abschlussjahrgang (Wiederholer in 10H, Datenfehler).
		// Dann ist er am Ende dieses Schuljahres dran, nicht in der Vergangenheit.
		verbleibend = 0
	}
	return schuljahrEnde + verbleibend, true
}

// IstAbschlussklasse sagt, ob eine Klasse am Ende dieses Schuljahres abgeht —
// dieselbe Aussage, die der Versetzungslauf über is_graduating trifft.
func IstAbschlussklasse(klasse string) bool {
	treffer := klassenMuster.FindStringSubmatch(strings.TrimSpace(klasse))
	if treffer == nil {
		return false
	}
	jahrgang, err := strconv.Atoi(treffer[1])
	if err != nil {
		return false
	}
	abschluss, ok := AbschlussJahrgang(treffer[2])
	return ok && jahrgang >= abschluss
}
