package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"bibliothek/apierrors"
)

// SystemEinstellungen holds configurable system-wide settings.
type SystemEinstellungen struct {
	FerienLeseclubAktiv     bool    `json:"ferien_leseclub_aktiv"`
	FerienLeseclubZieldatum *string `json:"ferien_leseclub_zieldatum"` // ISO date string "YYYY-MM-DD" or null
	LmfStichtag             string  `json:"lmf_stichtag"`              // "MM-DD" format, e.g. "07-31"
}

// querySettings reads system settings from the database, returning safe defaults on error.
func (s *Server) querySettings(ctx context.Context) (*SystemEinstellungen, error) {
	rows, err := s.DB.Pool.Query(ctx, `SELECT schluessel, wert FROM system_einstellungen`)
	if err != nil {
		return &SystemEinstellungen{LmfStichtag: "07-31"}, nil
	}
	defer rows.Close()

	settings := &SystemEinstellungen{LmfStichtag: "07-31"}
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

		pairs := [][2]string{
			{"ferien_leseclub_aktiv", aktiv},
			{"lmf_stichtag", stichtag},
			{"ferien_leseclub_zieldatum", zieldatum},
		}
		for _, p := range pairs {
			if _, err := s.DB.Pool.Exec(ctx, upsert, p[0], p[1]); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
