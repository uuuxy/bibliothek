package service

import (
	"context"
	"strings"
	"time"

	"bibliothek/internal/pkg/lmfplan"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/repository"
)

// FristHoechstensJahre begrenzt eine Frist von Hand nach oben. Die längste, die das Programm selbst
// vergibt, hat ein Band über alle Jahrgänge (1 bis 13, chk_jahrgang_spanne): zwölf Jahre nach dem
// nächsten Stichtag. Ein Datum dahinter ist ein Tippfehler.
const FristHoechstensJahre = 13

// LetzteFristVonHand ist das späteste Datum, das eine Frist von Hand tragen kann: das Tagesende
// FristHoechstensJahre nach jetzt. Es gilt für die Frist einer Ausleihe wie für die aller
// Lernmittel einer Klasse.
func LetzteFristVonHand(jetzt time.Time) time.Time {
	return schulzeit.TagesEnde(jetzt.AddDate(FristHoechstensJahre, 0, 0))
}

// fristAngaben sind die Angaben, aus denen berechneFrist die Frist rechnet. Ein fehlendes Feld
// wäre der Nullwert und damit eine Frist von 0 Tagen; parameter_strukturen_test.go verlangt
// deshalb, dass jedes Literal jedes Feld setzt.
type fristAngaben struct {
	IstLernmittel bool
	Medientyp     string
	// Klasse bestimmt den Stichtag eines Lernmittels (repository.LMFStichtagAbTag).
	Klasse          string
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

	// Ein Lernmittel bleibt das Schuljahr über beim Kind, bis zum Stichtag. Der Rückweg des
	// LMF-Plans nimmt dieselbe Rechnung, damit beide für denselben Tag denselben Stichtag finden;
	// ein Band über mehrere Jahrgänge läuft entsprechend viele Schuljahre weiter.
	if angaben.IstLernmittel {
		ferien := lmfplan.FerientabelleAus(angaben.Sommerferien)
		return schulzeit.TagesEnde(repository.LMFStichtagAbTag(ortszeit, angaben.Klasse, angaben.ZusatzJahre, ferien))
	}

	// CDs, DVDs und Hörbücher sind gefragter und haben die kürzere Frist FristMedienTage.
	medientyp := strings.ToLower(angaben.Medientyp)
	if strings.Contains(medientyp, "cd") || strings.Contains(medientyp, "dvd") || strings.Contains(medientyp, "audio") {
		return lmfplan.FerientabelleAus(angaben.Sommerferien).Tagesfrist(ortszeit, angaben.FristMedienTage)
	}

	return lmfplan.FerientabelleAus(angaben.Sommerferien).Tagesfrist(ortszeit, angaben.FristBuchTage)
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
		frist := berechneFrist(heute, fristAngaben{
			IstLernmittel:   exemplar.IstLernmittel,
			Medientyp:       exemplar.Medientyp,
			Klasse:          klasse,
			FristBuchTage:   repository.FristBuchTageVorgabe,
			FristMedienTage: repository.FristMedienTageVorgabe,
			ZusatzJahre:     zusatzJahre,
			Sommerferien:    "",
		})
		return fristOhneSchulbuch(frist, exemplar, klasse, "", heute), nil
	}

	if frist, ok := d.ladeLMFPlanFrist(ctx, exemplar, klasse, heute, einst, zusatzJahre); ok {
		return frist, nil
	}
	// Ein Abgänger ist nach den Sommerferien nicht mehr an der Schule: Der Leseclub gilt für ihn
	// nicht (AbgaengerFrist).
	if !repository.IstAbschlussklasse(klasse) {
		if frist, ok := leseclubFrist(exemplar, einst, heute); ok {
			return frist, nil
		}
	}

	frist := berechneFrist(heute, fristAngaben{
		IstLernmittel:   exemplar.IstLernmittel,
		Medientyp:       exemplar.Medientyp,
		Klasse:          klasse,
		FristBuchTage:   einst.FristBuchTage,
		FristMedienTage: einst.FristMedienTage,
		ZusatzJahre:     zusatzJahre,
		Sommerferien:    einst.Sommerferien,
	})
	return fristOhneSchulbuch(frist, exemplar, klasse, einst.Sommerferien, heute), nil
}

