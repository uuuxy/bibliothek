package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/lmfplan"
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
// @Description  Saves global configuration values. Nur mitgeschickte Felder werden geschrieben; fehlende bleiben unveraendert. Requires admin privileges.
// @Tags         system
// @Accept       json
// @Produce      json
// @Param        settings  body      repository.EinstellungenPatch  true  "Nur die Felder der gespeicherten Kategorie"
// @Success      200       {object}  map[string]string
// @Failure      400       {object}  map[string]string
// @Failure      500       {object}  map[string]string
// @Router       /einstellungen/speichern [post]
func (s *Server) UpdateSettingsHandler(settingsRepo repository.SystemSettingsRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return s.handleUpdateSettings(w, r, settingsRepo)
	})
}

// handleUpdateSettings prüft den Patch, speichert ihn, protokolliert die Änderung und meldet neue
// Sitzungsfristen an die offenen Arbeitsplätze.
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request, settingsRepo repository.SystemSettingsRepository) error {
	// Patch statt vollem Objekt: Der Rumpf trägt nur die Felder einer Kategorie
	// (system_settings_patch.go). Ein volles Objekt dekodierte hier zu lauter Nullwerten
	// und setzte beim Speichern die übrigen Kategorien zurück.
	// Streng dekodiert (DecodeStrictAndValidate): Der Rumpf entsteht aus einer
	// geschlossenen Liste benannter Schlüssel, ein unbekanntes Feld ist hier also
	// immer ein Fehler — und zwar der teuerste, weil er sonst still verschwindet
	// und die Oberfläche trotzdem "gespeichert" meldet.
	var req repository.EinstellungenPatch
	if !DecodeStrictAndValidate(w, r, &req) {
		return nil // Error is already sent by DecodeAndValidate
	}

	if req.IstLeer() {
		return apierrors.BadRequest("Es wurde keine einzige Einstellung mitgeschickt.",
			errors.New("leerer Einstellungs-Patch"))
	}
	// Zahlen: innerhalb ihrer Spanne oder gar nicht gespeichert
	// (repository/system_settings_zahlen.go). Beide Prüfungen stehen vor dem
	// Protokoll-Eintrag, damit dort nicht die Eingabe steht, während in der Datenbank
	// etwas anderes liegt.
	if err := req.PruefeZahlen(); err != nil {
		return apierrors.BadRequest(err.Error(), err)
	}
	if err := normalisiereLernmittelAngaben(&req); err != nil {
		return apierrors.BadRequest(err.Error(), err)
	}

	if err := settingsRepo.SaveSettings(r.Context(), &req); err != nil {
		return apierrors.Internal("Fehler beim Speichern der Einstellungen", err)
	}
	s.protokolliereEinstellungen(r, req)

	// Die Sitzungsfristen holt jeder Tab nur einmal beim Anmelden (App.svelte).
	// Das Signal erreicht auch offene Tabs an anderen Arbeitsplätzen; die Werte
	// holen sich die Clients per GET /api/einstellungen/sitzung — dort sitzt die
	// Vorgaben-Logik, hier wäre sie dupliziert.
	if s.Broker != nil && (req.ThekeLeerenMinuten != nil || req.SperreMinuten != nil) {
		s.Broker.Broadcast("sitzungsfristen", "{}")
	}

	RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	return nil
}

// normalisiereLernmittelAngaben bringt Sommerferien, Eingangsjahrgänge und Stichtag in
// Normalform. An ihnen hängt jede Lernmittel-Frist: Unlesbares wird nicht gespeichert, statt
// beim Lesen still auf die Vorgabe zurückzufallen.
func normalisiereLernmittelAngaben(req *repository.EinstellungenPatch) error {
	if req.Sommerferien != nil {
		norm, err := lmfplan.NormalisiereSommerferien(*req.Sommerferien)
		if err != nil {
			return err
		}
		req.Sommerferien = &norm
	}
	if req.LmfEingangsjahrgaenge != nil {
		norm, err := repository.NormalisiereEingangsjahrgaenge(*req.LmfEingangsjahrgaenge)
		if err != nil {
			return err
		}
		req.LmfEingangsjahrgaenge = &norm
	}
	if req.LmfStichtag != nil {
		norm, err := repository.NormalisiereLmfStichtag(*req.LmfStichtag)
		if err != nil {
			return err
		}
		req.LmfStichtag = &norm
	}
	return nil
}

// protokolliereEinstellungen schreibt den gespeicherten Patch ins Protokoll; die IP-Adresse
// wird nicht gespeichert.
func (s *Server) protokolliereEinstellungen(r *http.Request, req repository.EinstellungenPatch) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	detailsBytes, merr := json.Marshal(req)
	if merr != nil {
		log.Printf("audit: Settings-Details konnten nicht serialisiert werden: %v", merr)
		return
	}
	logExec(s.DB.Pool.Exec(r.Context(), "INSERT INTO audit_logs (admin_id, aktion, details) VALUES ($1, $2, $3::jsonb)", claims.UserID, "UPDATE_SETTINGS", string(detailsBytes)))
}

// SitzungsEinstellungen sind die zwei Inaktivitäts-Fristen des Clients (A4 in
// docs/datenschutz_offene_punkte.md): Minuten bis die Theken-Ansicht den geladenen
// Schüler fallen lässt, Minuten bis zum Sperrbildschirm. 0 = aus.
type SitzungsEinstellungen struct {
	ThekeLeerenMinuten int `json:"theke_leeren_minuten"`
	SperreMinuten      int `json:"sperre_minuten"`
}

// GetSitzungsEinstellungenHandler liefert die Sitzungs-Fristen an jeden angemeldeten Client.
func (s *Server) GetSitzungsEinstellungenHandler(settingsRepo repository.SystemSettingsRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		settings, err := settingsRepo.GetSettings(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Einstellungen", err)
		}
		RespondJSON(w, http.StatusOK, SitzungsEinstellungen{
			ThekeLeerenMinuten: repository.TageOderStandard(settings.ThekeLeerenMinuten, repository.StandardThekeLeerenMinuten),
			SperreMinuten:      repository.TageOderStandard(settings.SperreMinuten, repository.StandardSperreMinuten),
		})
		return nil
	})
}
