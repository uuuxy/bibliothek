package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// exemplarStandortAnfrage ist der Körper von PUT /api/exemplare/standort.
type exemplarStandortAnfrage struct {
	// Jede Kennung wird an der Tür geprüft (400 statt 22P02/500), leer ist keine Auswahl.
	ExemplarIDs []string `json:"exemplar_ids" validate:"required,dive,required,uuid_oder_leer"`
	// Standort: Freitext bis 255 Zeichen. Leer nimmt die Angabe weg, das Exemplar steht dann
	// wieder nach der Signatur. Zeiger, weil ein fehlendes Feld kein Auftrag zum Entfernen ist.
	Standort *string `json:"standort" validate:"required"`
}

// ExemplarStandortSetzen setzt den Standort markierter Exemplare (docs/OFFEN.md 5.53) — die
// Buchakte, Reiter „Exemplare": markieren, „Standort ändern".
//
// Recht edit_books wie Barcode, Status und Eigentum desselben Exemplars. Jede Änderung steht
// mit altem und neuem Wert im Protokoll (repository.SetzeExemplarStandort).
//
// @Summary      Standort markierter Exemplare setzen
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        body  body      exemplarStandortAnfrage  true  "Exemplare, Standort"
// @Success      200   {object}  map[string]int
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /exemplare/standort [put]
func (t *Tueren) ExemplarStandortSetzen() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req exemplarStandortAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("nicht angemeldet", errors.New("keine Sitzung"))
		}

		geaendert, err := repository.SetzeExemplarStandort(r.Context(), t.DB.Pool, repository.StandortAenderung{
			ExemplarIDs: req.ExemplarIDs, Standort: *req.Standort, BearbeiterID: claims.UserID,
		})
		switch {
		case errors.Is(err, repository.ErrStandortUngueltig):
			return apierrors.BadRequest(err.Error(), err)
		case errors.Is(err, repository.ErrExemplarNichtGefunden):
			// Alle oder keins: Der Satz sagt, dass auch die übrigen unverändert sind.
			return apierrors.NotFound("Ein gewähltes Exemplar gibt es nicht mehr — geändert wurde nichts. Bitte die Seite neu laden.", err)
		case err != nil:
			return apierrors.Internal("Der Standort konnte nicht gespeichert werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, map[string]int{"geaendert": geaendert})
		return nil
	})
}

// ExemplarStandorte liefert die Standorte, die an Exemplaren im Bestand vorkommen, mit
// ihrer Zahl — die Vorschläge des Dialogs „Standort ändern". Wer aus ihnen wählt, schreibt
// keinen zweiten Namen für dasselbe Regal.
//
// @Summary      Standorte der Exemplare im Bestand
// @Tags         books
// @Produce      json
// @Success      200  {array}   repository.StandortZahl
// @Failure      500  {object}  map[string]string
// @Router       /exemplare/standorte [get]
func (t *Tueren) ExemplarStandorte() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		liste, err := repository.LadeExemplarStandorte(r.Context(), t.DB.Pool)
		if err != nil {
			return apierrors.Internal("Die Standorte konnten nicht geladen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}
