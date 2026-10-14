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

// rollenRecht ist ein Feld der Rechte-Matrix: ob eine Rolle ein Recht hat.
type rollenRecht struct {
	Rolle   string `json:"role"`
	Recht   string `json:"permission"`
	Erlaubt bool   `json:"allowed"`
}

// RechteMatrix liefert für jede Rolle und jedes Recht, ob es erlaubt ist.
// @Summary      Rechte-Matrix
// @Description  Liefert je Rolle und Recht, ob es erlaubt ist.
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

// rollenRechtAnfrage schaltet ein Recht für eine Rolle an oder ab.
type rollenRechtAnfrage struct {
	Rolle   string `json:"role"`
	Recht   string `json:"permission"`
	Erlaubt bool   `json:"allowed"`
}

// RollenRechtSetzen schaltet ein Recht für eine Rolle an oder ab; das darf nur ein
// Administrator.
// @Summary      Recht einer Rolle setzen
// @Description  Schaltet ein Recht für eine Rolle an oder ab.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body  body      rollenRechtAnfrage  true  "Rolle, Recht und ob es erlaubt ist"
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

		// Wer Konten verwaltet, legt damit nicht fest, was Rollen dürfen: Mit manage_users
		// allein könnte man der eigenen Rolle jedes Recht zuschalten und ein weitergegebenes
		// Recht in einem Schritt zu vollem Zugriff ausbauen. Die Rechte-Matrix ist
		// Konfiguration des Systems, nicht Tagesgeschäft der Kontoverwaltung.
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

		// Kein Treffer heißt: Rolle oder Recht gibt es nicht. Meldete die Tür trotzdem
		// Erfolg, sähe ein Tippfehler im Namen des Rechts in der Oberfläche aus wie ein
		// gesetzter Haken, der in der Datenbank nichts bewirkt.
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

		// Den Rechte-Cache leeren, damit die Änderung sofort gilt.
		tuer.LeereRechteCache()

		w.Header().Set("Content-Type", "application/json")
		httpresp.Schreibe(w, []byte(`{"status":"success"}`))
	}
}
