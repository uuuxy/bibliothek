// Package ersatzwert rechnet den Vorschlag für den Schadensersatz an einem Exemplar: für ein
// Lernmittel nach der Staffel, die die Arbeitshilfe der Schule empfiehlt, für die
// Schülerbücherei nach dem Neuwert.
//
// Es bleibt ein Vorschlag: Der Betrag liegt im Ermessen der Schule („je nach Zustand des
// Lehrwerks, Ausleihhäufigkeit etc."). Das Ergebnis trägt deshalb seine Herleitung mit, damit
// der Dialog sagen kann, warum er 60 % von 41,50 € vorschlägt, und ein Mensch den Betrag
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
// Die Basis wechselt zwischen dem ersten und dem zweiten Jahr vom Kaufpreis zum heutigen
// Neupreis: Im ersten Jahr ist das Buch neu, danach zählt, was ein Ersatz heute kostet.
package ersatzwert

import "math"

// prozentstaffel nennt den Prozentsatz je Verleihjahr, beginnend mit dem ersten; ab dem
// sechsten Jahr gilt der letzte Wert.
var prozentstaffel = []int{100, 80, 60, 40, 20, 10}

// Basis benennt, auf welchen Preis sich der Prozentsatz bezieht.
type Basis string

const (
	// BasisKaufpreis ist der Preis, zu dem die Schule das Buch beschafft hat (1. Verleihjahr).
	BasisKaufpreis Basis = "kaufpreis"
	// BasisNeupreis ist, was ein Ersatz heute kostet (ab dem 2. Verleihjahr).
	BasisNeupreis Basis = "neupreis"
	// BasisKaufpreisErsatzweise heißt: Ab dem 2. Verleihjahr gälte der Neupreis, es ist aber
	// keiner hinterlegt, und gerechnet wird mit dem Kaufpreis. Der Dialog sagt das, damit
	// niemand der Zahl eine Genauigkeit zuschreibt, die sie nicht hat.
	BasisKaufpreisErsatzweise Basis = "kaufpreis_ersatzweise"
	// BasisKaufpreisGewaehlt heißt: Es gibt einen Listenpreis, die Schule hat aber eingestellt,
	// dass immer der Kaufpreis gilt. Der Wert ist ein eigener, weil „kein Listenpreis
	// hinterlegt" hier eine falsche Auskunft im Bescheid wäre.
	BasisKaufpreisGewaehlt Basis = "kaufpreis_gewaehlt"
)

// Preisquelle sagt, welcher Preis ab dem zweiten Verleihjahr die Grundlage ist; die Schule
// stellt es ein.
//
// Der Nullwert ist PreisquelleListenpreis, weil die Arbeitshilfe zum Erlass vom 17.12.2014 ab
// dem zweiten Verleihjahr „80 % des Neupreises zum Zeitpunkt des Verlusts" verlangt. Eine
// ungesetzte Einstellung und ein Rufer, der die Frage nicht kennt, bekommen so die Regel und
// nicht die Abweichung davon.
type Preisquelle string

const (
	// PreisquelleListenpreis ist die Vorgabe: der heutige Neupreis, wenn einer erfasst ist.
	PreisquelleListenpreis Preisquelle = ""
	// PreisquelleKaufpreis heißt: immer der Einkaufspreis der Schule, auch wenn ein
	// Listenpreis erfasst ist. Die Schule wählt es, und die Herleitung nennt es.
	PreisquelleKaufpreis Preisquelle = "kaufpreis"
)

// Vorschlag ist das Ergebnis mit seiner Herleitung: der Betrag, der Prozentsatz, die Basis und
// der Preis, auf den sich der Prozentsatz bezieht.
type Vorschlag struct {
	Betrag      float64
	Prozent     int
	Basis       Basis
	BasisPreis  float64
	Verleihjahr int
	// ZustandAbschlag ist der Prozentsatz, der für den Zustand dieses Exemplars vom Zeitwert
	// abgezogen wurde (0 heißt keiner). Er steht hier, damit die Herleitung ihn nennen kann:
	// „60 % von 41,50 €, abzüglich 20 % für den Zustand".
	ZustandAbschlag int
}

