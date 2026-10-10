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

// schoolLocation liefert die feste Zeitzone der Schule (Europe/Berlin).
// Fristen wie der LMF-Stichtag oder das Leseclub-Zieldatum sind Kalendertage
// ("Ende des 31.07.") und müssen in der Schul-Zeitzone berechnet werden —
// sonst hängt das tatsächliche Ablaufdatum davon ab, in welcher Zeitzone der
// Server/Container läuft (im Docker-Image standardmäßig UTC). Fällt das Laden
// fehl (fehlende tzdata), wird sicher auf UTC zurückgegriffen.
// Seit 23.08.2026 liegt die Zeitzone selbst in pkg/schulzeit — einem Blatt-Paket ohne
// Abhängigkeiten. Grund: Die gedruckten Dokumente (pdf/) brauchen dieselbe Zeitzone,
// können aber unmöglich das ganze service-Paket samt pgx und repository importieren.
// Diese Funktion bleibt als Name stehen, weil sie hier vierfach gerufen wird.
func schoolLocation() *time.Location {
	return schulzeit.Zone()
}

// TagesEndeInSchulzeitzone normalisiert einen Zeitpunkt auf das Ende seines Kalendertags
// (23:59:59) in der Schul-Zeitzone (Europe/Berlin). Die Definition dahinter,
// schulzeit.TagesEnde, ist die EINZIGE von "Ende des Tages" im System: JEDE Rückgabefrist
// (reguläre Bücher, Medien, LMF-Stichtag, Geräte, Handapparat/Lehrer-Dauerleihe) läuft
// darüber — seit 19.08.2026 auch die manuelle Frist-Überschreibung und die
// LMF-Massenverlängerung im api-Paket, die zuvor roh 23:59:59 UTC setzten und damit
// fristabhängig 1–2 h daneben lagen; seit 24.09.2026 über die Tagesfrist
// (lmfplan.Ferientabelle.Tagesfrist) auch die Einzel-Verlängerung, die bis dahin in SQL ab
// CURRENT_TIMESTAMP rechnete und bei einer überfälligen Ausleihe die Uhrzeit der
// Verlängerung behielt. Damit fällt die Fälligkeit immer
// deterministisch auf den Kalendertag — unabhängig von der Server-Zeitzone (Docker = UTC) —
// und es gibt keine zweite, rohe Berechnungsmethode mehr, die bei künftigen Änderungen
// (z. B. kürzere Handapparat-Frist) auf die Füße fällt.
func TagesEndeInSchulzeitzone(t time.Time) time.Time {
	return schulzeit.TagesEnde(t)
}

// SystemEinstellungen repräsentiert die Konfigurationsparameter des Ausleihsystems,
// die in der Datenbanktabelle `system_einstellungen` verwaltet werden.
type SystemEinstellungen struct {
	// FristBuchTage definiert die Standardleihfrist für normale Bücher in Tagen.
	FristBuchTage int
	// FristMedienTage definiert die Leihfrist für Sonder-Medien (CDs, DVDs, Hörbücher) in Tagen.
	FristMedienTage int
	// MaxAusleihenSchueler begrenzt die Anzahl der gleichzeitig ausgeliehenen regulären Bücher pro Schüler.
	MaxAusleihenSchueler int
	// LmfStichtag bestimmt den jährlichen Rückgabetermin für Schulbücher der Lernmittelfreiheit (z. B. "07-31").
	LmfStichtag string
	// FerienLeseclubAktiv ist wahr, wenn die verlängerte Ausleihe für den Ferien-Leseclub aktiv ist.
	FerienLeseclubAktiv bool
	// FerienLeseclubZieldatum definiert das feste Rückgabedatum für alle Leseclub-Ausleihen.
	FerienLeseclubZieldatum *string
	// MaxOverdueDays: Anzahl der Toleranztage bevor ein Schüler blockiert wird.
	MaxOverdueDays int
	// MaxOverdueItems: Anzahl der überfälligen Medien ab denen blockiert wird.
	MaxOverdueItems int
	// Sommerferien ist der Text der Einstellung „sommerferien" — die Jahre, die die Schule
	// neben der Programmtabelle eingetragen hat. Leer heißt: nur die Programmtabelle.
	Sommerferien string
}

