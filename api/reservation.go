package api

import (
	"errors"
	"fmt"
	"log"
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

// OffeneKlassensatzReservierungenHandler liefert die Warteschlange für das
// Kollegiums-Portal: wer vor der eigenen Reservierung steht (Titel, Klasse, Menge) —
// bewusst ohne Personendaten des Anfordernden. Eigener Endpunkt, weil die volle
// Liste (GET /api/reservierungen/klassensatz) hinter view_orders liegt und dem
// Kollegium verschlossen bleibt.
// GET /api/reservierungen/klassensatz/offen
func (s *Server) OffeneKlassensatzReservierungenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewReservationRepository(s.DB.Pool)
		offene, err := repo.OffeneKlassensatzReservierungen(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, offene)
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
		erledigt, err := repo.ErledigeKlassensatzReservierung(r.Context(), id)

		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if erledigt == nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, pgx.ErrNoRows)
			return
		}

		// Bereit-Mail an die anfragende Lehrkraft — Betreiber-Entscheidung 16.08.2026:
		// KEIN neuer Menuepunkt, das aktive Signal genuegt. Best effort: Die Buecher
		// liegen physisch bereit; ein kaputter Mailversand darf das Abschliessen nicht
		// rueckgaengig erscheinen lassen (nur Logzeile). Ohne Konto-Adresse (per SQL
		// angelegt, Konto geloescht) gibt es schlicht keine Mail.
		if erledigt.AnfragendeMail != nil && *erledigt.AnfragendeMail != "" {
			if err := SendEmail(MailRequest{
				To:      *erledigt.AnfragendeMail,
				Subject: fmt.Sprintf("Ihr Klassensatz liegt bereit: %s", erledigt.TitelName),
				Body: fmt.Sprintf("Ihre Klassensatz-Reservierung ist fertig zusammengestellt und liegt in der Bibliothek zur Abholung bereit.\n\n"+
					"  Titel:  %s\n  Klasse: %s\n  Anzahl: %d Exemplare\n\n"+
					"Diese Mail wurde automatisch beim Abschliessen der Reservierung verschickt.",
					erledigt.TitelName, erledigt.Klasse, erledigt.Anzahl),
			}); err != nil {
				log.Printf("Klassensatz-Bereit-Mail an %s fehlgeschlagen: %v", *erledigt.AnfragendeMail, err)
			}
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
