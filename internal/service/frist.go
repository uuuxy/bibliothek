package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// fristAngaben sind die Angaben, aus denen berechneFrist die Frist rechnet. Ein fehlendes Feld
// wäre der Nullwert und damit eine Frist von 0 Tagen; parameter_strukturen_test.go verlangt
// deshalb, dass jedes Literal jedes Feld setzt.
type fristAngaben struct {
	IstLernmittel   bool
	Medientyp       string
	LMFStichtag     string
	FristBuchTage   int
	FristMedienTage int
	ZusatzJahre     int
	// Sommerferien ist der Text der Einstellung „sommerferien"
	// (ausleihEinstellungen.Sommerferien); leer heißt die Tabelle des Programms.
	Sommerferien string
}

// berechneFrist rechnet die Frist aus dem Kennzeichen des Lernmittels, dem Medientyp und den
// eingestellten Fristen. jetzt ist die Uhr des Aufrufers, damit Tests die Frist um einen
// Rückgabetermin mit fester Uhr prüfen können.
func berechneFrist(jetzt time.Time, angaben fristAngaben) time.Time {
	// In der Zone der Schule gerechnet hängen Stichtag und Tagesende nicht von der Zone des
	// Servers ab.
	ortszeit := jetzt.In(schulzeit.Zone())

	// Ein Lernmittel bleibt das Schuljahr über beim Kind, bis zum Stichtag (Vorgabe: 31. Juli).
	if angaben.IstLernmittel {
		// Der nächste Stichtag ab heute; der Rückweg des LMF-Plans nimmt dieselbe Rechnung
		// (repository.LMFStichtagAbTag), damit beide für denselben Tag denselben Stichtag
		// finden.
		stichtag := repository.LMFStichtagAbTag(ortszeit, angaben.LMFStichtag)
		// Ein Band über mehrere Jahrgänge läuft entsprechend viele Stichtage weiter.
		stichtag = stichtag.AddDate(angaben.ZusatzJahre, 0, 0)
		return schulzeit.TagesEnde(stichtag)
	}

	// CDs, DVDs und Hörbücher sind gefragter und haben die kürzere Frist FristMedienTage.
	medientyp := strings.ToLower(angaben.Medientyp)
	if strings.Contains(medientyp, "cd") || strings.Contains(medientyp, "dvd") || strings.Contains(medientyp, "audio") {
		return lmfplan.FerientabelleAus(angaben.Sommerferien).Tagesfrist(ortszeit, angaben.FristMedienTage)
	}

	return lmfplan.FerientabelleAus(angaben.Sommerferien).Tagesfrist(ortszeit, angaben.FristBuchTage)
}

// leseJahrgang liest den Jahrgang aus dem Namen der Klasse; 0 heißt unbekannt.
func leseJahrgang(klasse string) int {
	gross := strings.ToUpper(strings.TrimSpace(klasse))
	if strings.HasPrefix(gross, "E") || gross == "EF" {
		return 11
	}
	if strings.HasPrefix(gross, "Q1") || strings.HasPrefix(gross, "Q2") {
		return 12
	}
	if strings.HasPrefix(gross, "Q3") || strings.HasPrefix(gross, "Q4") {
		return 13
	}
	// Sonst die erste Zahl im Namen.
	zahlen := regexp.MustCompile(`\d+`)
	treffer := zahlen.FindString(gross)
	if treffer != "" {
		if n, err := strconv.Atoi(treffer); err == nil {
			return n
		}
	}
	return 0
}

// bestimmeFrist bestimmt die Frist einer neuen Ausleihe ab der Uhr des Dienstes.
func (d *AusleihDienst) bestimmeFrist(ctx context.Context, exemplar *repository.Exemplar, klasse string) (time.Time, error) {
	return d.bestimmeFristAm(ctx, exemplar, klasse, d.heute())
}

// bestimmeFristAm bestimmt die Frist einer Ausleihe ab dem Tag heute: beim Nachbuchen ab dem
// Scan, weil Frist, Mahnwesen und Lesehistorie ab dem Scan rechnen. Der LMF-Plan der Klasse und
// der Ferien-Leseclub gehen der gewöhnlichen Frist vor.
func (d *AusleihDienst) bestimmeFristAm(ctx context.Context, exemplar *repository.Exemplar, klasse string, heute time.Time) (time.Time, error) {
	einst, err := d.ladeEinstellungen(ctx)
	heute = heute.In(schulzeit.Zone())
	zusatzJahre := mehrjahresbandJahre(exemplar, klasse)

	if err != nil {
		// Lassen sich die Einstellungen nicht lesen, gelten feste Werte, damit die Ausleihe
		// nicht an der Frist scheitert.
		return berechneFrist(heute, fristAngaben{
			IstLernmittel:   exemplar.IstLernmittel,
			Medientyp:       exemplar.Medientyp,
			LMFStichtag:     repository.LMFStichtagVorgabe,
			FristBuchTage:   repository.FristBuchTageVorgabe,
			FristMedienTage: repository.FristMedienTageVorgabe,
			ZusatzJahre:     zusatzJahre,
			Sommerferien:    "",
		}), nil
	}

	if frist, ok := d.ladeLMFPlanFrist(ctx, exemplar, klasse, heute, einst, zusatzJahre); ok {
		return frist, nil
	}
	if frist, ok := leseclubFrist(exemplar, einst, heute); ok {
		return frist, nil
	}

	return berechneFrist(heute, fristAngaben{
		IstLernmittel:   exemplar.IstLernmittel,
		Medientyp:       exemplar.Medientyp,
		LMFStichtag:     einst.LMFStichtag,
		FristBuchTage:   einst.FristBuchTage,
		FristMedienTage: einst.FristMedienTage,
		ZusatzJahre:     zusatzJahre,
		Sommerferien:    einst.Sommerferien,
	}), nil
}