// querySettings liest über den Pool, was die Ausleihe aus den Einstellungen braucht.
func (s *defaultLoanService) querySettings(ctx context.Context) (*SystemEinstellungen, error) {
	return ladeSystemEinstellungen(ctx, s.pool)
}

// ladeSystemEinstellungen liest über q, den Pool oder eine Transaktion, was die Ausleihe aus
// den Einstellungen braucht: Fristen und Sperr-Schwellen. Gelesen und abgebildet werden die
// Zeilen an einer Stelle (repository.EinstellungenUeber); ein fehlender oder unlesbarer Wert
// hat so an der Theke dieselbe Vorgabe wie in der Maske der Einstellungen.
func ladeSystemEinstellungen(ctx context.Context, q repository.DBQueryer) (*SystemEinstellungen, error) {
	einstellungen, err := repository.EinstellungenUeber(ctx, q)
	if err != nil {
		return nil, err
	}
	return ausleihEinstellungenAus(einstellungen), nil
}

// ausleihEinstellungenAus wählt aus den Einstellungen die der Ausleihe.
func ausleihEinstellungenAus(e *repository.SystemEinstellungen) *SystemEinstellungen {
	return &SystemEinstellungen{
		FristBuchTage:           e.FristBuchTage,
		FristMedienTage:         e.FristMedienTage,
		MaxAusleihenSchueler:    e.MaxAusleihenSchueler,
		LmfStichtag:             e.LmfStichtag,
		FerienLeseclubAktiv:     e.FerienLeseclubAktiv,
		FerienLeseclubZieldatum: e.FerienLeseclubZieldatum,
		MaxOverdueDays:          e.MaxOverdueDays,
		MaxOverdueItems:         e.MaxOverdueItems,
		Sommerferien:            e.Sommerferien,
	}
}

// DueDateOptions fasst die Parameter für die Fristenberechnung zusammen,
// um die Anzahl der Funktionsargumente übersichtlich zu halten.
type DueDateOptions struct {
	IstLernmittel   bool
	Medientyp       string
	LmfStichtag     string
	FristBuchTage   int
	FristMedienTage int
	AdditionalYears int
	// Sommerferien: Text der Einstellung „sommerferien" (SystemEinstellungen.Sommerferien);
	// leer heißt Programmtabelle.
	Sommerferien string
}

