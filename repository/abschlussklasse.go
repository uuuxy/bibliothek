package repository

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bibliothek/pkg/schulzeit"
)

// AbgangsjahrOhneKlasse ist das Abgangsjahr eines Schülers, dessen Klasse keinen Jahrgang
// hergibt („ABG", „AUS", ein Kursname): das Kalenderjahr plus fünf. Bis zum Abgang steht der
// Wert nur als „Abgang JJJJ" im Profil und lässt sich dort ändern; als Abgänger markieren die
// Versetzung und der LUSD-Import, und beide setzen dabei das tatsächliche Jahr
// (schuljahreswechsel.go, lusd_import.go).
//
// Verbraucher: die Handanlage und der LUSD-Import (AbgaengerJahrAm) und die
// Littera-Übernahme für die Gruppe „Im Ausland" — eine Antwort auf dieselbe Frage.
func AbgangsjahrOhneKlasse(jetzt time.Time) int {
	return jetzt.In(schulzeit.Zone()).Year() + 5
}

// klassenZahlSQL liest die Zahl am Anfang einer Klasse (NULL ohne führende Ziffer), höchstens
// neun Ziffern weit. Eine längere Folge, etwa eine Buchnummer im Feld Klasse, passt in keine
// ganze Zahl der Datenbank und bräche sonst jede Abfrage ab, die diese Klasse mitliest.
func klassenZahlSQL(spalte string) string {
	return fmt.Sprintf(`(substring(%s from '^\d{1,9}')::int)`, spalte)
}

// jahrgangImBereichSQL ist das Prädikat „diese Klasse beginnt mit einem Jahrgang von 1 bis 13“,
// führende Nullen erlaubt („05F1“, „13“). Was mit einer anderen Zahl beginnt („14a“, eine
// Buchnummer im Feld Klasse), ist kein Jahrgang der Schule: Die Versetzung lässt es stehen und
// nennt es, keine Abfrage liest daraus eine Abschlussklasse.
func jahrgangImBereichSQL(spalte string) string {
	return fmt.Sprintf(`(%s ~ '^0*([1-9]|1[0-3])(\D|$)')`, spalte)
}

// leerraumKlasseSQL sind die 25 Zeichen mit der Unicode-Eigenschaft White_Space für eine
// Klasse in einem regulären Ausdruck von PostgreSQL, dieselben wie in titeltext_normalform
// (Migration 160). Go zählt sie alle zum Leerraum (unicode.IsSpace).
const leerraumKlasseSQL = `\u0009-\u000D\u0020\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000`

// AbschlussklasseSQL liefert das SQL-Prädikat „diese Klasse ist eine Abschlussklasse" für einen
// Spaltenausdruck. Es ist die eine Regel für das Ende eines Bildungsgangs an dieser Schule,
// gelesen aus Jahrgang und Zweig der Klasse, nie aus dem Klassennamen als Text: `klasse IN
// ('9h','10r','13')` fände „09H1" nicht.
//
//	Hauptschulzweig (H): ab Jahrgang 9  — 9H1, und das freiwillige 10. Hauptschuljahr 10H1
//	Realschulzweig  (R): ab Jahrgang 10 — 10R1
//	alles andere       : ab Jahrgang 13 — „13"; der Gymnasialzweig bleibt über 10 hinaus
//
// Der Zweig ist der erste Buchstabe hinter dem Jahrgang (Klassen-Vokabular: „05F1", „09H2").
// Eine Klasse ohne führende Ziffer (E1, Q4, ABG) ist nie Abschlussklasse: Die Schule führt ihre
// Oberstufe als „13", nicht als Q4. Ebenso eine, die mit keinem Jahrgang von 1 bis 13 beginnt
// (jahrgangImBereichSQL).
//
// Verbraucher: die Versetzung (markiert sie als Abgänger, entfernt ihre Zuordnung zur
// Klassenleitung) und die Abgängerliste (zeigt sie in der Saison mit offenen Büchern). Beide
// müssen dieselbe Menge sehen; api/abgaenger_versetzung_paar_pg_test.go hält das fest.
func AbschlussklasseSQL(spalte string) string {
	jahrgang := klassenZahlSQL(spalte)
	// COALESCE ist Pflicht: Ohne Zweigbuchstaben („12", „13") wäre substring NULL, das ganze
	// Prädikat NULL — im WHERE nur „nicht wahr", in der Versetzung aber ein Schreiben von NULL
	// in ist_abgaenger NOT NULL.
	//
	// Zwischen Jahrgang und Zweig gilt jeder Leerraum, den AbschlussJahrgang wegnimmt; `\s` träfe in
	// PostgreSQL das geschützte Leerzeichen nicht.
	zweig := fmt.Sprintf(`coalesce(lower(substring(%s from '^\d+[`+leerraumKlasseSQL+`]*([A-Za-z])')), '')`, spalte)
	return fmt.Sprintf(`(%[1]s AND (
		   (%[2]s >= 9  AND %[3]s = 'h')
		OR (%[2]s >= 10 AND %[3]s = 'r')
		OR  %[2]s >= 13))`, jahrgangImBereichSQL(spalte), jahrgang, zweig)
}

