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
// (23:59:59) in der Schul-Zeitzone (Europe/Berlin). Dies ist die EINZIGE Definition von
// "Ende des Tages" im System: JEDE Rückgabefrist (reguläre Bücher, Medien, LMF-Stichtag,
// Geräte, Handapparat/Lehrer-Dauerleihe) läuft hierüber — seit 19.08.2026 auch die
// manuelle Frist-Überschreibung und die LMF-Massenverlängerung im api-Paket, die zuvor
// roh 23:59:59 UTC setzten und damit fristabhängig 1–2 h daneben lagen; seit 24.09.2026 über
// Tagesfrist auch die Einzel-Verlängerung, die bis dahin in SQL ab CURRENT_TIMESTAMP rechnete
// und bei einer überfälligen Ausleihe die Uhrzeit der Verlängerung behielt. Damit fällt die Fälligkeit immer
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

// querySettings liest die aktuellen Einstellungen aus der Datenbank aus und liefert
// bei Fehlern oder fehlenden Werten vordefinierte, sichere Standardwerte zurück.
func (s *defaultLoanService) querySettings(ctx context.Context) (*SystemEinstellungen, error) {
	return ladeSystemEinstellungen(ctx, s.pool)
}

// ladeSystemEinstellungen liest die Systemeinstellungen über q — den Pool oder eine
// Transaktion. Geteilt zwischen den Fristen (querySettings) und der Überfällig-Automatik
// (pruefeAusleihSperren), damit Fristen und Sperr-Schwellen aus EINER Quelle kommen.
func ladeSystemEinstellungen(ctx context.Context, pool repository.DBQueryer) (*SystemEinstellungen, error) {
	// coalesce: eine einzige NULL-wert-Zeile (z. B. nie gesetztes
	// ferien_leseclub_zieldatum) ließe sonst den Scan in string scheitern —
	// pgx bricht dann die Iteration ab und rows.Err() macht JEDEN Checkout zum 500.
	rows, err := pool.Query(ctx, "SELECT schluessel, coalesce(wert, '') FROM system_einstellungen")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialisierung mit Standardwerten für den Fall, dass die Tabelle leer ist
	settings := &SystemEinstellungen{
		FristBuchTage:        21,
		FristMedienTage:      7,
		MaxAusleihenSchueler: 5,
		LmfStichtag:          repository.StandardLmfStichtag,
		FerienLeseclubAktiv:  false,
		MaxOverdueDays:       14,
		MaxOverdueItems:      1,
	}

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		applyEinstellung(settings, key, value)
	}
	// Ein mittendrin abgebrochener Query würde sonst stillschweigend die Defaults
	// liefern, statt den Fehler sichtbar zu machen — heikel, weil die Werte direkt
	// die Leihfristen und Sperr-Schwellen bestimmen.
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return settings, nil
}

// applyEinstellung überträgt einen einzelnen Schlüssel/Wert aus system_einstellungen
// in die Settings-Struktur; unbekannte Schlüssel und ungültige Zahlen werden ignoriert.
func applyEinstellung(settings *SystemEinstellungen, key, value string) {
	switch key {
	case "frist_buch_tage":
		if v, err := strconv.Atoi(value); err == nil {
			settings.FristBuchTage = v
		}
	case "frist_medien_tage":
		if v, err := strconv.Atoi(value); err == nil {
			settings.FristMedienTage = v
		}
	case "max_ausleihen_schueler":
		if v, err := strconv.Atoi(value); err == nil {
			settings.MaxAusleihenSchueler = v
		}
	case "lmf_stichtag":
		settings.LmfStichtag = value
	case "ferien_leseclub_aktiv":
		settings.FerienLeseclubAktiv = (value == "true")
	case "ferien_leseclub_zieldatum":
		if value != "" {
			val := value
			settings.FerienLeseclubZieldatum = &val
		}
	case "max_overdue_days":
		if v, err := strconv.Atoi(value); err == nil {
			settings.MaxOverdueDays = v
		}
	case "max_overdue_items":
		if v, err := strconv.Atoi(value); err == nil {
			settings.MaxOverdueItems = v
		}
	case lmfplan.SommerferienSchluessel:
		settings.Sommerferien = value
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
		return Tagesfrist(now, opts.FristMedienTage, lmfplan.FerientabelleAus(opts.Sommerferien))
	}

	// 3. Fall: Reguläre Bücher
	// Standardleihfrist für normale Buchbestände (fristBuchTage).
	return Tagesfrist(now, opts.FristBuchTage, lmfplan.FerientabelleAus(opts.Sommerferien))
}