// calculateDueDate berechnet das Rückgabedatum auf Basis von Lernmittel-Kennzeichen,
// Medientyp und den definierten Standardfristen.
//
// jetzt ist die Uhr des Aufrufers (resolveCheckoutDueDate reicht die des Dienstes durch), damit
// die Frist um den Rückgabetermin mit fester Uhr prüfbar ist.
func calculateDueDate(jetzt time.Time, opts DueDateOptions) time.Time {
	// In Schul-Zeitzone rechnen, damit sowohl der Jahreswechsel-Stichtag (August)
	// als auch das "Ende des Tages" (23:59:59) deterministisch sind — unabhängig
	// von der Server-Zeitzone. now.Location() ist dadurch schoolLocation().
	now := jetzt.In(schoolLocation())

	// 1. Fall: Lernmittelfreiheit (Schulbücher)
	// Schulbücher (buecher_titel.ist_lernmittel) werden für das gesamte Schuljahr
	// ausgeliehen. Sie müssen spätestens am definierten Stichtag (standardmäßig 31. Juli)
	// zurückgegeben werden.
	if opts.IstLernmittel {
		// Der nächste Stichtag ab heute — DIE Rechnung, die auch der Rückweg des
		// LMF-Plans benutzt (repository.LmfStichtagAbTag). Bis zum 06.09.2026 stand sie
		// hier ein zweites Mal und rechnete „ab August ins nächste Kalenderjahr"; das
		// stimmt nur für einen Stichtag von Januar bis Juli. Ein Stichtag ab August
		// ergab hier den Termin des FOLGENDEN Schuljahres — dreizehn Monate Frist —,
		// während der Rückweg denselben Stichtag ein Jahr früher setzte.
		stichtag := repository.LmfStichtagAbTag(now, opts.LmfStichtag)
		// Mehrjährige Ausleihen (Zieljahrgang über der Klasse des Entleihers) laufen
		// entsprechend viele Stichtage weiter.
		stichtag = stichtag.AddDate(opts.AdditionalYears, 0, 0)
		return TagesEndeInSchulzeitzone(stichtag)
	}

	// 2. Fall: Audiovisuelle/Digitale Medien
	// Medien wie CDs, DVDs oder Audio-Dateien haben aufgrund der höheren Nachfrage
	// eine verkürzte Ausleihfrist (fristMedienTage).
	lower := strings.ToLower(opts.Medientyp)
	if strings.Contains(lower, "cd") || strings.Contains(lower, "dvd") || strings.Contains(lower, "audio") {
		return lmfplan.FerientabelleAus(opts.Sommerferien).Tagesfrist(now, opts.FristMedienTage)
	}

	// 3. Fall: Reguläre Bücher
	// Standardleihfrist für normale Buchbestände (fristBuchTage).
	return lmfplan.FerientabelleAus(opts.Sommerferien).Tagesfrist(now, opts.FristBuchTage)
}

// parseGrade extrahiert den Jahrgang aus dem Klassen-String.
func parseGrade(klasse string) int {
	upper := strings.ToUpper(strings.TrimSpace(klasse))
	if strings.HasPrefix(upper, "E") || upper == "EF" {
		return 11
	}
	if strings.HasPrefix(upper, "Q1") || strings.HasPrefix(upper, "Q2") {
		return 12
	}
	if strings.HasPrefix(upper, "Q3") || strings.HasPrefix(upper, "Q4") {
		return 13
	}
	// Fallback auf Extraktion der ersten Zahl
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(upper)
	if match != "" {
		if val, err := strconv.Atoi(match); err == nil {
			return val
		}
	}
	return 0
}

// resolveCheckoutDueDate ermittelt das Fälligkeitsdatum für eine neue Buchausleihe.
// Hierbei werden Sonderaktionen wie der Ferien-Leseclub ausgewertet, um reguläre Leihfristen zu überschreiben.
// resolveCheckoutDueDate rechnet die Frist ab der Uhr des Dienstes (jetzt).
func (s *defaultLoanService) resolveCheckoutDueDate(ctx context.Context, copy *repository.BookCopy, borrowerKlasse string) (time.Time, error) {
	return s.resolveCheckoutDueDateAm(ctx, copy, borrowerKlasse, s.heute())
}

