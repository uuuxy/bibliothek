package theke

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// Nachbuch-Meldungen (Migration 117): Was beim Nachbuchen vom Offline-Scan abwich, steht
// hier, bis jemand es quittiert. Liste und Quittieren tragen view_students, weil die Zeilen
// Ausleiher und Vorbesitzer nennen; der Zähler fürs Band ist eine Zahl ohne Personenbezug
// und gehört jeder Theken-Rolle (perform_actions). Die SQL liegt in repository/.

// NachbuchMeldungen liefert die offenen Meldungen (?alle=1: auch quittierte).
// @Summary      Nachbuch-Meldungen
// @Tags         theke
// @Produce      json
// @Param        alle query bool false "auch quittierte"
// @Success      200 {array} repository.NachbuchMeldung
// @Router       /action/nachbuch-meldungen [get]
func (t *Tueren) NachbuchMeldungen() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		nurOffen := r.URL.Query().Get("alle") != "1"
		liste, err := repository.LadeNachbuchMeldungen(r.Context(), t.DB.Pool, nurOffen)
		if err != nil {
			return apierrors.Internal("Nachbuch-Meldungen konnten nicht gelesen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}

// NachbuchMeldungenAnzahl ist der Zähler fürs Band an der Theke.
// @Summary      Anzahl offener Nachbuch-Meldungen
// @Tags         theke
// @Produce      json
// @Success      200 {object} map[string]int
// @Router       /action/nachbuch-meldungen/anzahl [get]
func (t *Tueren) NachbuchMeldungenAnzahl() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		n, err := repository.ZaehleOffeneNachbuchMeldungen(r.Context(), t.DB.Pool)
		if err != nil {
			return apierrors.Internal("Nachbuch-Meldungen konnten nicht gezählt werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, map[string]int{"offen": n})
		return nil
	})
}

// NachbuchMeldungQuittieren quittiert eine offene Meldung — nur eine offene: Eine
// schon quittierte oder unbekannte ist 404, kein stiller Erfolg.
// @Summary      Nachbuch-Meldung quittieren
// @Tags         theke
// @Produce      json
// @Param        id path string true "Meldungs-ID"
// @Success      200 {object} map[string]string
// @Router       /action/nachbuch-meldungen/{id}/quittieren [post]
func (t *Tueren) NachbuchMeldungQuittieren() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("nicht angemeldet", errors.New("missing session information"))
		}
		// {id} ist durch uuidPfadMiddleware schon als UUID geprüft.
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}
		err := repository.QuittiereNachbuchMeldung(r.Context(), t.DB.Pool, id, claims.UserID)
		if errors.Is(err, repository.ErrNachbuchMeldungNichtOffen) {
			return apierrors.NotFound("keine offene Nachbuch-Meldung mit dieser ID", err)
		}
		if err != nil {
			return apierrors.Internal("Nachbuch-Meldung konnte nicht quittiert werden", err)
		}
		// Eine Meldung weniger: Das Band der anderen Arbeitsplätze zeigt sonst weiter die
		// alte Zahl, bis dort jemand neu anmeldet.
		t.meldeMeldungsstand()
		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return nil
	})
}
