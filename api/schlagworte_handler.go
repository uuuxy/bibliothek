package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// Schlagworte am Titel (Migration 138, docs/OFFEN.md 4.20). Die Regeln stehen in
// repository.SetzeSchlagworte; das Buchformular erreicht sie über PUT /api/books/{id},
// der Bestellkorb über die Tür hier — beide über dieselbe Funktion.

// TitelSchlagworteRequest ist die Eingabe für PutTitelSchlagworteHandler.
type TitelSchlagworteRequest struct {
	// Zeiger: Ein Körper ohne das Feld ist ein Fehler, keine leere Liste. Diese Tür hat
	// keinen anderen Zweck, als die Schlagworte zu setzen — ein vergessenes Feld darf
	// nicht still alle entfernen.
	Schlagworte *[]string `json:"schlagworte"`
}

// TitelSchlagworte ist die Antwort beider Titel-Türen: die Schlagworte in der
// gespeicherten Schreibweise, alphabetisch.
type TitelSchlagworte struct {
	ID          string   `json:"id"`
	Schlagworte []string `json:"schlagworte"`
}

// GetSchlagwortVorschlaegeHandler liefert die Schlagworte, die im Bestand vorkommen —
// die häufigsten zuerst, gekappt auf 500. Die Vorschlagsliste der Eingabefelder, wie
// GET /api/signaturen für die Signatur.
//
// @Summary      List keyword suggestions
// @Description  Keywords carried by at least one title, most frequent first (max. 500).
// @Tags         books
// @Produce      json
// @Success      200  {array}   repository.SchlagwortZahl
// @Failure      500  {object}  map[string]string
// @Router       /schlagworte [get]
func (s *Server) GetSchlagwortVorschlaegeHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		vorschlaege, err := repository.SchlagwortVorschlaege(r.Context(), s.DB.Pool)
		if err != nil {
			return apierrors.Internal("Schlagworte konnten nicht geladen werden", err)
		}
		RespondJSON(w, http.StatusOK, vorschlaege)
		return nil
	})
}

// GetTitelSchlagworteHandler liefert die Schlagworte eines Titels. Der Bestellkorb
// braucht sie, bevor er ändert: Er setzt die Menge als Ganzes, und wer die vorhandenen
// nicht kennt, würde sie mit seinem ersten Wort überschreiben.
//
// @Summary      Get a title's keywords
// @Tags         books
// @Produce      json
// @Param        id   path      string  true  "Title ID"
// @Success      200  {object}  TitelSchlagworte
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /buecher/titel/{id}/schlagworte [get]
func (s *Server) GetTitelSchlagworteHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if !kennung.IstUUID(id) {
			return apierrors.BadRequest("ungültige Titel-ID", errors.New("keine UUID"))
		}
		woerter, err := repository.SchlagworteDesTitels(r.Context(), s.DB.Pool, id)
		if err != nil {
			return schlagwortFehler(err)
		}
		RespondJSON(w, http.StatusOK, TitelSchlagworte{ID: id, Schlagworte: woerter})
		return nil
	})
}

// PutTitelSchlagworteHandler ersetzt die Schlagworte eines Titels — die schlanke Tür
// des Bestellkorbs, Schwester von UpdateTitelSignaturHandler. Eine leere Liste entfernt
// alle; ein Wort, das es in anderer Schreibweise schon gibt, wird nicht neu angelegt.
//
// @Summary      Replace a title's keywords
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path      string                   true  "Title ID"
// @Param        body  body      TitelSchlagworteRequest  true  "Complete keyword list"
// @Success      200   {object}  TitelSchlagworte
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /buecher/titel/{id}/schlagworte [put]
func (s *Server) PutTitelSchlagworteHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if !kennung.IstUUID(id) {
			return apierrors.BadRequest("ungültige Titel-ID", errors.New("keine UUID"))
		}
		var req TitelSchlagworteRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if req.Schlagworte == nil {
			return apierrors.BadRequest("schlagworte fehlt — zum Entfernen aller Schlagworte eine leere Liste schicken",
				errors.New("feld schlagworte fehlt"))
		}
		woerter, err := repository.SetzeSchlagworte(r.Context(), s.DB.Pool, id, *req.Schlagworte)
		if err != nil {
			return schlagwortFehler(err)
		}
		RespondJSON(w, http.StatusOK, TitelSchlagworte{ID: id, Schlagworte: woerter})
		return nil
	})
}

// schlagwortFehler übersetzt die fachlichen Fehler des Repositorys: eine verletzte
// Grenze ist 400 mit ihrem Text, ein unbekannter Titel 404, alles andere 500.
func schlagwortFehler(err error) error {
	switch {
	case errors.Is(err, repository.ErrSchlagwortUngueltig):
		return apierrors.BadRequest(err.Error(), err)
	case errors.Is(err, repository.ErrTitelNichtGefunden):
		return apierrors.NotFound("Titel nicht gefunden", err)
	default:
		return apierrors.Internal("Schlagworte konnten nicht gespeichert werden", err)
	}
}
