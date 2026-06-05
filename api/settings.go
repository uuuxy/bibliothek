package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"bibliothek/apierrors"
)

// SystemEinstellungen holds configurable system-wide settings.
type SystemEinstellungen struct {
	FerienLeseclubAktiv     bool    `json:"ferien_leseclub_aktiv"`
	FerienLeseclubZieldatum *string `json:"ferien_leseclub_zieldatum"` // ISO date string "YYYY-MM-DD" or null
	LmfStichtag             string  `json:"lmf_stichtag"`              // "MM-DD" format, e.g. "07-31"
	MaxAusleihenSchueler    int     `json:"max_ausleihen_schueler"`
	FristBuchTage           int     `json:"frist_buch_tage"`
	FristMedienTage         int     `json:"frist_medien_tage"`
}

// querySettings reads system settings from the database, returning safe defaults on error.
func (s *Server) querySettings(ctx context.Context) (*SystemEinstellungen, error) {
	rows, err := s.DB.Pool.Query(ctx, `SELECT schluessel, wert FROM system_einstellungen`)
	if err != nil {
		return &SystemEinstellungen{LmfStichtag: "07-31", MaxAusleihenSchueler: 5, FristBuchTage: 21, FristMedienTage: 7}, nil
	}
	defer rows.Close()

	settings := &SystemEinstellungen{LmfStichtag: "07-31", MaxAusleihenSchueler: 5, FristBuchTage: 21, FristMedienTage: 7}
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
		}
	}
	return settings, rows.Err()
}

// GetSettingsHandler returns all system settings.
func (s *Server) GetSettingsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		settings, err := s.querySettings(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settings)
	}
}

// UpdateSettingsHandler persists system settings.
func (s *Server) UpdateSettingsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SystemEinstellungen
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		upsert := `
			INSERT INTO system_einstellungen (schluessel, wert)
			VALUES ($1, $2), ($3, $4), ($5, $6), ($7, $8), ($9, $10), ($11, $12)
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

		if _, err := s.DB.Pool.Exec(ctx, upsert,
			"ferien_leseclub_aktiv", aktiv,
			"lmf_stichtag", stichtag,
			"ferien_leseclub_zieldatum", zieldatum,
			"max_ausleihen_schueler", maxAusleihen,
			"frist_buch_tage", fristBuch,
			"frist_medien_tage", fristMedien,
		); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
