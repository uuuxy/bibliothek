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

// fristAngaben fasst die Parameter für die Fristenberechnung zusammen,
// um die Anzahl der Funktionsargumente übersichtlich zu halten.
type fristAngaben struct {
	IstLernmittel   bool
	Medientyp       string
	LMFStichtag     string
	FristBuchTage   int
	FristMedienTage int
	ZusatzJahre     int
	// Sommerferien: Text der Einstellung „sommerferien" (ausleihEinstellungen.Sommerferien);
	// leer heißt Programmtabelle.
	Sommerferien string
}

// berechneFrist berechnet das Rückgabedatum auf Basis von Lernmittel-Kennzeichen,
// Medientyp und den definierten Standardfristen.
//
// jetzt ist die Uhr des Aufrufers (bestimmeFrist reicht die des Dienstes durch), damit
// die Frist um den Rückgabetermin mit fester Uhr prüfbar ist.
func berechneFrist(jetzt time.Time, angaben fristAngaben) time.Time {
	// In Schul-Zeitzone rechnen, damit sowohl der Jahreswechsel-Stichtag (August)
	// als auch das "Ende des Tages" (23:59:59) deterministisch sind — unabhängig
	// von der Server-Zeitzone. now.Location() ist dadurch schulzeit.Zone().
	ortszeit := jetzt.In(schulzeit.Zone())

	// 1. Fall: Lernmittelfreiheit (Schulbücher)
	// Schulbücher (buecher_titel.ist_lernmittel) werden für das gesamte Schuljahr
	// ausgeliehen. Sie müssen spätestens am definierten Stichtag (standardmäßig 31. Juli)
	// zurückgegeben werden.
	if angaben.IstLernmittel {
		// Der nächste Stichtag ab heute — DIE Rechnung, die auch der Rückweg des
		// LMF-Plans benutzt (repository.LMFStichtagAbTag). Bis zum 06.09.2026 stand sie
		// hier ein zweites Mal und rechnete „ab August ins nächste Kalenderjahr"; das
		// stimmt nur für einen Stichtag von Januar bis Juli. Ein Stichtag ab August
		// ergab hier den Termin des FOLGENDEN Schuljahres — dreizehn Monate Frist —,
		// während der Rückweg denselben Stichtag ein Jahr früher setzte.
		stichtag := repository.LMFStichtagAbTag(ortszeit, angaben.LMFStichtag)
		// Mehrjährige Ausleihen (Zieljahrgang über der Klasse des Entleihers) laufen
		// entsprechend viele Stichtage weiter.
		stichtag = stichtag.AddDate(angaben.ZusatzJahre, 0, 0)
		return schulzeit.TagesEnde(stichtag)
	}

	// 2. Fall: Audiovisuelle/Digitale Medien
	// Medien wie CDs, DVDs oder Audio-Dateien haben aufgrund der höheren Nachfrage
	// eine verkürzte Ausleihfrist (fristMedienTage).
	medientyp := strings.ToLower(angaben.Medientyp)
	if strings.Contains(medientyp, "cd") || strings.Contains(medientyp, "dvd") || strings.Contains(medientyp, "audio") {
		return lmfplan.FerientabelleAus(angaben.Sommerferien).Tagesfrist(ortszeit, angaben.FristMedienTage)
	}

	// 3. Fall: Reguläre Bücher
	// Standardleihfrist für normale Buchbestände (fristBuchTage).
	return lmfplan.FerientabelleAus(angaben.Sommerferien).Tagesfrist(ortszeit, angaben.FristBuchTage)
}

// leseJahrgang extrahiert den Jahrgang aus dem Klassen-String.
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
	// Fallback auf Extraktion der ersten Zahl
	zahlen := regexp.MustCompile(`\d+`)
	treffer := zahlen.FindString(gross)
	if treffer != "" {
		if n, err := strconv.Atoi(treffer); err == nil {
			return n
		}
	}
	return 0
}

// bestimmeFrist ermittelt das Fälligkeitsdatum für eine neue Buchausleihe.
// Hierbei werden Sonderaktionen wie der Ferien-Leseclub ausgewertet, um reguläre Leihfristen zu überschreiben.
// bestimmeFrist rechnet die Frist ab der Uhr des Dienstes (jetzt).
func (d *ausleihDienst) bestimmeFrist(ctx context.Context, exemplar *repository.Exemplar, klasse string) (time.Time, error) {
	return d.bestimmeFristAm(ctx, exemplar, klasse, d.heute())
}

