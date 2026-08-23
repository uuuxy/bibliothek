package api

// inventur_verlust_aktionen.go — Handlungsspielraum für den Fehlbestandsbericht:
// ein als Verlust gebuchtes Exemplar kann wiedergefunden (zurück in Umlauf) oder
// endgültig gelöscht werden. Vorher war der Bericht reine Anzeige ohne Folgehandlung.

import (
	"encoding/json"
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// InventurVerlustGefundenHandler markiert ein Exemplar der aktuellen Session als beim
// Nachsuchen wiedergefunden und bringt es zurück in Umlauf.
// @Router /buecher/exemplare/{id}/gefunden [post]
func (s *Server) InventurVerlustGefundenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("exemplar-id fehlt", nil)
		}
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("nicht angemeldet", errors.New("missing session information"))
		}

		ctx := r.Context()
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			return apierrors.Internal("Transaktion konnte nicht gestartet werden", err)
		}
		defer db.SafeRollback(ctx, tx)

		invRepo := repository.NewInventoryRepository(tx)
		gefunden, err := invRepo.MarkiereVerlustAlsGefunden(ctx, id, claims.UserID)
		if err != nil {
			return apierrors.Internal("Fund konnte nicht verbucht werden", err)
		}
		if !gefunden {
			return apierrors.NotFound("kein offener Verlust mit dieser ID", nil)
		}
		if err := tx.Commit(ctx); err != nil {
			return apierrors.Internal("Transaktion konnte nicht abgeschlossen werden", err)
		}
		RespondSuccess(w)
		return nil
	})
}

// verlustLoeschenRequest benennt die endgültig zu löschenden Exemplare.
type verlustLoeschenRequest struct {
	ExemplarIDs []string `json:"exemplar_ids"`
}

// InventurVerlusteLoeschenHandler löscht als Verlust gebuchte Exemplare endgültig.
// Bewusst NICHT der allgemeine Lösch-Weg (der soft löscht, siehe DeleteCopyHandler) —
// diese Route rührt ausschliesslich Exemplare an, die schon als VERLUST gelten.
// @Router /buecher/exemplare/verlust-endgueltig-loeschen [post]
func (s *Server) InventurVerlusteLoeschenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req verlustLoeschenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return apierrors.BadRequest("ungültiger Request-Body", err)
		}
		if len(req.ExemplarIDs) == 0 {
			return apierrors.BadRequest("exemplar_ids fehlt oder ist leer", nil)
		}
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("nicht angemeldet", errors.New("missing session information"))
		}

		ctx := r.Context()
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			return apierrors.Internal("Transaktion konnte nicht gestartet werden", err)
		}
		defer db.SafeRollback(ctx, tx)

		invRepo := repository.NewInventoryRepository(tx)
		anzahl, err := invRepo.EndgueltigLoescheVerlustExemplare(ctx, req.ExemplarIDs, claims.UserID)
		if err != nil {
			// 409 statt 500: Ein gebundenes Exemplar ist eine Lage, kein Störfall. Als
			// Internal ersetzte der Sanitizer den Text durch „interner Datenbankfehler"
			// (apierrors.SendHTTPError) — die Bedienung erführe nie, WELCHES Exemplar
			// warum im Weg steht. Wrap gibt die Message eines APIError unverändert aus.
			if errors.Is(err, repository.ErrVerlustNochGebunden) {
				return apierrors.Conflict(err.Error(), err)
			}
			return apierrors.Internal("Endgültiges Löschen fehlgeschlagen", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return apierrors.Internal("Transaktion konnte nicht abgeschlossen werden", err)
		}
		RespondJSON(w, http.StatusOK, map[string]int{"geloescht": anzahl})
		return nil
	})
}