// resolveCheckoutDueDateAm rechnet die Frist ab einem gegebenen Tag — beim Nachbuchen der
// Scan-Zeitpunkt (Frist, Mahnwesen und Lesehistorie rechnen ab dem Scan), am Online-Scan jetzt.
func (s *defaultLoanService) resolveCheckoutDueDateAm(ctx context.Context, copy *repository.BookCopy, borrowerKlasse string, heute time.Time) (time.Time, error) {
	settings, err := s.querySettings(ctx)
	heute = heute.In(schoolLocation())
	additionalYears := mehrjahresbandJahre(copy, borrowerKlasse)

	if err != nil {
		// Bei einem Datenbankfehler greifen wir auf feste Notfall-Standardwerte zurück
		return calculateDueDate(heute, DueDateOptions{
			IstLernmittel:   copy.IstLernmittel,
			Medientyp:       copy.Medientyp,
			LmfStichtag:     repository.StandardLmfStichtag,
			FristBuchTage:   21,
			FristMedienTage: 7,
			AdditionalYears: additionalYears,
			Sommerferien:    "",
		}), nil
	}

	if frist, ok := s.fristNachLmfPlan(ctx, copy, borrowerKlasse, heute, settings, additionalYears); ok {
		return frist, nil
	}
	if frist, ok := leseclubFrist(copy, settings, heute); ok {
		return frist, nil
	}

	// Reguläre Fristenberechnung
	return calculateDueDate(heute, DueDateOptions{
		IstLernmittel:   copy.IstLernmittel,
		Medientyp:       copy.Medientyp,
		LmfStichtag:     settings.LmfStichtag,
		FristBuchTage:   settings.FristBuchTage,
		FristMedienTage: settings.FristMedienTage,
		AdditionalYears: additionalYears,
		Sommerferien:    settings.Sommerferien,
	}), nil
}

// mehrjahresbandJahre: Ein Mehrjahresband (Migration 134) bleibt bis zum Ende von JahrgangBis
// beim Kind. Die Jahre über das laufende Schuljahr hinaus sind der Abstand zwischen der Klasse
// des Kindes und JahrgangBis; ein Kind über der Spanne bekommt ein Schuljahr wie jedes andere.
func mehrjahresbandJahre(copy *repository.BookCopy, borrowerKlasse string) int {
	if copy.Mehrjahresband && copy.JahrgangBis > 0 && borrowerKlasse != "" {
		currentGrade := parseGrade(borrowerKlasse)
		if currentGrade > 0 && copy.JahrgangBis >= currentGrade {
			return copy.JahrgangBis - currentGrade
		}
	}
	return 0
}

// fristNachLmfPlan liefert die Frist eines Schulbuchs, wenn der LMF-Plan der Klasse sie bestimmt:
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
func (s *defaultLoanService) fristNachLmfPlan(ctx context.Context, copy *repository.BookCopy, borrowerKlasse string, heute time.Time, settings *SystemEinstellungen, additionalYears int) (time.Time, bool) {
	if copy.IstLernmittel && borrowerKlasse != "" {
		lage, lageErr := repository.NewLmfTerminRepository(s.pool).RueckgabeTerminLage(ctx, borrowerKlasse, heute)
		if lageErr == nil && lage.Vergangen {
			folgendes := repository.SchuljahrBeginn(heute).AddDate(1, 0, 0)
			stichtag := repository.LmfStichtagImSchuljahr(folgendes, settings.LmfStichtag)
			return TagesEndeInSchulzeitzone(stichtag.AddDate(max(additionalYears-1, 0), 0, 0)), true
		}
		if lageErr == nil && lage.Bevorstehend && additionalYears == 0 {
			return TagesEndeInSchulzeitzone(lage.Naechster), true
		}
	}
	return time.Time{}, false
}

// leseclubFrist: Ist die Ferien-Leseclub-Aktion aktiv und ein Zieldatum eingestellt, ist es die
// Frist aller regulären Bestände (ausgenommen Lernmittel) — aber nur, solange es nicht vorbei
// ist. Bleibt der Schalter nach den Ferien an, wäre das vergangene Ferienende sonst die Frist
// jeder neuen Ausleihe: sofort überfällig, nach 14 Tagen gesperrt. Dann gilt die reguläre Frist.
func leseclubFrist(copy *repository.BookCopy, settings *SystemEinstellungen, heute time.Time) (time.Time, bool) {
	if !copy.IstLernmittel && settings.FerienLeseclubAktiv && settings.FerienLeseclubZieldatum != nil {
		t, parseErr := time.Parse("2006-01-02", *settings.FerienLeseclubZieldatum)
		if parseErr == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, schoolLocation())
			if end.After(heute) {
				return end, true
			}
		}
	}
	return time.Time{}, false
}
