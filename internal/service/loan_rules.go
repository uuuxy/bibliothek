package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"bibliothek/repository"
)

var (
	schoolLoc     *time.Location
	schoolLocOnce sync.Once
)

// schoolLocation liefert die feste Zeitzone der Schule (Europe/Berlin).
// Fristen wie der LMF-Stichtag oder das Leseclub-Zieldatum sind Kalendertage
// ("Ende des 31.07.") und müssen in der Schul-Zeitzone berechnet werden —
// sonst hängt das tatsächliche Ablaufdatum davon ab, in welcher Zeitzone der
// Server/Container läuft (im Docker-Image standardmäßig UTC). Fällt das Laden
// fehl (fehlende tzdata), wird sicher auf UTC zurückgegriffen.
func schoolLocation() *time.Location {
	schoolLocOnce.Do(func() {
		loc, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			loc = time.UTC
		}
		schoolLoc = loc
	})
	return schoolLoc
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
}

// querySettings liest die aktuellen Einstellungen aus der Datenbank aus und liefert
// bei Fehlern oder fehlenden Werten vordefinierte, sichere Standardwerte zurück.
func (s *defaultLoanService) querySettings(ctx context.Context) (*SystemEinstellungen, error) {
	// coalesce: eine einzige NULL-wert-Zeile (z. B. nie gesetztes
	// ferien_leseclub_zieldatum) ließe sonst den Scan in string scheitern —
	// pgx bricht dann die Iteration ab und rows.Err() macht JEDEN Checkout zum 500.
	rows, err := s.pool.Query(ctx, "SELECT schluessel, coalesce(wert, '') FROM system_einstellungen")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialisierung mit Standardwerten für den Fall, dass die Tabelle leer ist
	settings := &SystemEinstellungen{
		FristBuchTage:        21,
		FristMedienTage:      7,
		MaxAusleihenSchueler: 5,
		LmfStichtag:          "07-31",
		FerienLeseclubAktiv:  false,
		MaxOverdueDays:       14,
		MaxOverdueItems:      1,
	}

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
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
		}
	}
	// Ein mittendrin abgebrochener Query würde sonst stillschweigend die Defaults
	// liefern, statt den Fehler sichtbar zu machen — heikel, weil die Werte direkt
	// die Leihfristen und Sperr-Schwellen bestimmen.
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return settings, nil
}

// DueDateConfig bündelt die Parameter zur Berechnung des Rückgabedatums.
type DueDateConfig struct {
	Titel           string
	Medientyp       string
	LmfStichtag     string
	FristBuchTage   int
	FristMedienTage int
	AdditionalYears int
}

// calculateDueDate berechnet das Rückgabedatum auf Basis von Titel, Medientyp und
// den definierten Standardfristen.
func calculateDueDate(cfg DueDateConfig) time.Time {
	// In Schul-Zeitzone rechnen, damit sowohl der Jahreswechsel-Stichtag (August)
	// als auch das "Ende des Tages" (23:59:59) deterministisch sind — unabhängig
	// von der Server-Zeitzone. now.Location() ist dadurch schoolLocation().
	now := time.Now().In(schoolLocation())

	// 1. Fall: Lernmittelfreiheit (Schulbücher)
	// Schulbücher (erkennbar am Präfix "lmf-" oder "LMF-") werden für das gesamte Schuljahr ausgeliehen.
	// Sie müssen spätestens am definierten Stichtag (standardmäßig 31. Juli) zurückgegeben werden.
	if strings.HasPrefix(strings.ToLower(cfg.Titel), "lmf-") {
		year := now.Year()
		// Wenn wir uns bereits im oder nach dem August befinden (neues Schuljahr),
		// liegt der Stichtag im nächsten Kalenderjahr.
		if now.Month() >= time.August {
			year++
		}

		// Mehrjährige Ausleihen
		year += cfg.AdditionalYears

		month := time.July
		day := 31

		// Stichtag aus den Einstellungen parsen (Format: MM-DD, z.B. "07-31")
		parts := strings.SplitN(cfg.LmfStichtag, "-", 2)
		if len(parts) == 2 {
			m, err1 := strconv.Atoi(parts[0])
			d, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && m >= 1 && m <= 12 && d >= 1 && d <= 31 {
				month = time.Month(m)
				day = d
			}
		}
		// Rückgabezeitpunkt auf das Ende des Stichtags (23:59:59 Uhr) setzen
		return time.Date(year, month, day, 23, 59, 59, 0, now.Location())
	}

	// 2. Fall: Audiovisuelle/Digitale Medien
	// Medien wie CDs, DVDs oder Audio-Dateien haben aufgrund der höheren Nachfrage
	// eine verkürzte Ausleihfrist (fristMedienTage).
	lower := strings.ToLower(cfg.Medientyp)
	if strings.Contains(lower, "cd") || strings.Contains(lower, "dvd") || strings.Contains(lower, "audio") {
		return now.AddDate(0, 0, cfg.FristMedienTage)
	}

	// 3. Fall: Reguläre Bücher
	// Standardleihfrist für normale Buchbestände (fristBuchTage).
	return now.AddDate(0, 0, cfg.FristBuchTage)
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
func (s *defaultLoanService) resolveCheckoutDueDate(ctx context.Context, copy *repository.BookCopy, borrowerKlasse string) (time.Time, error) {
	settings, err := s.querySettings(ctx)

	additionalYears := 0
	if copy.ZielJahrgang > 0 && borrowerKlasse != "" {
		currentGrade := parseGrade(borrowerKlasse)
		if currentGrade > 0 && copy.ZielJahrgang >= currentGrade {
			additionalYears = copy.ZielJahrgang - currentGrade
		}
	}

	if err != nil {
		// Bei einem Datenbankfehler greifen wir auf feste Notfall-Standardwerte zurück
		return calculateDueDate(DueDateConfig{
			Titel:           copy.Titel,
			Medientyp:       copy.Medientyp,
			LmfStichtag:     "07-31",
			FristBuchTage:   21,
			FristMedienTage: 7,
			AdditionalYears: additionalYears,
		}), nil
	}

	isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")

	// Leseclub-Regel: Falls die Ferien-Leseclub-Aktion aktiv ist und ein Zieldatum konfiguriert wurde,
	// erhalten alle regulären Buchbestände (ausgenommen LMF-Schulbücher) dieses Zieldatum als Frist.
	if !isLMF && settings.FerienLeseclubAktiv && settings.FerienLeseclubZieldatum != nil {
		t, parseErr := time.Parse("2006-01-02", *settings.FerienLeseclubZieldatum)
		if parseErr == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, schoolLocation())
			return end, nil
		}
	}

	// Reguläre Fristenberechnung
	return calculateDueDate(DueDateConfig{
		Titel:           copy.Titel,
		Medientyp:       copy.Medientyp,
		LmfStichtag:     settings.LmfStichtag,
		FristBuchTage:   settings.FristBuchTage,
		FristMedienTage: settings.FristMedienTage,
		AdditionalYears: additionalYears,
	}), nil
}
