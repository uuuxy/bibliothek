package service

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/pkg/betrag"
	"bibliothek/pkg/ersatzwert"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Der Vorschlag eines Betrags für den Dialog „Verlust/Schaden melden". Eine feste Zahl hätte
// keinen Bezug zum Buch. Es gelten zwei Regeln, nicht eine; deshalb tut dieser Vorschlag mehr
// als bescheidVorschlagAus:
//
//   - Lernmittel (Landesmittel): die Staffel der Arbeitshilfe zum Erlass vom 17.12.2014.
//     1. Verleihjahr voller Kaufpreis, dann 80/60/40/20 %, ab dem 6. Jahr 10 %.
//   - Schülerbücherei (Mittel des Schulträgers): „zuerst Ersatzbeschaffung, sonst Geld in
//     Höhe des NEUWERTS" (Benutzungsordnung, mittel_konzept.md 1.2). Kein Abschlag für
//     das Alter: Ein fünf Jahre alter Roman kostet in der Ersatzbeschaffung so viel wie
//     ein neuer.
//
// Die Staffel auf ein Büchereibuch anzuwenden wäre bequem und falsch: Sie steht in einer
// Arbeitshilfe für Schulbücher der Lernmittelfreiheit und gilt dort, weil das Land die Bücher
// bezahlt hat.
//
// In beiden Fällen ist es ein Vorschlag, kein Automat: Der Betrag liegt im Ermessen der Schule,
// der Mensch kann ihn überschreiben, und wenn kein Preis hinterlegt ist, bleibt das Feld bei 0
// mit einem Satz, der es sagt. Ein geratener Betrag wäre in einer Forderung gegen Eltern
// schlimmer als ein leeres Feld.

// ErsatzwertVorschlag ist die Antwort für den Melde-Dialog.
type ErsatzwertVorschlag struct {
	// Betrag ist der Vorschlag; 0 heißt „kein Preis hinterlegt".
	Betrag betrag.Cent `json:"betrag"`
	// Herleitung sagt dem Personal, warum dieser Betrag vorgeschlagen wird — damit im
	// Dialog nachvollziehbar steht, woher die Zahl kommt, bevor sie in einer Forderung
	// landet.
	Herleitung string `json:"herleitung"`
	// Topf ist das Eigentum des Exemplars und unterscheidet die beiden Regeln (Staffel des
	// Landes oder Neuwert); die Herleitung benennt sie.
	Topf string `json:"topf"`
	// Bekannt sagt, ob dem Betrag ein Preis zugrunde liegt; erst damit wird 0,00 € lesbar:
	// Ergebnis eines Totalschadens oder fehlende Angabe?
	//
	// Es ist ein eigenes Feld, weil die Oberfläche es sonst aus dem deutschen Satz der
	// Herleitung lesen müsste. Ein Satz als Schnittstelle hält bis zur ersten Umformulierung;
	// danach zeigte die Karte still „Ersatzwert heute: 0,00 €" über einem Buch, dessen Preis
	// bloß niemand erfasst hat.
	Bekannt bool `json:"bekannt"`
}

// ErsatzwertEingabe sind die Größen, aus denen jeder Betragsvorschlag entsteht, gleich ob er
// aus einem Exemplar (Melde-Dialog), einer offenen Forderung oder einer überfälligen Ausleihe
// kommt (Bescheid-Vorschlag). Ein Typ, eine Rechnung, eine Herleitung: Mit zwei Rechnungen
// nannten Dialog und Brief für dasselbe Buch verschiedene Beträge.
type ErsatzwertEingabe struct {
	Kaufpreis       betrag.Cent
	Listenpreis     betrag.Cent
	ZustandAbschlag int
	// Die beiden Größen des Verleihjahrs — nur die Staffel benutzt sie.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
	// Topf entscheidet über die Regel, nicht über einen Faktor: Staffel des Landes für ein Buch
	// des Landes, sonst Neuwert der Benutzungsordnung. Der Topf ist das Eigentum nach
	// repository.ExemplarTopfSQL und nicht ist_lernmittel des Titels: Eine Lektüre aus Mitteln
	// des Landes gehört dem Land, auch wenn sie kein Lernmittel ist.
	Topf string
}

// Rechne wählt die Regel nach dem Topf: die Staffel des Landes für ein Buch des Landes, sonst
// den Neuwert der Benutzungsordnung.
func (e ErsatzwertEingabe) Rechne(quelle ersatzwert.Preisquelle) ersatzwert.Vorschlag {
	if e.Topf != mitteltopf.Land {
		return ersatzwert.BerechneNeuwert(e.Kaufpreis, e.Listenpreis, e.ZustandAbschlag, quelle)
	}
	return ersatzwert.Berechne(ersatzwert.Verleihjahr(e.SchuljahreMitAusleihe, e.SchuljahreImBestand),
		e.Kaufpreis, e.Listenpreis, e.ZustandAbschlag, quelle)
}