// fristOhneSchulbuch wendet AbgaengerFrist auf alles an, was kein Lernmittel ist: Schulbücher
// haben ihre eigene Regel.
func fristOhneSchulbuch(frist time.Time, exemplar *repository.Exemplar, klasse, sommerferien string, heute time.Time) time.Time {
	if exemplar.IstLernmittel {
		return frist
	}
	return AbgaengerFrist(frist, klasse, lmfplan.FerientabelleAus(sommerferien), heute)
}

// AbgaengerFrist begrenzt die Frist eines Abgängers auf den letzten Schultag vor den Sommerferien,
// mit denen er die Schule verlässt; wer danach noch ausleiht, bekommt das Ende des Tages. Für
// Schulbücher gilt sie nicht, die haben ihre eigene Regel. Ohne bekannte Sommerferien bleibt die
// Frist, wie sie ist.
func AbgaengerFrist(frist time.Time, klasse string, kalender lmfplan.Ferientabelle, heute time.Time) time.Time {
	if !repository.IstAbschlussklasse(klasse) {
		return frist
	}
	ferien, _, ok := kalender.Naechste(heute, false)
	if !ok {
		return frist
	}
	tag := lmfplan.LetzterSchultagVor(ferien)
	grenze := schulzeit.TagesEnde(time.Date(tag.Year(), tag.Month(), tag.Day(), 12, 0, 0, 0, schulzeit.Zone()))
	if heuteEnde := schulzeit.TagesEnde(heute); grenze.Before(heuteEnde) {
		grenze = heuteEnde
	}
	if frist.After(grenze) {
		return grenze
	}
	return frist
}

// mehrjahresbandJahre zählt die Jahre, die ein Band über mehrere Jahrgänge über das laufende
// Schuljahr hinaus beim Kind bleibt: den Abstand zwischen seiner Klasse und JahrgangBis. Ein
// Kind über der Spanne bekommt ein Schuljahr wie jedes andere, ebenso eines in einer Klasse ohne
// Jahrgang. Den Jahrgang liest die Regel, nach der auch der Stichtag rechnet
// (repository.AbschlussJahrgang).
func mehrjahresbandJahre(exemplar *repository.Exemplar, klasse string) int {
	if !exemplar.Mehrjahresband || exemplar.JahrgangBis <= 0 {
		return 0
	}
	jahrgang, _, ok := repository.AbschlussJahrgang(klasse)
	if ok && exemplar.JahrgangBis >= jahrgang {
		return exemplar.JahrgangBis - jahrgang
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
//     verbleibenden Jahre, deshalb das größere von zusatzJahre und 1: Ein Kind der 7 mit einem
//     Band bis Jahrgang 9, ausgegeben im Büchertausch, gibt ihn am Ende der Klasse 9 zurück.
//
// Ein Fehler beim Nachschlagen hält die Ausleihe nicht an: Dann gilt der Stichtag.
func (d *AusleihDienst) ladeLMFPlanFrist(ctx context.Context, exemplar *repository.Exemplar, klasse string, heute time.Time, einst *ausleihEinstellungen, zusatzJahre int) (time.Time, bool) {
	if exemplar.IstLernmittel && klasse != "" {
		lage, lageErr := repository.NewLMFAbfragen(d.pool).LadeRueckgabeTerminLage(ctx, klasse, heute)
		if lageErr == nil && lage.TauschBegonnen {
			stichtag := repository.LMFStichtagNachJahren(heute, klasse, max(zusatzJahre, 1),
				lmfplan.FerientabelleAus(einst.Sommerferien))
			return schulzeit.TagesEnde(stichtag), true
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
	if exemplar.IstLernmittel {
		return time.Time{}, false
	}
	return leseclubDatum(einst, heute)
}

// leseclubDatum liefert das Zieldatum des Ferien-Leseclubs als Frist, solange er läuft und das
// Datum nicht vorbei ist. Es gilt für Bücher, Medien und Geräte, nicht für Lernmittel.
func leseclubDatum(einst *ausleihEinstellungen, heute time.Time) (time.Time, bool) {
	if einst.FerienLeseclubAktiv && einst.FerienLeseclubZieldatum != nil {
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
