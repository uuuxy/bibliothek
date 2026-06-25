package api

// copy_admin.go — Handlers for physical book copy and title administration:
// damage notes, deletion, decommissioning and copy listing.
// Part of the admin layer; authentication/authorization is enforced in router.go.

import (
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// DeleteCopyHandler removes a physical copy from circulation.
// @Summary      Delete physical book copy
// @Description  Deletes a specific physical book copy by its ID from the library catalog and registers the deletion in the audit trail.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book copy ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/exemplare/{id} [delete]
func (s *Server) DeleteCopyHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		ctx := r.Context()

		err := auditRepo.DeleteCopy(ctx, id, claims.UserID)
		if err != nil {
			if err.Error() == "Exemplar ist aktuell noch verliehen!" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			} else {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			}
			return
		}

		RespondSuccess(w)
	}
}

// DeleteTitleHandler deletes a book title and all its physical copies from the database, creating an audit log.
// @Summary      Delete book title
// @Description  Deletes a specific book title and all associated physical copies, registering the deletion in the audit trail.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book title ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/titel/{id} [delete]
func (s *Server) DeleteTitleHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		err := auditRepo.DeleteTitle(ctx, id, claims.UserID)
		if err != nil {
			if len(err.Error()) > 21 && err.Error()[:22] == "Löschen fehlgeschlagen:" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			} else {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			}
			return
		}

		RespondSuccess(w)
	}
}

// GetTitleCopiesHandler lists all physical copies belonging to a book title.
// @Summary      List copies for a title
// @Description  Retrieves all physical book copies associated with a given title ID, including availability status.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book title ID (UUID)"
// @Success      200  {array}   object
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/titel/{id}/exemplare [get]
func (s *Server) GetTitleCopiesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		query := `
			SELECT e.id, e.barcode_id, coalesce(e.zustand_notiz, ''), e.ist_ausleihbar, e.ist_ausgesondert,
			       (SELECT COUNT(*) FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL) = 0 AS ist_verfuegbar
			FROM buecher_exemplare e
			WHERE e.titel_id = $1
			ORDER BY e.ist_ausgesondert ASC, e.barcode_id
		`
		rows, err := s.DB.Pool.Query(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// CopyResponse is the per-copy DTO returned by this handler.
		type CopyResponse struct {
			ID              string `json:"id"`
			BarcodeID       string `json:"barcode_id"`
			ZustandNotiz    string `json:"zustand_notiz"`
			IstAusleihbar   bool   `json:"ist_ausleihbar"`
			IstAusgesondert bool   `json:"ist_ausgesondert"`
			IstVerfuegbar   bool   `json:"ist_verfuegbar"`
		}

		copies := []CopyResponse{}
		for rows.Next() {
			var cp CopyResponse
			if err := rows.Scan(&cp.ID, &cp.BarcodeID, &cp.ZustandNotiz, &cp.IstAusleihbar, &cp.IstAusgesondert, &cp.IstVerfuegbar); err == nil {
				copies = append(copies, cp)
			}
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, copies)
	}
}

// TitleBorrower represents a student who is currently borrowing a copy of a title.
type TitleBorrower struct {
	Vorname         string    `json:"schueler_name"`
	Nachname        string    `json:"schueler_nachname"`
	Klasse          string    `json:"klasse"`
	SchuelerBarcode string    `json:"schueler_barcode"`
	ExemplarBarcode string    `json:"exemplar_barcode"`
	AusgeliehenAm   time.Time `json:"ausgeliehen_am"`
	RueckgabeFrist  time.Time `json:"rueckgabe_frist"`
}

// GetTitleBorrowersHandler lists all active borrowers for a book title.
// @Router       /buecher/titel/{id}/ausleiher [get]
func (s *Server) GetTitleBorrowersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		query := `
			SELECT s.vorname, s.nachname, s.klasse, s.barcode_id, e.barcode_id, a.ausgeliehen_am, a.rueckgabe_frist
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			JOIN schueler s ON a.schueler_id = s.id
			WHERE e.titel_id = $1 AND a.rueckgabe_am IS NULL
			ORDER BY a.rueckgabe_frist ASC
		`
		rows, err := s.DB.Pool.Query(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		borrowers := []TitleBorrower{}
		for rows.Next() {
			var b TitleBorrower
			if err := rows.Scan(&b.Vorname, &b.Nachname, &b.Klasse, &b.SchuelerBarcode, &b.ExemplarBarcode, &b.AusgeliehenAm, &b.RueckgabeFrist); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			borrowers = append(borrowers, b)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, borrowers)
	}
}

// TitleHistory represents a historical loan for a copy of a title.
type TitleHistory struct {
	Vorname         string     `json:"schueler_name"`
	Nachname        string     `json:"schueler_nachname"`
	Klasse          string     `json:"klasse"`
	ExemplarBarcode string     `json:"exemplar_barcode"`
	AusgeliehenAm   time.Time  `json:"ausgeliehen_am"`
	RueckgabeAm     *time.Time `json:"rueckgabe_am"` // Can be null if still borrowed
}

// GetTitleHistoryHandler lists all historical loans for a book title.
// @Router       /buecher/titel/{id}/historie [get]
func (s *Server) GetTitleHistoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		query := `
			SELECT s.vorname, s.nachname, s.klasse, e.barcode_id, a.ausgeliehen_am, a.rueckgabe_am
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			JOIN schueler s ON a.schueler_id = s.id
			WHERE e.titel_id = $1
			ORDER BY a.ausgeliehen_am DESC
			LIMIT 200
		`
		rows, err := s.DB.Pool.Query(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		history := []TitleHistory{}
		for rows.Next() {
			var h TitleHistory
			if err := rows.Scan(&h.Vorname, &h.Nachname, &h.Klasse, &h.ExemplarBarcode, &h.AusgeliehenAm, &h.RueckgabeAm); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			history = append(history, h)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, history)
	}
}
