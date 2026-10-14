package benutzer

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// rollenRecht holds a single role-permission flag returned by the API.
type rollenRecht struct {
	Rolle   string `json:"role"`
	Recht   string `json:"permission"`
	Erlaubt bool   `json:"allowed"`
}

// RechteMatrix returns all permission configurations grouped by role.
// @Summary      Get role permissions
// @Description  Retrieves current allowed/denied flags for permissions across all system roles.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {array}   rollenRecht
// @Failure      500  {object}  map[string]string
// @Router       /admin/permissions [get]
func (t *Tueren) RechteMatrix() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rechte, err := repository.LadeRollenRechte(ctx, t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		liste := []rollenRecht{}
		for _, re := range rechte {
			// Die Oberfläche führt die Rollen klein geschrieben.
			liste = append(liste, rollenRecht{Rolle: strings.ToLower(re.Rolle), Recht: re.Recht, Erlaubt: re.Erlaubt})
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// rollenRechtAnfrage is the payload for toggling a single role-permission flag.
type rollenRechtAnfrage struct {
	Rolle   string `json:"role"`
	Recht   string `json:"permission"`
	Erlaubt bool   `json:"allowed"`
}

// RollenRechtSetzen updates a specific permission setting.
// @Summary      Update role permission
// @Description  Enables or disables a specific permission for a user role.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body  body      rollenRechtAnfrage  true  "Permission update payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /admin/permissions [put]
func (t *Tueren) RollenRechtSetzen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req rollenRechtAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		// Wer Konten verwaltet, definiert damit nicht, was Rollen dürfen.
		//
		// Vorher genügte manage_users, und das schloss den Kreis: Der Endpunkt konnte der
		// EIGENEN Rolle jedes beliebige Recht zuschalten — audit_logs, delete_students,
		// und bis zum 06.08.2026 auch manage_users für weitere Rollen. Ein delegiertes
		// Verwaltungsrecht wuchs so in einem Schritt zu vollem Zugriff aus. Die Rechte-
		// Matrix ist Konfiguration des Systems, nicht Tagesgeschäft der Kontoverwaltung.
		if !istAufruferAdmin(r) {
			apierrors.SendHTTPError(w, http.StatusForbidden,
				errors.New("die Rechte-Matrix kann nur ein Administrator ändern"))
			return
		}

		ctx := r.Context()

		getroffen, err := repository.SetzeRollenRecht(ctx, t.DB.Pool, strings.ToUpper(req.Rolle), req.Recht, req.Erlaubt)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Kein Treffer heißt: Rolle oder Recht gibt es nicht. Ohne diese Prüfung
		// aktualisierte der Endpunkt null Zeilen und meldete trotzdem "success" — ein
		// Tippfehler im Rechtenamen sah in der Oberfläche aus wie ein gesetzter Haken,
		// blieb in der Datenbank aber wirkungslos. Genau die Bugklasse "still verworfen,
		// trotzdem 200", die dieses Projekt schon mehrfach getroffen hat.
		if getroffen == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				fmt.Errorf("unbekannte Kombination aus Rolle %q und Recht %q — nichts geändert", req.Rolle, req.Recht))
			return
		}

		// Die Matrix entscheidet an jeder Route, wer welche Daten sieht: Wer sie ändert, steht
		// im Protokoll wie bei Konten und Einstellungen.
		t.ProtokolliereVerwaltung(r.Context(), "RECHT_GEAENDERT", map[string]any{
			"rolle": strings.ToUpper(req.Rolle), "recht": req.Recht, "erlaubt": req.Erlaubt,
		})

		// Invalidate permission cache so permission changes take effect immediately
		tuer.LeereRechteCache()

		w.Header().Set("Content-Type", "application/json")
		httpresp.Schreibe(w, []byte(`{"status":"success"}`))
	}
}