// eingabeAusGroessen ist der Weg des Melde-Dialogs.
func eingabeAusGroessen(g repository.ErsatzwertGroessen) ErsatzwertEingabe {
	return ErsatzwertEingabe{
		Kaufpreis: g.Kaufpreis, Listenpreis: g.Listenpreis, ZustandAbschlag: g.ZustandAbschlag,
		SchuljahreMitAusleihe: g.SchuljahreMitAusleihe, SchuljahreImBestand: g.SchuljahreImBestand,
		Topf: g.Topf,
	}
}

// ErsatzwertVorschlagAus rechnet und formuliert die Herleitung.
func ErsatzwertVorschlagAus(g repository.ErsatzwertGroessen, quelle ersatzwert.Preisquelle) ErsatzwertVorschlag {
	v := eingabeAusGroessen(g).Rechne(quelle)
	return ErsatzwertVorschlag{
		Betrag:     v.Betrag,
		Herleitung: ErsatzwertHerleitung(v),
		Topf:       g.Topf,
		Bekannt:    v.HatPreis(),
	}
}

// LadeErsatzwertPreisquelle liest aus den Einstellungen, welcher Preis ab dem zweiten
// Verleihjahr die Grundlage ist (Anforderungsliste Nr. 3).
//
// Scheitert das Lesen, gilt die Regel der Arbeitshilfe. Das ist die einzige Richtung, die hier
// zu vertreten ist: Ein Fehler beim Lesen einer Einstellung darf nicht dazu führen, dass eine
// Forderung nach einer anderen Grundlage entsteht als die davor.
func LadeErsatzwertPreisquelle(ctx context.Context, pool db.Pool) ersatzwert.Preisquelle {
	einst, err := repository.NewEinstellungsAbfragen(pool).Lade(ctx)
	if err != nil || einst == nil || !einst.ErsatzwertImmerKaufpreis {
		return ersatzwert.PreisquelleListenpreis
	}
	return ersatzwert.PreisquelleKaufpreis
}

// ErsatzwertHerleitung formuliert, wie der Vorschlag zustande kommt.
//
// Der Satz ist kein Beiwerk: Er ist das, was eine Bibliothekskraft im Bescheid nachrechnen
// kann. Eine Zahl ohne ihn wäre für sie so wenig wert wie eine feste Pauschale.
func ErsatzwertHerleitung(v ersatzwert.Vorschlag) string {
	// Dieselbe Regel wie das Feld `bekannt` der Antwort — eine Funktion, nicht zwei
	// Bedingungen, die zufällig dasselbe sagen.
	if !v.HatPreis() {
		return "kein Preis hinterlegt — Betrag bitte eintragen"
	}
	basis := "Kaufpreis"
	switch v.Basis {
	case ersatzwert.BasisNeupreis:
		basis = "Listenpreis"
	case ersatzwert.BasisKaufpreisErsatzweise:
		// Ohne Klammer in der Klammer: Der Satz steht in einem Brief.
		basis = "Kaufpreis, kein Listenpreis hinterlegt"
	case ersatzwert.BasisKaufpreisGewaehlt:
		// „so eingestellt" statt „kein Listenpreis hinterlegt": Der Satz sagt, dass die
		// Schule die Grundlage gewählt hat, und behauptet nichts über die Datenlage.
		basis = "Kaufpreis (so eingestellt)"
	case ersatzwert.BasisKaufpreis:
		basis = "Kaufpreis"
	}
	// Verleihjahr 0 heißt: die Neuwert-Regel der Benutzungsordnung (Bestand des
	// Schulträgers). Dort gibt es keinen Staffelsatz zu nennen — und der Satz muss
	// ausdrücklich „ohne Altersabschlag" sagen, weil ein Zustandsabschlag gleich
	// dahinter stehen kann.
	satz := fmt.Sprintf("Bücherei-Bestand: Neuwert ohne Altersabschlag, %s (%s)",
		v.BasisPreis.Euro(), basis)
	if v.Verleihjahr > 0 {
		satz = fmt.Sprintf("%d. Verleihjahr → %d %% von %s (%s)",
			v.Verleihjahr, v.Prozent, v.BasisPreis.Euro(), basis)
	}
	if v.ZustandAbschlag > 0 {
		satz += fmt.Sprintf(", abzüglich %d %% für den Zustand", v.ZustandAbschlag)
	}
	return satz
}
