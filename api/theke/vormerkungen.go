package theke

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// vormerkungAnfrage is the body for POST /api/vormerkungen.
type vormerkungAnfrage struct {
	TitelID    string `json:"titel_id" validate:"required,uuid_oder_leer"`
	Notiz      string `json:"notiz,omitempty"`
	SchuelerID string `json:"schueler_id,omitempty" validate:"omitempty,uuid_oder_leer"`
}

// Vormerkungen handles GET /api/vormerkungen?titel_id=...
func (t *Tueren) Vormerkungen(vormerkungsAbfragen repository.VormerkungsAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()

		titelID, err := tuer.LeseUUIDParameter(r, "titel_id")
		if err != nil {
			return apierrors.BadRequest(err.Error(), nil)
		}
		schuelerID, err := tuer.LeseUUIDParameter(r, "schueler_id")
		if err != nil {
			return apierrors.BadRequest(err.Error(), nil)
		}

		liste, err := vormerkungsAbfragen.LadeListe(ctx, titelID, schuelerID)
		if err != nil {
			// Ohne Titel- oder Schüler-Filter gibt es bewusst keinen Voll-Abzug aller
			// Namen — das ist ein Bedienfehler (400), kein Serverfehler.
			if errors.Is(err, repository.ErrVormerkungFilterFehlt) {
				return apierrors.BadRequest(err.Error(), err)
			}
			return apierrors.Internal("Fehler beim Abrufen der Vormerkungen", err)
		}
		if liste == nil {
			liste = []repository.Vormerkung{}
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}

// VormerkungAnlegen handles POST /api/vormerkungen.
func (t *Tueren) VormerkungAnlegen(vormerkungsAbfragen repository.VormerkungsAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req vormerkungAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}

		ctx := r.Context()

		id, err := vormerkungsAbfragen.LegeAn(ctx, req.TitelID, req.Notiz, req.SchuelerID)
		if err != nil {
			// Drei fachliche Konflikte (409), kein Serverfehler: Der Schüler hat den Titel
			// bereits selbst ausgeliehen und darf ihn nicht zusätzlich vormerken, er steht
			// längst auf der Liste (UNIQUE(titel_id, schueler_id)) — oder die Id gehört
			// keinem Schüler, und die Warteschlange würde sie nie bedienen. Jedes Mal soll
			// die Theke den Satz lesen, der die Lage beschreibt, statt einer
			// Störungsmeldung, nach der unklar bleibt, ob die erste Vormerkung noch gilt.
			if errors.Is(err, repository.ErrTitelBereitsAusgeliehen) ||
				errors.Is(err, repository.ErrVormerkungBereitsVorhanden) ||
				errors.Is(err, repository.ErrVormerkungNurFuerSchueler) {
				return apierrors.Conflict(err.Error(), err)
			}
			return apierrors.Internal("Fehler beim Erstellen der Vormerkung", err)
		}

		tuer.AntworteJSON(w, http.StatusCreated, map[string]string{"id": id})
		return nil
	})
}

// VormerkungLoeschen handles DELETE /api/vormerkungen/{id}.
func (t *Tueren) VormerkungLoeschen(vormerkungsAbfragen repository.VormerkungsAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("ID fehlt", nil)
		}

		ctx := r.Context()

		if err := vormerkungsAbfragen.Loesche(ctx, id); err != nil {
			if errors.Is(err, repository.ErrVormerkungNichtGefunden) {
				return apierrors.NotFound("Vormerkung nicht gefunden", err)
			}
			return apierrors.Internal("Fehler beim Löschen der Vormerkung", err)
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Schreibe(w, []byte(`{"status":"gelöscht"}`))
		return nil
	})
}
