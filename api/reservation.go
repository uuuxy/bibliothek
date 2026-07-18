package api

import (
	"errors"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// KlassensatzReservierungRequest is the payload for a class-set reservation.
type KlassensatzReservierungRequest struct {
	TitelID string `json:"titel_id"`
	Klasse  string `json:"klasse"`
	Anzahl  int    `json:"anzahl"`
	Notiz   string `json:"notiz,omitempty"`
}

// CreateKlassensatzReservierungHandler lets a LEHRER submit a class-set reservation.
// POST /api/reservierungen/klassensatz
func (s *Server) CreateKlassensatzReservierungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KlassensatzReservierungRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if !validateKlassensatzRequest(w, &req) {
			return
		}

		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("fehlende Sitzungsinformationen"))
			return
		}

		ctx := r.Context()
		repo := repository.NewReservationRepository(s.DB.Pool)

		// Verify the title exists.
		exists, err := repo.CheckTitleExists(ctx, req.TitelID)
		if err != nil || !exists {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("buchtitel nicht gefunden"))
			return
		}

		// Bestandsdeckelung: Mehr Exemplare zu reservieren, als die Bibliothek überhaupt
		// besitzt, erzeugt eine dauerhaft unerfüllbare Aufgabe im Dashboard (Ghost-Order).
		// Die Wunschmenge wird deshalb gegen den physischen Bestand (nicht ausgesondert)
		// geprüft.
		bestand, err := repo.CountTitleStock(ctx, req.TitelID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if req.Anzahl > bestand {
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				fmt.Errorf("nur %d Exemplare im Bestand — %d können nicht reserviert werden", bestand, req.Anzahl))
			return
		}

		newID, err := repo.CreateKlassensatzReservierung(ctx, req.TitelID, req.Klasse, req.Anzahl, nullableString(req.Notiz), claims.UserID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusCreated, map[string]string{"id": newID, "status": "erstellt"})
	}
}

// GetKlassensatzReservierungenHandler lists all pending class-set reservations for admins.
// GET /api/reservierungen/klassensatz
func (s *Server) GetKlassensatzReservierungenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewReservationRepository(s.DB.Pool)
		result, err := repo.GetKlassensatzReservierungen(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Ensure we don't return null for empty slices
		if result == nil {
			result = []repository.KlassensatzReservierung{}
		}

		RespondJSON(w, http.StatusOK, result)
	}
}

// GetKlassensatzReservierungenAnzahlHandler returns the count of open reservations (for red badge).
// GET /api/reservierungen/klassensatz/anzahl
func (s *Server) GetKlassensatzReservierungenAnzahlHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewReservationRepository(s.DB.Pool)
		count, err := repo.GetKlassensatzReservierungenAnzahl(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]int{"anzahl": count})
	}
}

// ErledigeKlassensatzReservierungHandler marks a class-set reservation as done.
// PUT /api/reservierungen/klassensatz/{id}/erledigen
func (s *Server) ErledigeKlassensatzReservierungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("id fehlt"))
			return
		}

		repo := repository.NewReservationRepository(s.DB.Pool)
		rowsAffected, err := repo.ErledigeKlassensatzReservierung(r.Context(), id)

		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if rowsAffected == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, pgx.ErrNoRows)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// validateKlassensatzRequest prüft die Pflichtfelder und normalisiert die Anzahl
// (Default 1, Obergrenze 200). ok=false: die Fehlerantwort wurde bereits geschrieben.
func validateKlassensatzRequest(w http.ResponseWriter, req *KlassensatzReservierungRequest) bool {
	if req.TitelID == "" || req.Klasse == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("titel_id und klasse sind erforderlich"))
		return false
	}
	if req.Anzahl <= 0 {
		req.Anzahl = 1
	}
	if req.Anzahl > 200 {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("anzahl darf 200 nicht überschreiten"))
		return false
	}
	return true
}

// nullableString converts an empty string to nil for nullable DB columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
