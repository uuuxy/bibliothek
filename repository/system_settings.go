package repository

import (
	"bibliothek/db"
	"context"
	"strconv"
)

// SystemEinstellungen holds configurable system-wide settings.
type SystemEinstellungen struct {
	FerienLeseclubAktiv     bool    `json:"ferien_leseclub_aktiv"`
	FerienLeseclubZieldatum *string `json:"ferien_leseclub_zieldatum"` // ISO date string "YYYY-MM-DD" or null
	LmfStichtag             string  `json:"lmf_stichtag"`              // "MM-DD" format, e.g. "07-31"
	MaxAusleihenSchueler    int     `json:"max_ausleihen_schueler"`
	FristBuchTage           int     `json:"frist_buch_tage"`
	FristMedienTage         int     `json:"frist_medien_tage"`
	MaxOverdueDays          int     `json:"max_overdue_days"`
	MaxOverdueItems         int     `json:"max_overdue_items"`
	// Bestellbedarf: ob überhaupt gewarnt wird und ab welcher Exemplarzahl ein
	// (LMF-)Titel als Bestellbedarf gilt (gesamt < Schwelle). Löst den früheren
	// pauschalen Meldebestand-Default 5 ab, der fast jeden Titel fälschlich meldete.
	BestellbedarfWarnungAktiv bool `json:"bestellbedarf_warnung_aktiv"`
	BestellbedarfSchwelle     int  `json:"bestellbedarf_schwelle"`
	// School identity — used in PDF letter headers (set once via settings UI).
	SchuleName    string `json:"schule_name"`
	SchuleStrasse string `json:"schule_strasse"`
	SchulePLZ     string `json:"schule_plz"`
	SchuleOrt     string `json:"schule_ort"`
}

// SystemSettingsRepository defines operations for managing global system settings.
type SystemSettingsRepository interface {
	GetSettings(ctx context.Context) (*SystemEinstellungen, error)
	SaveSettings(ctx context.Context, settings *SystemEinstellungen) error
}

type pgSystemSettingsRepository struct {
	db db.PgxPoolIface
}

// NewSystemSettingsRepository returns a PostgreSQL implementation of SystemSettingsRepository.
func NewSystemSettingsRepository(db db.PgxPoolIface) SystemSettingsRepository {
	return &pgSystemSettingsRepository{db: db}
}

// standardEinstellungen liefert die Default-Konfiguration (Fallback, wenn ein
// Wert nicht in der DB steht).
func standardEinstellungen() *SystemEinstellungen {
	return &SystemEinstellungen{
		LmfStichtag:          "07-31",
		MaxAusleihenSchueler: 5,
		FristBuchTage:        21,
		FristMedienTage:      7,
		MaxOverdueDays:       14,
		MaxOverdueItems:      1,
		// Warnung standardmäßig an; Schwelle 3 (statt des früheren Default 5) als
		// ruhigerer Startwert — der Betreiber justiert sie in den Einstellungen.
		BestellbedarfWarnungAktiv: true,
		BestellbedarfSchwelle:     3,
	}
}

func setzeIntEinstellung(val *string, ziel *int) {
	if val == nil || *val == "" {
		return
	}
	if v, err := strconv.Atoi(*val); err == nil {
		*ziel = v
	}
}

func setzeStringNichtLeer(val *string, ziel *string) {
	if val != nil && *val != "" {
		*ziel = *val
	}
}

func setzeStringRoh(val *string, ziel *string) {
	if val != nil {
		*ziel = *val
	}
}

// applyEinstellung überträgt einen einzelnen Key/Value-Eintrag aus der DB auf
// die Settings-Struktur.
func applyEinstellung(settings *SystemEinstellungen, key string, val *string) {
	switch key {
	case "ferien_leseclub_aktiv":
		settings.FerienLeseclubAktiv = val != nil && *val == "true"
	case "ferien_leseclub_zieldatum":
		if val != nil && *val != "" {
			v := *val
			settings.FerienLeseclubZieldatum = &v
		}
	case "lmf_stichtag":
		setzeStringNichtLeer(val, &settings.LmfStichtag)
	case "max_ausleihen_schueler":
		setzeIntEinstellung(val, &settings.MaxAusleihenSchueler)
	case "frist_buch_tage":
		setzeIntEinstellung(val, &settings.FristBuchTage)
	case "frist_medien_tage":
		setzeIntEinstellung(val, &settings.FristMedienTage)
	case "max_overdue_days":
		setzeIntEinstellung(val, &settings.MaxOverdueDays)
	case "max_overdue_items":
		setzeIntEinstellung(val, &settings.MaxOverdueItems)
	case "bestellbedarf_warnung_aktiv":
		settings.BestellbedarfWarnungAktiv = val != nil && *val == "true"
	case "bestellbedarf_schwelle":
		setzeIntEinstellung(val, &settings.BestellbedarfSchwelle)
	case "schule_name":
		setzeStringRoh(val, &settings.SchuleName)
	case "schule_strasse":
		setzeStringRoh(val, &settings.SchuleStrasse)
	case "schule_plz":
		setzeStringRoh(val, &settings.SchulePLZ)
	case "schule_ort":
		setzeStringRoh(val, &settings.SchuleOrt)
	}
}

