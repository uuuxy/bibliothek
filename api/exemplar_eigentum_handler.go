package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// ExemplarEigentumRequest ist der Körper von PUT /api/exemplare/eigentum.
type ExemplarEigentumRequest struct {
	// Jede Kennung wird an der Tür geprüft (400 statt 22P02/500), leer ist keine Auswahl.
	ExemplarIDs []string `json:"exemplar_ids" validate:"required,dive,required,uuid_oder_leer"`
	// Eigentum: "land", "schultraeger" oder "" — leer nimmt die Angabe am Exemplar weg, dann
	// gilt wieder der Topf der Bestellung oder die Faustregel aus dem Titel.
	Eigentum string `json:"eigentum"`
	Grund    string `json:"grund"`
}

// ExemplarEigentumHandler setzt das Eigentum markierter Exemplare (docs/OFFEN.md 4.24,
// Stufe 3) — die Buchakte, Reiter „Exemplare": markieren, „Eigentum ändern".
//
// Recht edit_books wie Barcode und Status desselben Exemplars. Der Grund ist Pflicht; jede
// Änderung steht mit altem und neuem Wert im Protokoll (repository.SetzeExemplarEigentum).
//
// @Summary      Eigentum markierter Exemplare setzen
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        body  body      ExemplarEigentumRequest  true  "Exemplare, Eigentum, Grund"
// @Success      200   {object}  map[string]int
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /exemplare/eigentum [put]
func (s *Server) ExemplarEigentumHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req ExemplarEigentumRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("nicht angemeldet", errors.New("keine Sitzung"))
		}

		geaendert, err := repository.SetzeExemplarEigentum(r.Context(), s.DB.Pool, repository.EigentumAenderung{
			ExemplarIDs: req.ExemplarIDs, Eigentum: req.Eigentum, Grund: req.Grund, BearbeiterID: claims.UserID,
		})
		switch {
		case errors.Is(err, repository.ErrEigentumUngueltig):
			return apierrors.BadRequest(err.Error(), err)
		case errors.Is(err, repository.ErrExemplarNichtGefunden):
			// Alle oder keins: Der Satz sagt, dass auch die übrigen unverändert sind.
			return apierrors.NotFound("Ein gewähltes Exemplar gibt es nicht mehr — geändert wurde nichts. Bitte die Seite neu laden.", err)
		case err != nil:
			return apierrors.Internal("Das Eigentum konnte nicht gespeichert werden", err)
		}
		RespondJSON(w, http.StatusOK, map[string]int{"geaendert": geaendert})
		return nil
	})
}
