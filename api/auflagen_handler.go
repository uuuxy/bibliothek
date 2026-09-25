package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// Auflagen eines Schulbuchs (Migration 148, docs/OFFEN.md 4.18). Die Regeln stehen in
// repository/auflagen.go; die Titelmaske erreicht sie über die Türen hier (edit_books), der
// Bestellkorb bekommt in Stufe 4 eine eigene Route mit create_orders — beide über dieselben
// Funktionen, wie bei den Schlagworten.

// TitelAuflagen ist die Antwort aller drei Türen: die Auflagen des Buchs, zu dem der Titel
// gehört, die neueste zuerst. Ein Titel ohne andere Auflage steht allein darin.
type TitelAuflagen struct {
	ID       string               `json:"id"`
	Auflagen []repository.Auflage `json:"auflagen"`
}

// AuflageZuordnenRequest nennt den Titel, der eine andere Auflage desselben Buchs ist.
type AuflageZuordnenRequest struct {
	TitelID string `json:"titel_id" validate:"required,uuid_oder_leer"`
}

// GetTitelAuflagenHandler liefert die Auflagen des Buchs, zu dem ein Titel gehört, mit
// ihrem Bestand.
//
// @Summary      List the editions of a title's book
// @Tags         books
// @Produce      json
// @Param        id   path      string  true  "Title ID"
// @Success      200  {object}  TitelAuflagen
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /buecher/titel/{id}/auflagen [get]
func (s *Server) GetTitelAuflagenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := titelIDAusPfad(r)
		if err != nil {
			return err
		}
		auflagen, err := repository.AuflagenDesTitels(r.Context(), s.DB.Pool, id)
		if err != nil {
			return auflagenFehler(err, "gelesen")
		}
		RespondJSON(w, http.StatusOK, TitelAuflagen{ID: id, Auflagen: auflagen})
		return nil
	})
}

// PostTitelAuflagenHandler fasst einen Titel mit einer anderen Auflage desselben Buchs
// zusammen. Gehört einer der beiden schon zu einem Buch mit weiteren Auflagen, kommen alle
// zusammen. Zwei Routen, ein Handler: die Titelmaske (…/auflagen, edit_books) und der
// Bestellbedarf (…/neue-auflage, create_orders — „Der Besteller legt das Werk an",
// docs/OFFEN.md 4.18, entschieden am 23.09.2026).
//
// @Summary      Join another edition to a title's book
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path      string                  true  "Title ID"
// @Param        body  body      AuflageZuordnenRequest  true  "The other edition"
// @Success      200   {object}  TitelAuflagen
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /buecher/titel/{id}/auflagen [post]
// @Router       /buecher/titel/{id}/neue-auflage [post]
func (s *Server) PostTitelAuflagenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := titelIDAusPfad(r)
		if err != nil {
			return err
		}
		var req AuflageZuordnenRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		auflagen, err := repository.FasseAuflagenZusammen(r.Context(), s.DB.Pool, id, req.TitelID)
		if err != nil {
			return auflagenFehler(err, "gespeichert")
		}
		RespondJSON(w, http.StatusOK, TitelAuflagen{ID: id, Auflagen: auflagen})
		return nil
	})
}

// DeleteTitelAuflagenHandler nimmt einen Titel aus seinem Buch; die anderen Auflagen
// bleiben zusammen, solange es mindestens zwei sind.
//
// @Summary      Detach a title from its book's editions
// @Tags         books
// @Produce      json
// @Param        id   path      string  true  "Title ID"
// @Success      200  {object}  TitelAuflagen
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /buecher/titel/{id}/auflagen [delete]
func (s *Server) DeleteTitelAuflagenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := titelIDAusPfad(r)
		if err != nil {
			return err
		}
		auflagen, err := repository.LoeseAuflage(r.Context(), s.DB.Pool, id)
		if err != nil {
			return auflagenFehler(err, "gespeichert")
		}
		RespondJSON(w, http.StatusOK, TitelAuflagen{ID: id, Auflagen: auflagen})
		return nil
	})
}

// titelIDAusPfad liest die Titel-ID aus dem Pfad; keine UUID ist 400.
func titelIDAusPfad(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if !kennung.IstUUID(id) {
		return "", apierrors.BadRequest("ungültige Titel-ID", errors.New("keine UUID"))
	}
	return id, nil
}

// auflagenFehler übersetzt die fachlichen Fehler des Repositorys: eine verletzte Regel ist
// 400 mit ihrem Text, ein unbekannter Titel 404, alles andere 500.
func auflagenFehler(err error, handlung string) error {
	switch {
	case errors.Is(err, repository.ErrAuflageUngueltig):
		return apierrors.BadRequest(err.Error(), err)
	case errors.Is(err, repository.ErrTitelNichtGefunden):
		return apierrors.NotFound("Titel nicht gefunden", err)
	default:
		return apierrors.Internal("Auflagen konnten nicht "+handlung+" werden", err)
	}
}