// bestimmeFristAm rechnet die Frist ab einem gegebenen Tag — beim Nachbuchen der
// Scan-Zeitpunkt (Frist, Mahnwesen und Lesehistorie rechnen ab dem Scan), am Online-Scan jetzt.
func (d *ausleihDienst) bestimmeFristAm(ctx context.Context, exemplar *repository.Exemplar, klasse string, heute time.Time) (time.Time, error) {
	einst, err := d.ladeEinstellungen(ctx)
	heute = heute.In(schulzeit.Zone())
	zusatzJahre := mehrjahresbandJahre(exemplar, klasse)

	if err != nil {
		// Bei einem Datenbankfehler greifen wir auf feste Notfall-Standardwerte zurück
		return berechneFrist(heute, fristAngaben{
			IstLernmittel:   exemplar.IstLernmittel,
			Medientyp:       exemplar.Medientyp,
			LMFStichtag:     repository.LMFStichtagVorgabe,
			FristBuchTage:   21,
			FristMedienTage: 7,
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

	// Reguläre Fristenberechnung
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

// mehrjahresbandJahre: Ein Mehrjahresband (Migration 134) bleibt bis zum Ende von JahrgangBis
// beim Kind. Die Jahre über das laufende Schuljahr hinaus sind der Abstand zwischen der Klasse
// des Kindes und JahrgangBis; ein Kind über der Spanne bekommt ein Schuljahr wie jedes andere.
func mehrjahresbandJahre(exemplar *repository.Exemplar, klasse string) int {
	if exemplar.Mehrjahresband && exemplar.JahrgangBis > 0 && klasse != "" {
		jahrgang := leseJahrgang(klasse)
		if jahrgang > 0 && exemplar.JahrgangBis >= jahrgang {
			return exemplar.JahrgangBis - jahrgang
		}
	}
	return 0
}

// ladeLMFPlanFrist liefert die Frist eines Schulbuchs, wenn der LMF-Plan der Klasse sie bestimmt:
//
//   - Steht der Rückgabetermin der Klasse bevor, ist er die Frist — vor dem globalen Stichtag,
//     und nur für die einjährige Ausleihe. Ein Mehrjahresband bleibt beim Kind und rechnet
//     weiter über den Stichtag.
//   - Am oder nach dem Rückgabetermin gibt die Klasse ein neu ausgegebenes Schulbuch erst im
//     nächsten Schuljahr zurück, die Frist ist dessen Stichtag — auch wenn noch ein
//     Nachzügler-Termin folgt. Mit dem Stichtag des laufenden Schuljahres wäre die Klasse nach
//     den Ferien überfällig und nach 14 Tagen gesperrt.
//   - Beim Mehrjahresband steckt in diesem Sprung schon eines der verbleibenden Jahre, deshalb
//     additionalYears-1: Ein Kind der 7 mit einem Band bis Jahrgang 9, ausgegeben nach dem
//     Termin im Juli 2027, gibt ihn am 31.07.2029 zurück.
//
// Ein Fehler beim Nachschlagen hält die Ausleihe nicht an: Dann gilt der Stichtag.
func (d *ausleihDienst) ladeLMFPlanFrist(ctx context.Context, exemplar *repository.Exemplar, klasse string, heute time.Time, einst *ausleihEinstellungen, zusatzJahre int) (time.Time, bool) {
	if exemplar.IstLernmittel && klasse != "" {
		lage, lageErr := repository.NewLMFAbfragen(d.pool).LadeRueckgabeTerminLage(ctx, klasse, heute)
		if lageErr == nil && lage.Vergangen {
			folgendes := repository.SchuljahrBeginn(heute).AddDate(1, 0, 0)
			stichtag := repository.LMFStichtagImSchuljahr(folgendes, einst.LMFStichtag)
			return schulzeit.TagesEnde(stichtag.AddDate(max(zusatzJahre-1, 0), 0, 0)), true
		}
		if lageErr == nil && lage.Bevorstehend && zusatzJahre == 0 {
			return schulzeit.TagesEnde(lage.Naechster), true
		}
	}
	return time.Time{}, false
}

// leseclubFrist: Ist die Ferien-Leseclub-Aktion aktiv und ein Zieldatum eingestellt, ist es die
// Frist aller regulären Bestände (ausgenommen Lernmittel) — aber nur, solange es nicht vorbei
// ist. Bleibt der Schalter nach den Ferien an, wäre das vergangene Ferienende sonst die Frist
// jeder neuen Ausleihe: sofort überfällig, nach 14 Tagen gesperrt. Dann gilt die reguläre Frist.
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
