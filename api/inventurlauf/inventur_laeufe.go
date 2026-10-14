package inventurlauf

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// offenerLauf ist ein offener Lauf, wie die Oberfläche ihn zeigt.
type offenerLauf struct {
	LaufID      string `json:"session_id"`
	Bereich     string `json:"scope"`
	Bezeichnung string `json:"label"`
	GestartetAm string `json:"gestartet_am"`
	Erfasst     int    `json:"erfasst"`
	Erwartet    int    `json:"erwartet"`
}

// abgeschlossenerLauf ist eine fertige Inventur, wie die Oberfläche sie zur Auswahl zeigt:
// wann, welcher Bereich, wie viele Verluste und ob sie verworfen wurde; dann ist der Bereich
// nicht geprüft, auch ohne Verlust.
type abgeschlossenerLauf struct {
	LaufID          string `json:"session_id"`
	Bezeichnung     string `json:"label"`
	AbgeschlossenAm string `json:"abgeschlossen_am"`
	Erfasst         int    `json:"erfasst"`
	Verluste        int    `json:"verluste"`
	Verworfen       bool   `json:"verworfen"`
}

// AbgeschlosseneInventuren liefert die zehn zuletzt abgeschlossenen Inventuren.
//
// Über sie kommt die Oberfläche an die Kennung für GET /api/inventur/fehlbestand: Die Liste
// der laufenden Inventuren nennt eine fertige nicht mehr, und ohne Kennung bliebe der
// Fehlbestand nach einem Neuladen oder an einem zweiten Arbeitsplatz unerreichbar, obwohl er
// in inventur_verluste gespeichert ist.
//
// @Summary      Abgeschlossene Inventuren
// @Tags         inventory
// @Produce      json
// @Success      200   {array}   abgeschlossenerLauf
// @Failure      500   {object}  map[string]string
// @Router       /inventur/abgeschlossen [get]
func (t *Tueren) AbgeschlosseneInventuren() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		abfragen := repository.NewInventurAbfragen(t.DB.Pool)

		laeufe, err := abfragen.LadeAbgeschlosseneLaeufe(r.Context(), 10)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		liste := make([]abgeschlossenerLauf, 0, len(laeufe))
		for i := range laeufe {
			abgeschlossen := ""
			if laeufe[i].AbgeschlossenAm != nil {
				abgeschlossen = *laeufe[i].AbgeschlossenAm
			}
			liste = append(liste, abgeschlossenerLauf{
				LaufID:          laeufe[i].ID,
				Bezeichnung:     laeufe[i].Bezeichnung,
				AbgeschlossenAm: abgeschlossen,
				Erfasst:         laeufe[i].Erfasst,
				Verluste:        laeufe[i].Verluste,
				Verworfen:       laeufe[i].Verworfen,
			})
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// LaufendeInventuren liefert alle offenen Läufe. Die Oberfläche zeigt sie, damit eine
// Lehrkraft eine laufende Inventur fortsetzt, statt vergeblich eine zweite im selben Bereich
// zu starten.
//
// @Summary      Laufende Inventuren
// @Tags         inventory
// @Produce      json
// @Success      200   {array}   offenerLauf
// @Failure      500   {object}  map[string]string
// @Router       /inventur/sessions [get]
func (t *Tueren) LaufendeInventuren() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		abfragen := repository.NewInventurAbfragen(t.DB.Pool)

		laeufe, err := abfragen.LadeOffeneLaeufe(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		liste := make([]offenerLauf, 0, len(laeufe))
		for i := range laeufe {
			erwartet, err := abfragen.ZaehleBereich(ctx, laeufe[i].Bereich())
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			liste = append(liste, offenerLauf{
				LaufID:      laeufe[i].ID,
				Bereich:     laeufe[i].BereichArt,
				Bezeichnung: laeufe[i].Bezeichnung,
				GestartetAm: laeufe[i].GestartetAm,
				Erfasst:     laeufe[i].Erfasst,
				Erwartet:    erwartet,
			})
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// verwerfenAnfrage nennt den Lauf, der verworfen wird.
type verwerfenAnfrage struct {
	LaufID string `json:"session_id" validate:"omitempty,uuid_oder_leer"`
}

// InventurVerwerfen verwirft einen Lauf, ohne Verluste zu buchen, etwa eine abgebrochene oder
// hängen gebliebene Inventur. Danach ist der Bereich für einen neuen Lauf frei.
//
// @Summary      Inventur verwerfen
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      verwerfenAnfrage   true  "Lauf"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/abort [post]
func (t *Tueren) InventurVerwerfen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verwerfenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}
		if req.LaufID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("session_id fehlt"))
			return
		}

		ctx := r.Context()
		abfragen := repository.NewInventurAbfragen(t.DB.Pool)
		if err := abfragen.VerwerfeLauf(ctx, req.LaufID); err != nil {
			if errors.Is(err, repository.ErrInventurlaufNichtGefunden) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteErfolg(w)
	}
}
