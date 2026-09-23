package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// Pflege der Schlagworte (Migration 143, docs/OFFEN.md 4.20 Stufe 1). Die Regeln stehen in
// repository/schlagworte_pflege.go; jede Tür hier ist eine Aktion der Pflegeseite, nicht ein
// allgemeines PATCH — so trägt jede ihre Regel sichtbar mit.

// SchlagwortWortRequest ist die neue Schreibweise für PUT /api/schlagworte/{id}/wort.
// AlteAlsVerweis: ob die alte Schreibweise als Verweis stehen bleibt
// (repository.BenenneSchlagwortUm). Pflicht wie ist_filter — ohne das Feld wäre die Vorgabe
// je Tür eine andere (Zusammenführen ließ den Verweis immer stehen, Umbenennen nie).
type SchlagwortWortRequest struct {
	Wort           string `json:"wort"`
	AlteAlsVerweis *bool  `json:"alte_als_verweis"`
}

// SchlagwortFilterRequest setzt oder nimmt die Filter-Markierung. Zeiger: Ein Körper ohne
// das Feld ist ein Fehler, kein „aus".
type SchlagwortFilterRequest struct {
	IstFilter *bool `json:"ist_filter"`
}

// SchlagwortZielRequest nennt das Ziel eines Zusammenführens und, Pflicht wie beim
// Umbenennen, ob das alte Wort als Verweis stehen bleibt.
type SchlagwortZielRequest struct {
	ZielID         string `json:"ziel_id" validate:"required,uuid_oder_leer"`
	AlteAlsVerweis *bool  `json:"alte_als_verweis"`
}

// SchlagwortVerweisRequest legt einen Verweis an: die Schreibweise und ihr Ziel.
type SchlagwortVerweisRequest struct {
	Wort   string `json:"wort"`
	ZielID string `json:"ziel_id" validate:"required,uuid_oder_leer"`
}

// SchlagworteLoeschenRequest nennt die Wörter, die fallen sollen. Jede Kennung ist Pflicht und
// eine UUID — eine leere käme sonst als 500 aus der Datenbank zurück.
type SchlagworteLoeschenRequest struct {
	IDs []string `json:"ids" validate:"required,dive,required,uuid_oder_leer"`
}

// SchlagwortAenderung ist die Antwort der Pflege-Türen: was sich getan hat, als Zahlen.
// Woerter trägt nur das Löschen.
type SchlagwortAenderung struct {
	Wort     string `json:"wort,omitempty"`
	Woerter  int    `json:"woerter,omitempty"`
	Titel    int    `json:"titel"`
	Verweise int    `json:"verweise"`
}

// GetSchlagwortPflegeHandler liefert alle Schlagworte mit Titelzahl, Verweisziel, den
// Verweisen darauf und der Filter-Markierung.
//
// @Summary      List keywords for maintenance
// @Tags         books
// @Produce      json
// @Success      200  {object}  repository.SchlagwortPflegeListe
// @Failure      500  {object}  map[string]string
// @Router       /schlagworte/pflege [get]
func (s *Server) GetSchlagwortPflegeHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		liste, err := repository.SchlagworteZurPflege(r.Context(), s.DB.Pool)
		if err != nil {
			return apierrors.Internal("Schlagworte konnten nicht geladen werden", err)
		}
		RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}

// PutSchlagwortWortHandler gibt einem Schlagwort eine neue Schreibweise.
//
// @Summary      Rename a keyword
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path  string                 true  "Keyword ID"
// @Param        body  body  SchlagwortWortRequest  true  "New spelling"
// @Success      200  {object}  SchlagwortAenderung
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /schlagworte/{id}/wort [put]
func (s *Server) PutSchlagwortWortHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := schlagwortIDAusPfad(r)
		if err != nil {
			return err
		}
		var req SchlagwortWortRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if req.AlteAlsVerweis == nil {
			return errAlteAlsVerweisFehlt()
		}
		wort, verweis, err := repository.BenenneSchlagwortUm(r.Context(), s.DB.Pool, id, req.Wort, *req.AlteAlsVerweis)
		if err != nil {
			return schlagwortPflegeFehler(err)
		}
		antwort := SchlagwortAenderung{Wort: wort}
		if verweis {
			antwort.Verweise = 1
		}
		RespondJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// PutSchlagwortFilterHandler setzt oder nimmt die Filter-Markierung (Portal, Stufe 2).
