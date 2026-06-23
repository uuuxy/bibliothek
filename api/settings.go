package api

import (
	"encoding/json"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// GetSettingsHandler returns all system settings.
// @Summary      Get system settings
// @Description  Retrieves global configuration values like loan limits, grace periods, and feature flags.
// @Tags         system
// @Accept       json
// @Produce      json
// @Success      200  {object}  repository.SystemEinstellungen
// @Failure      500  {object}  map[string]string
// @Router       /einstellungen [post]
func (s *Server) GetSettingsHandler(settingsRepo repository.SystemSettingsRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		settings, err := settingsRepo.GetSettings(ctx)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Einstellungen", err)
		}
		RespondJSON(w, http.StatusOK, settings)
		return nil
	})
}

// UpdateSettingsHandler persists system settings.
// @Summary      Update system settings
// @Description  Saves global configuration values. Requires admin privileges.
// @Tags         system
// @Accept       json
// @Produce      json
// @Param        settings  body      repository.SystemEinstellungen  true  "Updated settings"
// @Success      200       {object}  map[string]string
// @Failure      400       {object}  map[string]string
// @Failure      500       {object}  map[string]string
// @Router       /einstellungen/speichern [post]
func (s *Server) UpdateSettingsHandler(settingsRepo repository.SystemSettingsRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req repository.SystemEinstellungen
		if !DecodeAndValidate(w, r, &req) {
			return nil // Error is already sent by DecodeAndValidate
		}

		ctx := r.Context()

		if err := settingsRepo.SaveSettings(ctx, &req); err != nil {
			return apierrors.Internal("Fehler beim Speichern der Einstellungen", err)
		}

		// Admin audit log (IP-Adresse wird gemäß DSGVO nicht gespeichert)
		if claims, ok := auth.GetClaims(r.Context()); ok {
			if detailsBytes, merr := json.Marshal(req); merr != nil {
				log.Printf("audit: Settings-Details konnten nicht serialisiert werden: %v", merr)
			} else {
				logExec(s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details) VALUES ($1, $2, $3::jsonb)", claims.UserID, "UPDATE_SETTINGS", string(detailsBytes)))
			}
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return nil
	})
}
