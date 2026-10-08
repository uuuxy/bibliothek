package repository

import (
	"fmt"
	"time"
)

// AbgangsjahrOhneKlasse ist das Abgangsjahr eines Schülers, dessen Klasse keinen Jahrgang
// hergibt („ABG", „AUS", ein Kursname): das Kalenderjahr plus fünf. Bis zum Abgang steht der
// Wert nur als „Abgang JJJJ" im Profil und lässt sich dort ändern; als Abgänger markieren die
// Versetzung und der LUSD-Import, und beide setzen dabei das tatsächliche Jahr
// (api/student_promotion.go, repository/lusd_import.go).
//
// Verbraucher: die Handanlage und der LUSD-Import (api.abgaengerJahrAm) und die
// Littera-Übernahme für die Gruppe „Im Ausland" — eine Antwort auf dieselbe Frage.
func AbgangsjahrOhneKlasse(jetzt time.Time) int {
	return jetzt.Year() + 5
}

// klassenZahlSQL liest die Zahl am Anfang einer Klasse (NULL ohne führende Ziffer), höchstens
// neun Ziffern weit. Eine längere Folge, etwa eine Buchnummer im Feld Klasse, passt in keine
// ganze Zahl der Datenbank und bräche sonst jede Abfrage ab, die diese Klasse mitliest.
func klassenZahlSQL(spalte string) string {
	return fmt.Sprintf(`(substring(%s from '^\d{1,9}')::int)`, spalte)
}

// AbschlussklasseSQL liefert das SQL-Prädikat „diese Klasse ist eine Abschlussklasse" für
// einen Spaltenausdruck. Es ist die EINE Regel für das Ende eines Bildungsgangs an dieser
// Schule, gelesen aus Jahrgang und Zweig der Klasse — nie aus dem Klassennamen als Text.
// Daran scheiterte der Ursprung der Abgängerliste: `klasse IN ('9h','10r','13')` fand
// „09H1" nicht, und statt die Regel zu heilen, wurde am 25.06./16.07.2026 der Begriff
// getauscht (Register, Entscheidung 2 vom 05.09.2026).
//
//	Hauptschulzweig (H): ab Jahrgang 9  — 9H1, und das freiwillige 10. Hauptschuljahr 10H1
//	Realschulzweig  (R): ab Jahrgang 10 — 10R1
//	alles andere       : ab Jahrgang 13 — „13"; der Gymnasialzweig bleibt über 10 hinaus
//
// Der Zweig ist der ERSTE Buchstabe hinter dem Jahrgang (Klassen-Vokabular, Migration
// 087: „05F1", „09H2"). Eine Klasse ohne führende Ziffer (E1, Q4, ABG) ist nie
// Abschlussklasse — die Schule führt ihre Oberstufe als „13", nicht als Q4.
//
// Verbraucher: die Versetzung (markiert sie als Abgänger, entfernt ihre Klassenleitungs-
// Zuordnung) und die Abgängerliste (zeigt sie in der Saison mit offenen Büchern). Beide
// MÜSSEN dieselbe Menge sehen — api/graduates_pg_test.go (Paar-Gate) hält das fest.
func AbschlussklasseSQL(spalte string) string {
	jahrgang := klassenZahlSQL(spalte)
	// COALESCE ist Pflicht: Ohne Zweigbuchstaben („12", „13") wäre substring NULL, das
	// ganze Prädikat NULL — im WHERE nur „nicht wahr", in der Versetzung aber ein Schreiben
	// von NULL in ist_abgaenger NOT NULL (am 05.09.2026 vom Paar-Gate gefunden).
	zweig := fmt.Sprintf(`coalesce(lower(substring(%s from '^\d+\s*([A-Za-z])')), '')`, spalte)
	return fmt.Sprintf(`(%[1]s ~ '^\d+' AND (
		   (%[2]s >= 9  AND %[3]s = 'h')
		OR (%[2]s >= 10 AND %[3]s = 'r')
		OR  %[2]s >= 13))`, spalte, jahrgang, zweig)
}
