package api

import (
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// BestellKonfiguration nennt die Systemeinstellungen, nach denen sich die Oberfläche des
// Bestellwesens richtet.
type BestellKonfiguration struct {
	PreiseErfassen bool `json:"preise_erfassen"`
}

// BestellKonfigurationHandler liefert die Anzeige-Regeln des Bestellwesens.
//
// Eigene, schmale Route statt GET /api/einstellungen: Die vollständigen Einstellungen
// verlangen manage_users, das Bestellwesen aber nur view_orders. Wer bestellen darf, müsste
// sonst Benutzerverwaltungsrechte bekommen, nur damit sein Warenkorb weiss, ob er ein
// Preisfeld zeigen soll — ein Recht auszuweiten, um eine Anzeige zu steuern, ist der
// falsche Weg herum.
//
// Zugeschnitten auf das Bestellwesen und auch so benannt: Braucht später eine andere
// Ansicht eine Anzeige-Regel, gehört sie nicht hierher, sondern an ihre eigene Domäne —
// sonst wächst hier still ein zweiter Einstellungs-Endpunkt heran.
//
// @Summary      Anzeige-Regeln des Bestellwesens
// @Tags         orders
// @Produce      json
// @Success      200  {object}  BestellKonfiguration
// @Router       /bestellungen/konfiguration [get]
func (s *Server) BestellKonfigurationHandler(settingsRepo repository.SystemSettingsRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		settings, err := settingsRepo.GetSettings(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Bestell-Konfiguration", err)
		}

		RespondJSON(w, http.StatusOK, BestellKonfiguration{PreiseErfassen: settings.PreiseErfassen})
		return nil
	})
}