// AbschlussJahrgang liest aus einer Klasse ihren Jahrgang und den Jahrgang, mit dem ihr
// Bildungsgang endet: H endet mit 9, R mit 10, alles andere mit 13. „E" ohne Ziffer ist die
// Einführungsphase, Jahrgang 11. Es ist der Go-Zwilling von AbschlussklasseSQL; das Paar-Gate
// repository/abgaenger_jahr_paar_pg_test.go vergleicht beide über den ganzen Formenraum.
//
// ok=false heißt: kein Jahrgang von 1 bis 13 (ABG, Q4, leer, 14a). Der Aufrufer nimmt dann
// einen Rückfallwert.
func AbschlussJahrgang(klasse string) (jahrgang, abschluss int, ok bool) {
	klasse = strings.ToLower(strings.TrimSpace(klasse))
	if strings.HasPrefix(klasse, "e") {
		return 11, 13, true // Einführungsphase: ET, E1, E2 — Jahrgang 11
	}

	ziffern := ""
	zweig := ""
	for i, c := range klasse {
		if c >= '0' && c <= '9' {
			ziffern += string(c)
		} else {
			zweig = strings.TrimSpace(klasse[i:])
			break
		}
	}
	// Jahrgänge 1 bis 13 wie jahrgangImBereichSQL; „14a“ oder eine Buchnummer ist keiner.
	zahl, err := strconv.Atoi(ziffern)
	if err != nil || zahl < 1 || zahl > 13 {
		return 0, 0, false
	}

	switch {
	case strings.HasPrefix(zweig, "h"):
		return zahl, 9, true
	case strings.HasPrefix(zweig, "r"):
		return zahl, 10, true
	default:
		return zahl, 13, true
	}
}

// AbgaengerJahr errechnet das voraussichtliche Abgangsjahr eines Schülers aus seiner Klasse
// und dem laufenden Schuljahr. Die Anlage von Hand, das Ändern der Klasse, das Zusammenführen
// und der LUSD-Import fragen hier: eine Antwort auf dieselbe Frage.
func AbgaengerJahr(klasse string) int {
	return AbgaengerJahrAm(klasse, time.Now())
}

// AbgaengerJahrAm rechnet wie AbgaengerJahr zu einem genannten Zeitpunkt. Das Schuljahr endet
// im Juli; ab August zählt das folgende Kalenderjahr als Basis.
func AbgaengerJahrAm(klasse string, jetzt time.Time) int {
	jetzt = jetzt.In(schulzeit.Zone())
	jahrgang, abschluss, ok := AbschlussJahrgang(klasse)
	if !ok {
		return AbgangsjahrOhneKlasse(jetzt)
	}
	yearsLeft := abschluss - jahrgang
	if yearsLeft < 0 {
		yearsLeft = 0
	}
	baseYear := jetzt.Year()
	if jetzt.Month() >= time.August {
		baseYear++
	}
	return baseYear + yearsLeft
}
