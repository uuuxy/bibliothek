package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// InventurFinishRequest benennt die abzuschließende Session.
type InventurFinishRequest struct {
	SessionID string `json:"session_id"`
}

// InventurFinishResponse contains the outcome statistics of the completed inventory.
type InventurFinishResponse struct {
	VerlorenGemeldet int `json:"verloren_gemeldet"`

	// Fehlbestand nennt die betroffenen Exemplare, nicht nur ihre Anzahl. Vorher stand
	// hier allein die Zahl — mit „47 Verluste" kann niemand ins Regal gehen und nachsehen.
	Fehlbestand []repository.InventurVerlust `json:"fehlbestand"`
}

// InventurFinishHandler schließt eine Inventur-Session ab: Alle im Scope erwarteten,
// aber in DIESER Session nicht erfassten (und nicht verliehenen) Exemplare werden als
// Verlust markiert. Der Fortschritt paralleler Sessions bleibt unberührt.
//
// @Summary      Finalize an inventory session
// @Description  Marks scope items not scanned in this session as lost and closes it.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      InventurFinishRequest   true  "Session to finalize"
// @Success      200   {object}  InventurFinishResponse
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/finish [post]
func (s *Server) InventurFinishHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InventurFinishRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if req.SessionID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("session_id fehlt"))
			return
		}

		ctx := r.Context()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer db.SafeRollback(ctx, tx)

		invRepo := repository.NewInventoryRepository(tx)

		// Session in derselben Transaktion sperren/prüfen: verhindert doppelten
		// Abschluss und liefert die signature_id für den Scope des Verlust-Updates.
		session, err := invRepo.LadeInventurSession(ctx, req.SessionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("keine laufende Inventur zu dieser Session"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		count, err := invRepo.FinishInventurSession(ctx, session.ID, session.Scope())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// In DERSELBEN Transaktion gelesen, vor dem Commit: Danach ist der Fehlbestand
		// zwar dauerhaft gespeichert, aber ein Fehler beim Lesen dürfte den Abschluss
		// nicht mehr zurückrollen — die Aussonderung wäre dann schon geschrieben.
		fehlbestand, err := invRepo.LadeInventurVerluste(ctx, session.ID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventurFinishResponse{
			VerlorenGemeldet: count,
			Fehlbestand:      fehlbestand,
		})
	}
}

// InventurFehlbestandHandler liefert den Fehlbestand einer bereits abgeschlossenen
// Inventur.
//
// Der Abschluss nennt die Liste zwar sofort, aber genau in dem Moment steht man mit dem
// Scanner im Regal und will sie nicht am Bildschirm lesen. Ohne diesen Weg wäre sie nach
// dem Schliessen der Ansicht verloren — und nachrechnen kann man sie nicht, weil die
// Exemplare durch die Aussonderung aus dem Scope fallen.
//
// @Summary      Fehlbestand einer abgeschlossenen Inventur
// @Tags         inventur
// @Produce      json
// @Param        session_id  query  string  true  "Session"
// @Success      200  {array}  repository.InventurVerlust
// @Router       /inventur/fehlbestand [get]
func (s *Server) InventurFehlbestandHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			return apierrors.BadRequest("session_id fehlt", nil)
		}

		invRepo := repository.NewInventoryRepository(s.DB.Pool)
		liste, err := invRepo.LadeInventurVerluste(r.Context(), sessionID)
		if err != nil {
			return apierrors.Internal("Fehlbestand konnte nicht geladen werden", err)
		}

		RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}