//
// @Summary      Mark a keyword as portal filter
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path  string                   true  "Keyword ID"
// @Param        body  body  SchlagwortFilterRequest  true  "Filter flag"
// @Success      204
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /schlagworte/{id}/filter [put]
func (s *Server) PutSchlagwortFilterHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := schlagwortIDAusPfad(r)
		if err != nil {
			return err
		}
		var req SchlagwortFilterRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if req.IstFilter == nil {
			return apierrors.BadRequest("ist_filter fehlt", errors.New("feld ist_filter fehlt"))
		}
		if err := repository.SetzeSchlagwortFilter(r.Context(), s.DB.Pool, id, *req.IstFilter); err != nil {
			return schlagwortPflegeFehler(err)
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

// PostSchlagwortZusammenfuehrenHandler hängt die Titel des Wortes an das Ziel und macht
// das Wort zum Verweis darauf oder löscht es (alte_als_verweis).
//
// @Summary      Merge a keyword into another
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path  string                 true  "Keyword ID"
// @Param        body  body  SchlagwortZielRequest  true  "Target keyword"
// @Success      200  {object}  SchlagwortAenderung
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /schlagworte/{id}/zusammenfuehren [post]
func (s *Server) PostSchlagwortZusammenfuehrenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := schlagwortIDAusPfad(r)
		if err != nil {
			return err
		}
		var req SchlagwortZielRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if req.AlteAlsVerweis == nil {
			return errAlteAlsVerweisFehlt()
		}
		titel, err := repository.FuehreSchlagworteZusammen(r.Context(), s.DB.Pool, id, req.ZielID, *req.AlteAlsVerweis)
		if err != nil {
			return schlagwortPflegeFehler(err)
		}
		RespondJSON(w, http.StatusOK, SchlagwortAenderung{Titel: titel})
		return nil
	})
}

// PostSchlagwortVerweisHandler leitet eine Schreibweise auf ein Schlagwort.
//
// @Summary      Create a keyword reference
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        body  body  SchlagwortVerweisRequest  true  "Spelling and target"
// @Success      204
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /schlagworte/verweise [post]
func (s *Server) PostSchlagwortVerweisHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req SchlagwortVerweisRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if err := repository.SetzeSchlagwortVerweis(r.Context(), s.DB.Pool, req.Wort, req.ZielID); err != nil {
			return schlagwortPflegeFehler(err)
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

// PostSchlagworteLoeschenHandler löscht die gewählten Schlagworte, alle oder keins; die Titel
// verlieren sie, Verweise darauf fallen mit. Eine Tür für ein Wort und für viele
// (repository.LoescheSchlagworte). Die Antwort nennt, wie viele Wörter, Titel und Verweise
// es waren.
//
// @Summary      Delete keywords
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        body  body  SchlagworteLoeschenRequest  true  "Keyword IDs"
// @Success      200  {object}  SchlagwortAenderung
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /schlagworte/loeschen [post]
func (s *Server) PostSchlagworteLoeschenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req SchlagworteLoeschenRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		geloescht, err := repository.LoescheSchlagworte(r.Context(), s.DB.Pool, req.IDs)
		if err != nil {
			return schlagwortPflegeFehler(err)
		}
		RespondJSON(w, http.StatusOK, SchlagwortAenderung{
			Woerter: geloescht.Woerter, Titel: geloescht.Titel, Verweise: geloescht.Verweise,
		})
		return nil
	})
}

// errAlteAlsVerweisFehlt ist die Antwort auf einen Körper ohne alte_als_verweis.
func errAlteAlsVerweisFehlt() error {
	return apierrors.BadRequest("alte_als_verweis fehlt", errors.New("feld alte_als_verweis fehlt"))
}

// schlagwortIDAusPfad prüft die Kennung, bevor irgendetwas die Datenbank fragt.
func schlagwortIDAusPfad(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if !kennung.IstUUID(id) {
		return "", apierrors.BadRequest("ungültige Schlagwort-ID", errors.New("keine UUID"))
	}
	return id, nil
}

// schlagwortPflegeFehler übersetzt die fachlichen Fehler der Pflege: verletzte Grenze 400,
// unbekanntes Wort 404, vorhandene Schreibweise und verletzte Regel 409 mit ihrem Text.
func schlagwortPflegeFehler(err error) error {
	switch {
	case errors.Is(err, repository.ErrSchlagwortUngueltig):
		return apierrors.BadRequest(err.Error(), err)
	case errors.Is(err, repository.ErrSchlagwortNichtGefunden):
		return apierrors.NotFound("Schlagwort nicht gefunden", err)
	case errors.Is(err, repository.ErrSchlagwortGibtEs):
		return apierrors.Conflict(err.Error()+" — zum Zusammenlegen „Zusammenführen“ wählen", err)
	case errors.Is(err, repository.ErrSchlagwortRegel):
		return apierrors.Conflict(err.Error(), err)
	default:
		return apierrors.Internal("Schlagwort konnte nicht geändert werden", err)
	}
}
