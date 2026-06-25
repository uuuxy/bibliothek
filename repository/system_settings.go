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

// GetSettings reads system settings from the database.
func (repo *pgSystemSettingsRepository) GetSettings(ctx context.Context) (*SystemEinstellungen, error) {
	rows, err := repo.db.Query(ctx, `SELECT schluessel, wert FROM system_einstellungen`)
	if err != nil {
		return &SystemEinstellungen{LmfStichtag: "07-31", MaxAusleihenSchueler: 5, FristBuchTage: 21, FristMedienTage: 7, MaxOverdueDays: 14, MaxOverdueItems: 1}, err
	}
	defer rows.Close()

	settings := &SystemEinstellungen{LmfStichtag: "07-31", MaxAusleihenSchueler: 5, FristBuchTage: 21, FristMedienTage: 7, MaxOverdueDays: 14, MaxOverdueItems: 1}
	for rows.Next() {
		var key string
		var val *string
		if scanErr := rows.Scan(&key, &val); scanErr != nil {
			continue
		}
		switch key {
		case "ferien_leseclub_aktiv":
			settings.FerienLeseclubAktiv = val != nil && *val == "true"
		case "ferien_leseclub_zieldatum":
			if val != nil && *val != "" {
				v := *val
				settings.FerienLeseclubZieldatum = &v
			}
		case "lmf_stichtag":
			if val != nil && *val != "" {
				settings.LmfStichtag = *val
			}
		case "max_ausleihen_schueler":
			if val != nil && *val != "" {
				if v, err := strconv.Atoi(*val); err == nil {
					settings.MaxAusleihenSchueler = v
				}
			}
		case "frist_buch_tage":
			if val != nil && *val != "" {
				if v, err := strconv.Atoi(*val); err == nil {
					settings.FristBuchTage = v
				}
			}
		case "frist_medien_tage":
			if val != nil && *val != "" {
				if v, err := strconv.Atoi(*val); err == nil {
					settings.FristMedienTage = v
				}
			}
		case "max_overdue_days":
			if val != nil && *val != "" {
				if v, err := strconv.Atoi(*val); err == nil {
					settings.MaxOverdueDays = v
				}
			}
		case "max_overdue_items":
			if val != nil && *val != "" {
				if v, err := strconv.Atoi(*val); err == nil {
					settings.MaxOverdueItems = v
				}
			}
		case "schule_name":
			if val != nil {
				settings.SchuleName = *val
			}
		case "schule_strasse":
			if val != nil {
				settings.SchuleStrasse = *val
			}
		case "schule_plz":
			if val != nil {
				settings.SchulePLZ = *val
			}
		case "schule_ort":
			if val != nil {
				settings.SchuleOrt = *val
			}
		}
	}
	return settings, rows.Err()
}

// SaveSettings persists system settings.
func (repo *pgSystemSettingsRepository) SaveSettings(ctx context.Context, req *SystemEinstellungen) error {
	upsert := `
		INSERT INTO system_einstellungen (schluessel, wert)
		VALUES ($1, $2)
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

	pairs := [][2]string{
		{"ferien_leseclub_aktiv", aktiv},
		{"lmf_stichtag", stichtag},
		{"ferien_leseclub_zieldatum", zieldatum},
		{"max_ausleihen_schueler", maxAusleihen},
		{"frist_buch_tage", fristBuch},
		{"frist_medien_tage", fristMedien},
		{"max_overdue_days", maxOverdueDays},
		{"max_overdue_items", maxOverdueItems},
		{"schule_name", req.SchuleName},
		{"schule_strasse", req.SchuleStrasse},
		{"schule_plz", req.SchulePLZ},
		{"schule_ort", req.SchuleOrt},
	}
	for _, p := range pairs {
		if _, err := repo.db.Exec(ctx, upsert, p[0], p[1]); err != nil {
			return err
		}
	}

	return nil
}