// GetSettings reads system settings from the database.
func (repo *pgSystemSettingsRepository) GetSettings(ctx context.Context) (*SystemEinstellungen, error) {
	settings := standardEinstellungen()

	rows, err := repo.db.Query(ctx, `SELECT schluessel, wert FROM system_einstellungen`)
	if err != nil {
		return settings, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var val *string
		if scanErr := rows.Scan(&key, &val); scanErr != nil {
			continue
		}
		applyEinstellung(settings, key, val)
	}
	return settings, rows.Err()
}

// SaveSettings persists system settings.
func (repo *pgSystemSettingsRepository) SaveSettings(ctx context.Context, req *SystemEinstellungen) error {
	upsert := `
		INSERT INTO system_einstellungen (schluessel, wert)
		SELECT * FROM UNNEST($1::varchar[], $2::text[])
		ON CONFLICT (schluessel) DO UPDATE
		  SET wert = EXCLUDED.wert, aktualisiert_am = CURRENT_TIMESTAMP
	`

	aktiv := "false"
	if req.FerienLeseclubAktiv {
		aktiv = "true"
	}
	stichtag := req.LmfStichtag
	if stichtag == "" {
		stichtag = "07-31"
	}
	zieldatum := ""
	if req.FerienLeseclubZieldatum != nil {
		zieldatum = *req.FerienLeseclubZieldatum
	}
	maxAusleihen := "5"
	if req.MaxAusleihenSchueler > 0 {
		maxAusleihen = strconv.Itoa(req.MaxAusleihenSchueler)
	}
	fristBuch := "21"
	if req.FristBuchTage > 0 {
		fristBuch = strconv.Itoa(req.FristBuchTage)
	}
	fristMedien := "7"
	if req.FristMedienTage > 0 {
		fristMedien = strconv.Itoa(req.FristMedienTage)
	}
	maxOverdueDays := "14"
	if req.MaxOverdueDays >= 0 {
		maxOverdueDays = strconv.Itoa(req.MaxOverdueDays)
	}
	maxOverdueItems := "1"
	if req.MaxOverdueItems > 0 {
		maxOverdueItems = strconv.Itoa(req.MaxOverdueItems)
	}
	bestellSchwelle := "3"
	if req.BestellbedarfSchwelle > 0 {
		bestellSchwelle = strconv.Itoa(req.BestellbedarfSchwelle)
	}
	bestellAktiv := "false"
	if req.BestellbedarfWarnungAktiv {
		bestellAktiv = "true"
	}

	pairs := [][2]string{
		{"ferien_leseclub_aktiv", aktiv},
		{"lmf_stichtag", stichtag},
		{"ferien_leseclub_zieldatum", zieldatum},
		{"max_ausleihen_schueler", maxAusleihen},
		{"frist_buch_tage", fristBuch},
		{"frist_medien_tage", fristMedien},
		{"max_overdue_days", maxOverdueDays},
		{"max_overdue_items", maxOverdueItems},
		{"bestellbedarf_warnung_aktiv", bestellAktiv},
		{"bestellbedarf_schwelle", bestellSchwelle},
	}

	// Schul-Identität (PDF-Briefkopf) getrennt behandeln: Diese Felder haben
	// KEINE eigene UI und werden von der Allgemein-Sektion NICHT mitgesendet.
	// Ein leerer Wert bedeutet hier "nicht angefasst", nicht "leeren". Ohne diese
	// Guard würde jedes Speichern der Allgemein-Einstellungen die Schuladresse
	// löschen, die in fünf PDF-Generatoren (Mahnung, Bestellbericht, Reports,
	// Print, Orders) als Briefkopf genutzt wird.
	for _, f := range [][2]string{
		{"schule_name", req.SchuleName},
		{"schule_strasse", req.SchuleStrasse},
		{"schule_plz", req.SchulePLZ},
		{"schule_ort", req.SchuleOrt},
	} {
		if f[1] != "" {
			pairs = append(pairs, f)
		}
	}
	seen := make(map[string]bool, len(pairs))
	schluessels := make([]string, 0, len(pairs))
	werts := make([]string, 0, len(pairs))

	for _, p := range pairs {
		if !seen[p[0]] {
			seen[p[0]] = true
			schluessels = append(schluessels, p[0])
			werts = append(werts, p[1])
		}
	}

	if _, err := repo.db.Exec(ctx, upsert, schluessels, werts); err != nil {
		return err
	}

	return nil
}
