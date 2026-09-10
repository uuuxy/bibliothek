// Package ersatzwert rechnet den Schadensersatz-VORSCHLAG für ein Lernmittel nach der
// Staffel, die die Arbeitshilfe der Schule empfiehlt.
//
// Vorschlag, kein Automat: Der Betrag liegt im Ermessen der Schule („je nach Zustand des
// Lehrwerks, Ausleihhäufigkeit etc."). Diese Funktion liefert die Herleitung mit, damit
// im Dialog steht, WARUM 60 % von 41,50 € vorgeschlagen werden — und der Mensch
// überschreiben kann.
//
// Die Staffel:
//
//  1. Verleihjahr   voller Preis zum Zeitpunkt des Kaufs
//  2. Verleihjahr   80 % des Neupreises zum Zeitpunkt des Verlusts
//  3. Verleihjahr   60 %
//  4. Verleihjahr   40 %
//  5. Verleihjahr   20 %
//     ab dem 6.        10 %
//
// Die BASIS wechselt zwischen dem ersten und dem zweiten Jahr: erst der Kaufpreis, dann
// der heutige Neupreis. Das ist keine Ungenauigkeit der Vorlage, sondern ihr Sinn — im
// ersten Jahr ist das Buch neu, danach zählt, was ein Ersatz heute kostet.
package ersatzwert

import "math"

// Prozentstaffel: Index 0 = 1. Verleihjahr. Ab dem sechsten Jahr gilt der letzte Wert.
var prozentstaffel = []int{100, 80, 60, 40, 20, 10}

// Basis benennt, auf welchen Preis sich der Prozentsatz bezieht.
type Basis string

const (
	// BasisKaufpreis ist der Preis, zu dem die Schule das Buch beschafft hat (1. Verleihjahr).
	BasisKaufpreis Basis = "kaufpreis"
	// BasisNeupreis ist, was ein Ersatz heute kostet (ab dem 2. Verleihjahr).
	BasisNeupreis Basis = "neupreis"
	// BasisKaufpreisErsatzweise heißt: Ab dem 2. Verleihjahr wäre der Neupreis maßgeblich, es ist
	// aber keiner hinterlegt — gerechnet wird mit dem Kaufpreis. Der Dialog sagt das, damit
	// niemand eine Genauigkeit annimmt, die die Zahl nicht hat.
	BasisKaufpreisErsatzweise Basis = "kaufpreis_ersatzweise"
)

// Vorschlag ist das Ergebnis: der Betrag, der Prozentsatz, die Basis und der Preis, auf
// den sich der Prozentsatz bezieht.
type Vorschlag struct {
	Betrag      float64
	Prozent     int
	Basis       Basis
	BasisPreis  float64
	Verleihjahr int
}

// Rechne liefert den Vorschlag für ein Exemplar.
//
// verleihjahr wird bei 1 abgeschnitten: Ein Buch, das gerade neu im Regal steht, ist im
// ersten Verleihjahr — eine 0 aus einer unvollständigen Historie darf nicht zu 0 € führen.
// Fehlen beide Preise, ist der Betrag 0 und der Mensch trägt ihn selbst ein; ein geratener
// Betrag wäre in einem Bescheid schlimmer als ein leeres Feld.
func Rechne(verleihjahr int, kaufpreis, neupreis float64) Vorschlag {
	if verleihjahr < 1 {
		verleihjahr = 1
	}
	prozent := prozentstaffel[len(prozentstaffel)-1]
	if verleihjahr <= len(prozentstaffel) {
		prozent = prozentstaffel[verleihjahr-1]
	}

	basis, preis := basisFuer(verleihjahr, kaufpreis, neupreis)
	return Vorschlag{
		Betrag:      rundeAufCent(preis * float64(prozent) / 100),
		Prozent:     prozent,
		Basis:       basis,
		BasisPreis:  preis,
		Verleihjahr: verleihjahr,
	}
}

// basisFuer wählt den Preis: im ersten Jahr der Kaufpreis, danach der Neupreis — und
// wenn es keinen gibt, ersatzweise der Kaufpreis (benannt, nicht verschwiegen).
func basisFuer(verleihjahr int, kaufpreis, neupreis float64) (Basis, float64) {
	if verleihjahr == 1 {
		if kaufpreis > 0 {
			return BasisKaufpreis, kaufpreis
		}
		// Kein Kaufpreis erfasst, aber ein Neupreis: besser als nichts, und benannt.
		return BasisNeupreis, neupreis
	}
	if neupreis > 0 {
		return BasisNeupreis, neupreis
	}
	return BasisKaufpreisErsatzweise, kaufpreis
}

// rundeAufCent rundet kaufmännisch auf zwei Stellen — der Betrag steht in einem Brief,
// eine Zahl mit vier Nachkommastellen wäre dort ein Fehler.
func rundeAufCent(betrag float64) float64 {
	return math.Round(betrag*100) / 100
}

// Verleihjahr leitet das Verleihjahr aus den beiden Größen ab, die das System kennt:
// der Zahl der Schuljahre, in denen das Exemplar ausgeliehen war, und dem Alter des
// Exemplars im Bestand.
//
// Das Maximum von beiden, weil beide Größen unvollständig sind: Aus dem Altbestand kamen
// nur die OFFENEN Ausleihen mit, die Historie eines 2019 gekauften Buchs fehlt also. Ein
// solches Buch ist nicht im ersten Verleihjahr, nur weil wir seine Ausleihen nicht kennen
// — sein Alter im Bestand verrät es. Umgekehrt kann ein junges Buch in einem Jahr
// mehrfach verliehen worden sein.
func Verleihjahr(schuljahreMitAusleihe, jahreImBestand int) int {
	jahr := schuljahreMitAusleihe
	if jahreImBestand+1 > jahr {
		jahr = jahreImBestand + 1
	}
	if jahr < 1 {
		return 1
	}
	return jahr
}
