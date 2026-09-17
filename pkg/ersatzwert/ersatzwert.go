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
	// BasisKaufpreisGewaehlt heißt: Es GIBT einen Listenpreis, die Schule hat aber
	// eingestellt, dass immer der Kaufpreis gilt. Ein eigener Wert, weil die Herleitung
	// beides unterscheiden muss — „kein Listenpreis hinterlegt" wäre hier eine falsche
	// Auskunft über die Datenlage, und in einem Bescheid steht keine falsche Auskunft.
	BasisKaufpreisGewaehlt Basis = "kaufpreis_gewaehlt"
)

// Preisquelle sagt, welcher Preis ab dem zweiten Verleihjahr die Grundlage ist
// (Anforderungsliste Nr. 3: „Einkaufspreis UND Listenpreis hinterlegen, für das
// Mahnwesen auswählbar, welcher gilt").
//
// Der Nullwert ist PreisquelleListenpreis, und das ist kein Zufall: Die Arbeitshilfe
// zum Erlass vom 17.12.2014 verlangt ab dem zweiten Verleihjahr „80 % des Neupreises
// zum Zeitpunkt des Verlusts". Wer diesen Typ nicht setzt — eine ungesetzte Einstellung,
// ein Aufrufer, der die Frage nicht kennt —, bekommt also die bindende Regel und nicht
// die Abweichung davon.
type Preisquelle string

const (
	// PreisquelleListenpreis ist die Vorgabe: der heutige Neupreis, wenn einer erfasst ist.
	PreisquelleListenpreis Preisquelle = ""
	// PreisquelleKaufpreis heißt: immer der Einkaufspreis der Schule, auch wenn ein
	// Listenpreis erfasst ist. Eine Wahl der Schule, die die Herleitung benennt.
	PreisquelleKaufpreis Preisquelle = "kaufpreis"
)

// Vorschlag ist das Ergebnis: der Betrag, der Prozentsatz, die Basis und der Preis, auf
// den sich der Prozentsatz bezieht.
type Vorschlag struct {
	Betrag      float64
	Prozent     int
	Basis       Basis
	BasisPreis  float64
	Verleihjahr int
	// ZustandAbschlag ist der Prozentsatz, der für den Zustand DIESES Exemplars vom
	// Zeitwert abgezogen wurde (0 = keiner). Er steht hier, damit die Herleitung ihn
	// nennen kann: „60 % von 41,50 €, abzüglich 20 % für den Zustand" ist nachvollziehbar,
	// eine Zahl ohne diesen Satz nicht.
	ZustandAbschlag int
}

