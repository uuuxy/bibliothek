package repository

// Der Stichtag ist der letzte Rückgabetag der Schulbücher eines Schuljahres: eine Rechnung für
// die Ausleihe, den Büchertausch und den Rückweg des Plans, sonst bekäme dasselbe Schulbuch je
// nach Weg eine andere Frist.
//
// Bis Jahrgang 10 ist es der Donnerstag vor den Sommerferien, an dem der Büchertausch und der
// Rückgabe-Plan enden (lmfplan.DonnerstagVor). Die Oberstufe organisiert Rückgabe und Ausgabe an
// dieser Schule selbst und behält den 31. Juli, das Ende des Schuljahres; ebenso eine Klasse ohne
// Jahrgang und ein Schuljahr, dessen Sommerferien nicht bekannt sind. Beide Tage liegen im
// Schuljahr, denn an der Frist lesen Bestellbedarf, Theke und Plan-Kopplung das Schuljahr ab.
//
// Zwei Fragen gehen an dieselbe Rechnung:
//
//   - LMFStichtagImSchuljahr: „Welcher Tag ist der Stichtag dieses Schuljahres?" Das fragt der
//     Rückweg: Verliert eine Klasse ihren Termin, gehen ihre Fristen an den Stichtag des
//     Schuljahres zurück, in dem der Termin lag.
//   - LMFStichtagAbTag: „Welcher Stichtag liegt noch vor mir?" Das fragt die Ausleihe: Eine Frist
//     in der Vergangenheit wäre eine Mahnung am Tag der Ausgabe.

import (
	"fmt"
	"time"

	"bibliothek/internal/pkg/lmfplan"
	"bibliothek/internal/pkg/schulzeit"
)

// lmfLetzterJahrgangMittelstufe ist der letzte Jahrgang vor der Oberstufe, wie im Rückgabe-Plan
// (LadeKlassenMitSchuelern).
const lmfLetzterJahrgangMittelstufe = 10

// LMFStichtagNachJahren liefert den Stichtag des Schuljahres, das jahre Schuljahre nach dem des
// Tages liegt, für ein Kind, das heute in der Klasse klasse ist; sein Jahrgang rückt jedes
// Schuljahr um eins vor. Die Uhrzeit ist 12:00 der Schulzeitzone, damit der Kalendertag bei keiner
// Zeitumstellung kippt; zur Frist wird er erst über schulzeit.TagesEnde.
func LMFStichtagNachJahren(tag time.Time, klasse string, jahre int, ferien lmfplan.Ferientabelle) time.Time {
	ende := SchuljahrBeginn(tag).Year() + 1 + jahre
	return lmfStichtagAmEnde(ende, lmfMittelstufe(klasse, jahre), ferien)
}

// LMFStichtagImSchuljahr liefert den Stichtag des Schuljahres, in dem der Tag liegt.
func LMFStichtagImSchuljahr(tag time.Time, klasse string, ferien lmfplan.Ferientabelle) time.Time {
	return LMFStichtagNachJahren(tag, klasse, 0, ferien)
}

// LMFStichtagAbTag liefert den nächsten Stichtag, der nicht vor dem Tag liegt, für ein Schulbuch,
// das zusatzJahre Schuljahre länger beim Kind bleibt (Band über mehrere Jahrgänge). Ist der
// Stichtag des laufenden Schuljahres vorbei, gilt das Buch dem nächsten; die Jahre eines Bands
// zählen schon ab dem laufenden, deshalb das größere von beiden. Verglichen werden Kalendertage:
// Wer am Stichtag ausleiht, bekommt ihn noch (bis 23:59:59).
func LMFStichtagAbTag(tag time.Time, klasse string, zusatzJahre int, ferien lmfplan.Ferientabelle) time.Time {
	tag = tag.In(schulzeit.Zone())
	heute := time.Date(tag.Year(), tag.Month(), tag.Day(), 0, 0, 0, 0, schulzeit.Zone())
	vorbei := 0
	if LMFStichtagImSchuljahr(tag, klasse, ferien).Before(heute) {
		vorbei = 1
	}
	return LMFStichtagNachJahren(tag, klasse, max(zusatzJahre, vorbei), ferien)
}

// lmfMittelstufe sagt, ob ein Kind der Klasse klasse in jahre Schuljahren höchstens im Jahrgang 10
// ist. lmfMittelstufeSQL ist der Zwilling für den Büchertausch, den die Datenbank schreibt.
func lmfMittelstufe(klasse string, jahre int) bool {
	jahrgang, _, ok := AbschlussJahrgang(klasse)
	return ok && jahrgang+jahre <= lmfLetzterJahrgangMittelstufe
}

// lmfMittelstufeSQL ist lmfMittelstufe für eine Spalte der Klasse. JahrgangSQL ist der Zwilling
// von AbschlussJahrgang (internal/repository/abgaenger_jahr_paar_pg_test.go).
func lmfMittelstufeSQL(spalte string, jahre int) string {
	return fmt.Sprintf(`COALESCE(%s + %d <= %d, false)`, JahrgangSQL(spalte), jahre, lmfLetzterJahrgangMittelstufe)
}

// lmfStichtagAmEnde liefert den Stichtag des Schuljahres, das im Sommer des Jahres ende endet.
func lmfStichtagAmEnde(ende int, mittelstufe bool, ferien lmfplan.Ferientabelle) time.Time {
	zone := schulzeit.Zone()
	juli := time.Date(ende, time.July, 31, 12, 0, 0, 0, zone)
	if !mittelstufe {
		return juli
	}
	z, ok := ferien.Sommerferien(ende)
	if !ok {
		return juli
	}
	d := lmfplan.DonnerstagVor(z.Von)
	donnerstag := time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, zone)
	// Eingetragene Ferien, die erst im August beginnen, legten den Donnerstag ins nächste Schuljahr.
	if donnerstag.After(juli) {
		return juli
	}
	return donnerstag
}
