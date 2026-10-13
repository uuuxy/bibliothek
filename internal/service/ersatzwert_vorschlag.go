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

// Der Betragsvorschlag für den Dialog „Verlust/Schaden melden" (OFFEN.md 9.3 a).
//
// Bis zum 17.09.2026 stand dort eine feste 15,00 € — eine Zahl ohne jeden Bezug zum Buch.
// Das Protokoll des Medienzentrums vom 16.09.2026 nennt genau das: „weder Einkaufs- bzw.
// Listenpreise noch Beschädigungsgrade hinterlegt, fehlende Restwertberechnung z.Zt.
// Handeingabe."
//
// ZWEI Regeln, nicht eine — das ist der Grund, warum dieser Handler mehr tut als
// bescheidVorschlagAus:
//
//   - Lernmittel (Landesmittel): die Staffel der Arbeitshilfe zum Erlass vom 17.12.2014.
//     1. Verleihjahr voller Kaufpreis, dann 80/60/40/20 %, ab dem 6. Jahr 10 %.
//   - Schülerbücherei (Mittel des Schulträgers): „zuerst Ersatzbeschaffung, sonst Geld in
//     Höhe des NEUWERTS" (Benutzungsordnung, mittel_konzept.md 1.2). Kein Abschlag für
//     das Alter — ein fünf Jahre alter Roman kostet in der Ersatzbeschaffung so viel wie
//     ein neuer.
//
// Die Staffel auf ein Büchereibuch anzuwenden wäre bequem und falsch: Sie steht in einer
// Arbeitshilfe für Schulbücher der Lernmittelfreiheit und gilt dort, weil das Land die
// Bücher bezahlt hat.
//
// In BEIDEN Fällen ist es ein Vorschlag, kein Automat: Der Betrag liegt im Ermessen der
// Schule, der Mensch kann ihn überschreiben, und wenn kein Preis hinterlegt ist, bleibt
// das Feld bei 0 mit einem Satz, der es sagt. Ein geratener Betrag wäre in einer
// Forderung gegen Eltern schlimmer als ein leeres Feld.

// ErsatzwertVorschlag ist die Antwort für den Melde-Dialog.
type ErsatzwertVorschlag struct {
	// Betrag ist der Vorschlag in Euro; 0 heißt „kein Preis hinterlegt".
	Betrag float64 `json:"betrag"`
	// Herleitung sagt dem Personal, WARUM dieser Betrag vorgeschlagen wird — damit im
	// Dialog nachvollziehbar steht, woher die Zahl kommt, bevor sie in einer Forderung
	// landet.
	Herleitung string `json:"herleitung"`
	// Topf ist das Eigentum des Exemplars und unterscheidet die beiden Regeln (Staffel des
	// Landes oder Neuwert); die Herleitung benennt sie.
	Topf string `json:"topf"`
	// Bekannt sagt, ob dem Betrag ein Preis zugrunde liegt — die Unterscheidung, die 0,00 €
	// erst lesbar macht: Ergebnis eines Totalschadens oder fehlende Angabe?
	//
	// Sie steht als eigenes Feld hier, weil die Oberfläche sie sonst aus dem deutschen
	// Herleitungssatz lesen müsste, und das tat sie bis zum 17.09.2026 auch
	// (`startsWith("kein Preis hinterlegt")`). Ein Satz als Schnittstelle hält bis zur
	// ersten Umformulierung; danach zeigt die Karte still „Ersatzwert heute: 0,00 €" über
	// einem Buch, dessen Preis bloß niemand erfasst hat.
	Bekannt bool `json:"bekannt"`
}

// ErsatzwertEingabe sind die Größen, aus denen JEDER Betragsvorschlag entsteht — egal ob
// er aus einem Exemplar (Melde-Dialog), einer offenen Forderung oder einer überfälligen
// Ausleihe kommt (Bescheid-Vorschlag).
//
// Bis zum 17.09.2026 waren es drei Wege mit ZWEI Rechnungen: Der Melde-Dialog wählte die
// Regel nach IstLernmittel, die beiden Bescheid-Wege wendeten die Staffel der
// Arbeitshilfe auf ALLES an — auch auf Büchereibücher, für die sie nicht gilt. Für
// dasselbe Buch nannten Dialog und Brief damit verschiedene Beträge, und der Brief war
// der falsche. Ein Typ, eine Rechnung, eine Herleitung.
type ErsatzwertEingabe struct {
	Kaufpreis       float64
	Listenpreis     float64
	ZustandAbschlag int
	// Die beiden Größen des Verleihjahrs — nur die Staffel benutzt sie.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
	// Topf entscheidet über die REGEL, nicht über einen Faktor: Staffel des Landes für ein
	// Buch des Landes, sonst Neuwert der Benutzungsordnung. Der Topf ist das Eigentum nach
	// repository.ExemplarTopfSQL — bis zum 29.09.2026 entschied ist_lernmittel des Titels,
	// und eine Lektüre aus LMF-Mitteln bekam den Neuwert, obwohl sie dem Land gehört
	// (entschieden am 29.09.2026, docs/OFFEN.md 4.24).
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

// eingabeAusGroessen: der Weg des Melde-Dialogs.
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

// ErsatzwertPreisquelle liest aus den Einstellungen, welcher Preis ab dem zweiten Verleihjahr die
// Grundlage ist (Anforderungsliste Nr. 3, OFFEN.md 9.8 Stufe 4).
//
// Scheitert das Lesen, gilt die Regel der Arbeitshilfe. Das ist die einzige Richtung, die
// hier zu vertreten ist: Ein Fehler beim Lesen einer Einstellung darf nicht dazu führen,
// dass eine Forderung nach einer anderen Grundlage entsteht als die davor.
func ErsatzwertPreisquelle(ctx context.Context, pool db.PgxPoolIface) ersatzwert.Preisquelle {
	einst, err := repository.NewSystemSettingsRepository(pool).GetSettings(ctx)
	if err != nil || einst == nil || !einst.ErsatzwertImmerKaufpreis {
		return ersatzwert.PreisquelleListenpreis
	}
	return ersatzwert.PreisquelleKaufpreis
}

// ErsatzwertHerleitung formuliert, wie der Vorschlag zustande kommt.
//
// Der Satz ist kein Beiwerk: Er ist das, was eine Bibliothekskraft im Bescheid nachrechnen
// kann. Eine Zahl ohne ihn ist für sie dasselbe wie die feste 15,00 €, die bis zum
// 17.09.2026 im Melde-Dialog stand.
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
	// ausdrücklich „ohne ALTERSabschlag" sagen, weil ein Zustandsabschlag gleich
	// dahinter stehen kann.
	satz := fmt.Sprintf("Bücherei-Bestand: Neuwert ohne Altersabschlag, %s (%s)",
		betrag.Euro(v.BasisPreis), basis)
	if v.Verleihjahr > 0 {
		satz = fmt.Sprintf("%d. Verleihjahr → %d %% von %s (%s)",
			v.Verleihjahr, v.Prozent, betrag.Euro(v.BasisPreis), basis)
	}
	if v.ZustandAbschlag > 0 {
		satz += fmt.Sprintf(", abzüglich %d %% für den Zustand", v.ZustandAbschlag)
	}
	return satz
}