// mehrjahresbandJahre zählt die Jahre, die ein Band über mehrere Jahrgänge über das laufende
// Schuljahr hinaus beim Kind bleibt: den Abstand zwischen seiner Klasse und JahrgangBis. Ein
// Kind über der Spanne bekommt ein Schuljahr wie jedes andere.
func mehrjahresbandJahre(exemplar *repository.Exemplar, klasse string) int {
	if exemplar.Mehrjahresband && exemplar.JahrgangBis > 0 && klasse != "" {
		jahrgang := leseJahrgang(klasse)
		if jahrgang > 0 && exemplar.JahrgangBis >= jahrgang {
			return exemplar.JahrgangBis - jahrgang
		}
	}
	return 0
}

// ladeLMFPlanFrist liefert die Frist eines Schulbuchs, wenn der veröffentlichte Rückgabe-Plan
// sie bestimmt:
//
//   - Ab dem ersten Tag des Büchertauschs gilt jedes neu ausgegebene Schulbuch dem nächsten
//     Schuljahr; die Frist ist dessen Stichtag. Das gilt für jede Klasse, auch eine, die nicht im
//     Plan steht, für Nachzügler und für Kinder, deren Termin noch bevorsteht: Im Zeitraum des
//     Tauschs holt niemand mehr ein Schulbuch für das laufende Schuljahr. Mit dessen Stichtag
//     wäre das Kind nach den Ferien überfällig und nach 14 Tagen gesperrt. Beim Veröffentlichen
//     zieht der Plan dieselbe Regel nach (SetzeLernmittelFristAbTauschbeginn).
//   - Vor dem Büchertausch ist der bevorstehende Rückgabetermin der Klasse die Frist, nur für
//     die einjährige Ausleihe. Ein Band über mehrere Jahrgänge bleibt beim Kind und rechnet
//     weiter über den Stichtag.
//   - Beim Band über mehrere Jahrgänge steckt im Sprung ins nächste Schuljahr schon eines der
//     verbleibenden Jahre, deshalb zusatzJahre-1: Ein Kind der 7 mit einem Band bis Jahrgang 9,
//     ausgegeben im Büchertausch, gibt ihn am Ende der Klasse 9 zurück.
//
// Ein Fehler beim Nachschlagen hält die Ausleihe nicht an: Dann gilt der Stichtag.
func (d *AusleihDienst) ladeLMFPlanFrist(ctx context.Context, exemplar *repository.Exemplar, klasse string, heute time.Time, einst *ausleihEinstellungen, zusatzJahre int) (time.Time, bool) {
	if exemplar.IstLernmittel && klasse != "" {
		lage, lageErr := repository.NewLMFAbfragen(d.pool).LadeRueckgabeTerminLage(ctx, klasse, heute)
		if lageErr == nil && lage.TauschBegonnen {
			stichtag := repository.LMFStichtagImFolgendenSchuljahr(heute, einst.LMFStichtag)
			return schulzeit.TagesEnde(stichtag.AddDate(max(zusatzJahre-1, 0), 0, 0)), true
		}
		if lageErr == nil && lage.Bevorstehend && zusatzJahre == 0 {
			return schulzeit.TagesEnde(lage.Naechster), true
		}
	}
	return time.Time{}, false
}

// leseclubFrist liefert das Zieldatum des Ferien-Leseclubs als Frist aller Bestände außer den
// Lernmitteln, solange es nicht vorbei ist. Bliebe der Schalter nach den Ferien an, wäre sonst
// jede neue Ausleihe sofort überfällig und nach 14 Tagen gesperrt.
func leseclubFrist(exemplar *repository.Exemplar, einst *ausleihEinstellungen, heute time.Time) (time.Time, bool) {
	if !exemplar.IstLernmittel && einst.FerienLeseclubAktiv && einst.FerienLeseclubZieldatum != nil {
		t, err := time.Parse("2006-01-02", *einst.FerienLeseclubZieldatum)
		if err == nil {
			ende := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, schulzeit.Zone())
			if ende.After(heute) {
				return ende, true
			}
		}
	}
	return time.Time{}, false
}