// HatPreis sagt, ob dem Vorschlag ein Preis zugrunde liegt.
//
// 0,00 € heißt zweierlei: Bei 100 % Wertverlust ist die Null das Ergebnis und gehört auf den
// Bildschirm. Bei einem Titel ohne Preis sagt sie nichts, und „Ersatzwert heute: 0,00 €" wäre
// falsch, weil nur der Preis nicht erfasst ist. Die Oberfläche fragt deshalb diesen Wert und
// liest nicht den Satz der Herleitung.
func (v Vorschlag) HatPreis() bool {
	return v.BasisPreis > 0
}

// Berechne liefert den Vorschlag für ein Lernmittel nach der Staffel.
//
// verleihjahr ist mindestens 1: Ein Buch, das neu im Regal steht, ist im ersten Verleihjahr,
// und eine 0 aus einer unvollständigen Historie soll nicht zu 0 € führen. Fehlen beide Preise,
// ist der Betrag 0, und ein Mensch trägt ihn ein; ein geratener Betrag wäre in einem Bescheid
// schlimmer als ein leeres Feld.
//
// zustandAbschlag ist der Prozentsatz für den Zustand dieses Exemplars, etwa 20 % für einen
// Wasserschaden. Er wirkt nach der Staffel: erst der Zeitwert des Werks, dann der Abzug für
// dieses eine Stück. So rechnet es ein Mensch im Bescheid nach („60 % von 41,50 €, davon 20 %
// ab").
//
// Werte außerhalb von 0 bis 100 werden gekappt und nicht abgelehnt: Die Spalte lässt sie nicht
// zu (chk_zustand_abwertung_bereich), und ein negativer Abschlag, der den Betrag erhöht, wäre
// in einer Forderung schlimmer als ein übergangener Tippfehler.
func Berechne(verleihjahr int, kaufpreis, neupreis float64, zustandAbschlag int, quelle Preisquelle) Vorschlag {
	if verleihjahr < 1 {
		verleihjahr = 1
	}
	prozent := prozentstaffel[len(prozentstaffel)-1]
	if verleihjahr <= len(prozentstaffel) {
		prozent = prozentstaffel[verleihjahr-1]
	}
	if zustandAbschlag < 0 {
		zustandAbschlag = 0
	}
	if zustandAbschlag > 100 {
		zustandAbschlag = 100
	}

	basis, preis := basisFuer(verleihjahr, kaufpreis, neupreis, quelle)
	zeitwert := preis * float64(prozent) / 100
	return Vorschlag{
		Betrag:          rundeAufCent(zeitwert * float64(100-zustandAbschlag) / 100),
		Prozent:         prozent,
		Basis:           basis,
		BasisPreis:      preis,
		Verleihjahr:     verleihjahr,
		ZustandAbschlag: zustandAbschlag,
	}
}

