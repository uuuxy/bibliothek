package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"bibliothek/repository"
)

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
	rows, err := s.pool.Query(ctx, "SELECT schluessel, wert FROM system_einstellungen")
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
	return settings, nil
}

// calculateDueDate berechnet das Rückgabedatum auf Basis von Titel, Medientyp und
// den definierten Standardfristen.
func calculateDueDate(titel, medientyp, lmfStichtag string, fristBuchTage, fristMedienTage int) time.Time {
	now := time.Now()

	// 1. Fall: Lernmittelfreiheit (Schulbücher)
	// Schulbücher (erkennbar am Präfix "lmf-" oder "LMF-") werden für das gesamte Schuljahr ausgeliehen.
	// Sie müssen spätestens am definierten Stichtag (standardmäßig 31. Juli) zurückgegeben werden.
	if strings.HasPrefix(strings.ToLower(titel), "lmf-") {
		year := now.Year()
		// Wenn wir uns bereits im oder nach dem August befinden (neues Schuljahr),
		// liegt der Stichtag im nächsten Kalenderjahr.
		if now.Month() >= time.August {
			year++
		}
		month := time.July
		day := 31

		// Stichtag aus den Einstellungen parsen (Format: MM-DD, z.B. "07-31")
		parts := strings.SplitN(lmfStichtag, "-", 2)
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
	lower := strings.ToLower(medientyp)
	if strings.Contains(lower, "cd") || strings.Contains(lower, "dvd") || strings.Contains(lower, "audio") {
		return now.AddDate(0, 0, fristMedienTage)
	}

	// 3. Fall: Reguläre Bücher
	// Standardleihfrist für normale Buchbestände (fristBuchTage).
	return now.AddDate(0, 0, fristBuchTage)
}

// resolveCheckoutDueDate ermittelt das Fälligkeitsdatum für eine neue Buchausleihe.
// Hierbei werden Sonderaktionen wie der Ferien-Leseclub ausgewertet, um reguläre Leihfristen zu überschreiben.
func (s *defaultLoanService) resolveCheckoutDueDate(ctx context.Context, copy *repository.BookCopy) (time.Time, error) {
	settings, err := s.querySettings(ctx)
	if err != nil {
		// Bei einem Datenbankfehler greifen wir auf feste Notfall-Standardwerte zurück
		return calculateDueDate(copy.Titel, copy.Medientyp, "07-31", 21, 7), nil
	}

	isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")

	// Leseclub-Regel: Falls die Ferien-Leseclub-Aktion aktiv ist und ein Zieldatum konfiguriert wurde,
	// erhalten alle regulären Buchbestände (ausgenommen LMF-Schulbücher) dieses Zieldatum als Frist.
	if !isLMF && settings.FerienLeseclubAktiv && settings.FerienLeseclubZieldatum != nil {
		t, parseErr := time.Parse("2006-01-02", *settings.FerienLeseclubZieldatum)
		if parseErr == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
			return end, nil
		}
	}

	// Reguläre Fristenberechnung
	return calculateDueDate(copy.Titel, copy.Medientyp, settings.LmfStichtag, settings.FristBuchTage, settings.FristMedienTage), nil
}
