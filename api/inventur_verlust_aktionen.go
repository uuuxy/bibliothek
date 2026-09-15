package api

// inventur_verlust_aktionen.go — Handlungsspielraum für den Fehlbestandsbericht:
// ein als Verlust gebuchtes Exemplar kann wiedergefunden (zurück in Umlauf) oder
// endgültig gelöscht werden. Vorher war der Bericht reine Anzeige ohne Folgehandlung.

import (
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
		gefunden, befund, err := invRepo.MarkiereVerlustAlsGefunden(ctx, id, claims.UserID)
		if errors.Is(err, repository.ErrExemplarNichtGefunden) {
			// Zwischen Lesen und Zurückholen verschwunden — kein Serverfehler, nichts gebucht.
			return apierrors.NotFound("kein offener Verlust mit dieser ID", err)
		}
		if err != nil {
			return apierrors.Internal("Fund konnte nicht verbucht werden", err)
		}
		if !gefunden {
			return apierrors.NotFound("kein offener Verlust mit dieser ID", nil)
		}
		if err := tx.Commit(ctx); err != nil {
			return apierrors.Internal("Transaktion konnte nicht abgeschlossen werden", err)
		}
		// Die Folge für die Forderung steht in der Antwort: Der Bericht sagt, dass die
		// Forderung storniert ist — oder dass die Aufsicht zu informieren ist, weil der
		// Bescheid dort schon liegt (dann bleibt sie offen).
		RespondJSON(w, http.StatusOK, VerlustGefundenResponse{
			Status:                "ok",
			StornierteForderungen: befund.StornierteForderungen,
			StornierterBetrag:     befund.StornierterBetrag,
			Hinweis:               befund.AufsichtHinweis(),
		})
		return nil
	})
}

// VerlustGefundenResponse ist die Antwort der Fund-Meldung: was mit der Forderung
// geschah, die das Buch abgerechnet hatte.
type VerlustGefundenResponse struct {
	Status                string  `json:"status"`
	StornierteForderungen int     `json:"stornierte_forderungen"`
	StornierterBetrag     float64 `json:"stornierter_betrag"`
	// Hinweis ist leer, wenn nichts zu tun ist.
	Hinweis string `json:"hinweis,omitempty"`
}

// verlustLoeschenRequest benennt die endgültig zu löschenden Exemplare.
type verlustLoeschenRequest struct {
	ExemplarIDs []string `json:"exemplar_ids" validate:"omitempty,dive,uuid_oder_leer"`
}

// InventurVerlusteLoeschenHandler löscht als Verlust gebuchte Exemplare endgültig.
// Bewusst NICHT der allgemeine Lösch-Weg (der soft löscht, siehe DeleteCopyHandler) —
// diese Route rührt ausschliesslich Exemplare an, die schon als VERLUST gelten.
// @Router /buecher/exemplare/verlust-endgueltig-loeschen [post]
func (s *Server) InventurVerlusteLoeschenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req verlustLoeschenRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
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
		geloeschteIDs, err := invRepo.EndgueltigLoescheVerlustExemplare(ctx, req.ExemplarIDs, claims.UserID)
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
		// Beide Angaben: die Zahl für die Meldung, die IDs, damit die Oberfläche genau
		// die Zeilen entfernt, die wirklich weg sind — und nicht die, die sie angefragt
		// hatte. Nie nil, damit der Client [] statt null bekommt.
		if geloeschteIDs == nil {
			geloeschteIDs = []string{}
		}
		RespondJSON(w, http.StatusOK, map[string]any{
			"geloescht":      len(geloeschteIDs),
			"geloeschte_ids": geloeschteIDs,
		})
		return nil
	})
}