// Tagesfrist ist die Frist einer Leihe, die in Tagen zählt — Buch, Medium, Gerät,
// Verlängerung: ab plus tage, und fällt dieser Tag auf ein Wochenende, einen Feiertag oder
// in die Ferien, der nächste Schultag; Tagesende in der Schulzeitzone.
//
// Entscheidung vom 24.09.2026: Wer vor den Herbstferien ausleiht, soll nicht gemahnt
// werden, weil die Frist in die Ferien fiel. Nicht hierüber laufen Stichtage und von Hand
// gesetzte Fristen — Lernmittel (Stichtag, LMF-Plan), Ferien-Leseclub, Frist-Überschreibung,
// die Jahresfrist der Dauerleihe. Der LMF-Stichtag 31.07. liegt in jedem Jahr der Tabelle in
// den Sommerferien; über diese Regel stünde jedes Lernmittel am Tag der Bücherausgabe.
func Tagesfrist(ab time.Time, tage int, kalender lmfplan.Ferientabelle) time.Time {
	tag := kalender.NaechsterSchultag(ab.In(schoolLocation()).AddDate(0, 0, tage))
	return TagesEndeInSchulzeitzone(time.Date(tag.Year(), tag.Month(), tag.Day(), 12, 0, 0, 0, schoolLocation()))
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
// Scan-Zeitpunkt (Entscheidung vom 13.09.2026: Frist, Mahnwesen und Lesehistorie rechnen
// ab dem Scan), am Online-Scan jetzt.
func (s *defaultLoanService) resolveCheckoutDueDateAm(ctx context.Context, copy *repository.BookCopy, borrowerKlasse string, heute time.Time) (time.Time, error) {
	settings, err := s.querySettings(ctx)
	heute = heute.In(schoolLocation())

	// Mehrjahresband (Migration 134): Das Buch bleibt bis zum Ende von JahrgangBis beim Kind.
	// Die Jahre über das laufende Schuljahr hinaus sind der Abstand zwischen der Klasse des
	// Kindes und JahrgangBis; ein Kind über der Spanne bekommt ein Schuljahr wie jedes andere.
	additionalYears := 0
	if copy.Mehrjahresband && copy.JahrgangBis > 0 && borrowerKlasse != "" {
		currentGrade := parseGrade(borrowerKlasse)
		if currentGrade > 0 && copy.JahrgangBis >= currentGrade {
			additionalYears = copy.JahrgangBis - currentGrade
		}
	}

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

	// LMF-Plan (Register, Entscheidung 3a): Nennt der Plan für die Klasse einen
	// Rückgabe-Termin, ist der die Frist ihrer Schulbücher — vor dem globalen Stichtag.
	// Nur für die einjährige Ausleihe; eine mehrjährige rechnet weiter über den Stichtag.
	// Ein Fehler beim Nachschlagen blockiert die Ausleihe nicht: dann gilt der Stichtag.
	//
	// Mehrjahresband (Antwort der Schule vom 22.09.2026, docs/OFFEN.md 9.6): Der
	// Rückgabetermin der Klasse geht das Buch nichts an, es bleibt beim Kind — außer der
	// Termin ist schon vorbei. Dann ist das laufende Schuljahr für diese Klasse zu Ende, und
	// die Frist rechnet vom folgenden Schuljahr aus: Von den verbleibenden Jahren steckt
	// eines bereits in diesem Sprung, deshalb additionalYears-1. Ein Kind der 7 mit einem
	// Band bis Jahrgang 9, ausgegeben nach dem Termin im Juli 2027, gibt ihn am 31.07.2029
	// zurück — nicht 2030.
	//
	// Am oder nach dem Rückgabetermin der Klasse (Entscheidung vom 13.09.2026): Wer jetzt
	// noch ein Schulbuch bekommt, gibt es erst im nächsten Schuljahr zurück — die Frist ist
	// dessen Stichtag. Das gilt auch, wenn die Klasse noch einen Nachzügler-Termin vor sich
	// hat (zweimal im Plan): Nach ihrem Rückgabetermin ist jedes neu ausgegebene Schulbuch
	// eines fürs nächste Schuljahr, keins für fünf Tage. Bis zum 14.09.2026 war am Termintag
	// der Termin selbst die Frist (heute 23:59) und danach der Stichtag des laufenden
	// Schuljahres, ein Tag in den Ferien: Nach den Ferien wäre die ganze Klasse überfällig und
	// nach 14 Tagen gesperrt gewesen.
	if copy.IstLernmittel && borrowerKlasse != "" {
		lage, lageErr := repository.NewLmfTerminRepository(s.pool).RueckgabeTerminLage(ctx, borrowerKlasse, heute)
		if lageErr == nil && lage.Vergangen {
			folgendes := repository.SchuljahrBeginn(heute).AddDate(1, 0, 0)
			stichtag := repository.LmfStichtagImSchuljahr(folgendes, settings.LmfStichtag)
			return TagesEndeInSchulzeitzone(stichtag.AddDate(max(additionalYears-1, 0), 0, 0)), nil
		}
		if lageErr == nil && lage.Bevorstehend && additionalYears == 0 {
			return TagesEndeInSchulzeitzone(lage.Naechster), nil
		}
	}

	// Leseclub-Regel: Falls die Ferien-Leseclub-Aktion aktiv ist und ein Zieldatum konfiguriert wurde,
	// erhalten alle regulären Buchbestände (ausgenommen Lernmittel) dieses Zieldatum als Frist —
	// aber nur, solange es nicht vorbei ist. Bleibt der Schalter nach den Ferien an, wäre das
	// vergangene Ferienende sonst die Frist jeder neuen Ausleihe: sofort überfällig, nach 14
	// Tagen gesperrt (Bestands-Durchgang 10.09.2026). Dann gilt die reguläre Frist.
	if !copy.IstLernmittel && settings.FerienLeseclubAktiv && settings.FerienLeseclubZieldatum != nil {
		t, parseErr := time.Parse("2006-01-02", *settings.FerienLeseclubZieldatum)
		if parseErr == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, schoolLocation())
			if end.After(heute) {
				return end, nil
			}
		}
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