// Rechne liefert den Vorschlag für ein Exemplar.
//
// verleihjahr wird bei 1 abgeschnitten: Ein Buch, das gerade neu im Regal steht, ist im
// ersten Verleihjahr — eine 0 aus einer unvollständigen Historie darf nicht zu 0 € führen.
// Fehlen beide Preise, ist der Betrag 0 und der Mensch trägt ihn selbst ein; ein geratener
// Betrag wäre in einem Bescheid schlimmer als ein leeres Feld.
//
// zustandAbschlag ist der Prozentsatz für den Zustand DIESES Exemplars (Migration 127,
// Anforderungsliste Nr. 2: „20 % durch Wasserschaden"). Er wirkt NACH der Staffel, nicht
// neben ihr: erst der Zeitwert des Werks, dann der Abzug für dieses eine Stück. Die
// Reihenfolge ist die Lesart des Satzes „60 % von 41,50 €, davon 20 % ab für den
// Wasserschaden" — andersherum käme dieselbe Zahl heraus, aber die Herleitung wäre nicht
// mehr die, die ein Mensch im Bescheid nachrechnet.
//
// Werte außerhalb 0–100 werden gekappt statt abgelehnt: Die Spalte lässt sie ohnehin nicht
// zu (chk_zustand_abwertung_bereich), und ein negativer Abschlag, der den Betrag ERHÖHT,
// wäre in einer Forderung schlimmer als ein ignorierter Tippfehler.
func Rechne(verleihjahr int, kaufpreis, neupreis float64, zustandAbschlag int, quelle Preisquelle) Vorschlag {
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

// RechneNeuwert liefert den Vorschlag für den Bestand, der dem SCHULTRÄGER gehört —
// die Schülerbücherei.
//
// Zwei Regeln, nicht eine: Die Staffel der Arbeitshilfe gilt hier NICHT. Sie steht in
// einer Arbeitshilfe für Lehrwerke der Lernmittelfreiheit, und das Land bezahlt diese
// Bücher; für die Bücherei gilt die Benutzungsordnung: „zuerst Ersatzbeschaffung, sonst
// Geld in Höhe des Neuwerts" (mittel_konzept.md 1.2). Zur Bücherei sagt weder der Erlass
// vom 17.12.2014 noch die Arbeitshilfe ein Wort — am 17.09.2026 nachgesehen.
//
// „Neuwert" ist deshalb der LISTENPREIS, nicht der Kaufpreis: Was ein Ersatz heute
// kostet, ist der heutige Preis. Bis zum 17.09.2026 rechnete das Programm hier mit dem
// Einkaufspreis, weil es keinen anderen kannte — ein 2015 für 8 € gekaufter Roman, der
// heute 14 € kostet, wurde mit 8 € ersetzt, und die fehlenden 6 € waren Geld eines
// fremden Trägers.
//
// Der Zustandsabschlag zählt auch hier (Anforderungsliste Nr. 2 nennt „Medien", nicht
// „Lernmittel"). Was NICHT zählt, ist das Alter: Ein zehn Jahre alter Roman kostet in der
// Ersatzbeschaffung so viel wie ein neuer.
func RechneNeuwert(kaufpreis, listenpreis float64, zustandAbschlag int, quelle Preisquelle) Vorschlag {
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
		// Verleihjahr 0 heißt: Diese Rechnung kennt kein Verleihjahr. Daran unterscheidet
		// die Herleitung die beiden Regeln, ohne ein zweites Feld dafür zu brauchen.
		Verleihjahr:     0,
		ZustandAbschlag: zustandAbschlag,
	}
}

// neuwertBasis wählt den Preis für die Neuwert-Regel: der Listenpreis, wenn es einen
// gibt und die Schule nichts anderes eingestellt hat, sonst der Kaufpreis — benannt.
func neuwertBasis(kaufpreis, listenpreis float64, quelle Preisquelle) (Basis, float64) {
	if quelle == PreisquelleKaufpreis && kaufpreis > 0 {
		return BasisKaufpreisGewaehlt, kaufpreis
	}
	if listenpreis > 0 {
		return BasisNeupreis, listenpreis
	}
	return BasisKaufpreisErsatzweise, kaufpreis
}

// basisFuer wählt den Preis: im ersten Jahr der Kaufpreis, danach der Neupreis — und
// wenn es keinen gibt, ersatzweise der Kaufpreis (benannt, nicht verschwiegen).
//
// Die eingestellte Preisquelle greift erst ab dem zweiten Jahr, weil im ersten ohnehin
// der Kaufpreis gilt: Das Buch war neu, als es verliehen wurde. Eine Einstellung, die
// dort etwas änderte, änderte nichts — außer der Begründung, und die wäre dann falsch.
func basisFuer(verleihjahr int, kaufpreis, neupreis float64, quelle Preisquelle) (Basis, float64) {
	if verleihjahr == 1 {
		if kaufpreis > 0 {
			return BasisKaufpreis, kaufpreis
		}
		// Kein Kaufpreis erfasst, aber ein Neupreis: besser als nichts, und benannt.
		return BasisNeupreis, neupreis
	}
	if quelle == PreisquelleKaufpreis {
		// So eingestellt. Ohne Kaufpreis bleibt nur der Listenpreis — eine 0 in einer
		// Forderung wäre schlimmer als eine benannte Abweichung von der Einstellung.
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