// BerechneNeuwert liefert den Vorschlag für den Bestand, der dem Schulträger gehört: die
// Schülerbücherei.
//
// Die Staffel gilt hier nicht. Sie steht in einer Arbeitshilfe für die Lehrwerke der
// Lernmittelfreiheit, die das Land bezahlt; zur Bücherei sagen Erlass und Arbeitshilfe nichts.
// Für sie gilt die Benutzungsordnung: „zuerst Ersatzbeschaffung, sonst Geld in Höhe des
// Neuwerts" (docs/mittel_konzept.md 1.2).
//
// Neuwert ist der Listenpreis und nicht der Kaufpreis, weil ein Ersatz den heutigen Preis
// kostet. Ein 2015 für 8 € gekaufter Roman, der heute 14 € kostet, wäre mit dem Kaufpreis um
// 6 € zu niedrig ersetzt, und das Geld gehört einem fremden Träger.
//
// Der Abschlag für den Zustand zählt auch hier, das Alter nicht: Ein zehn Jahre alter Roman
// kostet in der Ersatzbeschaffung so viel wie ein neuer.
func BerechneNeuwert(kaufpreis, listenpreis float64, zustandAbschlag int, quelle Preisquelle) Vorschlag {
	if zustandAbschlag < 0 {
		zustandAbschlag = 0
	}
	if zustandAbschlag > 100 {
		zustandAbschlag = 100
	}

	basis, preis := neuwertBasis(kaufpreis, listenpreis, quelle)
	return Vorschlag{
		Betrag:     rundeAufCent(preis * float64(100-zustandAbschlag) / 100),
		Prozent:    100,
		Basis:      basis,
		BasisPreis: preis,
		// Verleihjahr 0 heißt, dass diese Rechnung kein Verleihjahr kennt. Daran unterscheidet
		// die Herleitung die zwei Regeln.
		Verleihjahr:     0,
		ZustandAbschlag: zustandAbschlag,
	}
}

// neuwertBasis wählt den Preis für den Neuwert: den Listenpreis, wenn es einen gibt und die
// Schule nichts anderes eingestellt hat, sonst den Kaufpreis. Die Basis nennt, welcher es ist.
func neuwertBasis(kaufpreis, listenpreis float64, quelle Preisquelle) (Basis, float64) {
	if quelle == PreisquelleKaufpreis && kaufpreis > 0 {
		return BasisKaufpreisGewaehlt, kaufpreis
	}
	if listenpreis > 0 {
		return BasisNeupreis, listenpreis
	}
	return BasisKaufpreisErsatzweise, kaufpreis
}

// basisFuer wählt den Preis für die Staffel: im ersten Jahr den Kaufpreis, danach den
// Neupreis und, wenn es keinen gibt, ersatzweise den Kaufpreis. Die Basis nennt, welcher es
// ist.
//
// Die eingestellte Preisquelle wirkt erst ab dem zweiten Jahr. Im ersten gilt der Kaufpreis,
// weil das Buch neu war, als es verliehen wurde.
func basisFuer(verleihjahr int, kaufpreis, neupreis float64, quelle Preisquelle) (Basis, float64) {
	if verleihjahr == 1 {
		if kaufpreis > 0 {
			return BasisKaufpreis, kaufpreis
		}
		// Kein Kaufpreis erfasst, aber ein Neupreis: besser als nichts, und die Basis nennt es.
		return BasisNeupreis, neupreis
	}
	if quelle == PreisquelleKaufpreis {
		// So eingestellt. Ohne Kaufpreis bleibt der Listenpreis: Eine 0 in einer Forderung wäre
		// schlimmer als eine benannte Abweichung von der Einstellung.
		if kaufpreis > 0 {
			return BasisKaufpreisGewaehlt, kaufpreis
		}
		return BasisNeupreis, neupreis
	}
	if neupreis > 0 {
		return BasisNeupreis, neupreis
	}
	return BasisKaufpreisErsatzweise, kaufpreis
}

// rundeAufCent rundet kaufmännisch auf zwei Stellen; der Betrag steht in einem Brief.
func rundeAufCent(betrag float64) float64 {
	return math.Round(betrag*100) / 100
}

// Verleihjahr leitet das Verleihjahr aus den zwei Größen ab, die das Programm kennt: der Zahl
// der Schuljahre, in denen das Exemplar ausgeliehen war, und seinem Alter im Bestand.
//
// Es gilt die größere, weil beide unvollständig sind: Aus dem Altbestand kamen nur die offenen
// Ausleihen mit, die Historie eines 2019 gekauften Buchs fehlt also, und sein Alter verrät,
// dass es nicht im ersten Verleihjahr ist. Umgekehrt kann ein junges Buch in einem Jahr
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
