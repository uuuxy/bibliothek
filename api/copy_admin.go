package api

// copy_admin.go — Handlers for physical book copy and title administration:
// damage notes, deletion, decommissioning and copy listing.
// Part of the admin layer; authentication/authorization is enforced in router.go.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// DamageNoteRequest holds the payload for updating a copy's damage note.
type DamageNoteRequest struct {
	Note string `json:"note"`
}

// UpdateDamageNoteHandler updates the physical condition note of a book copy.
// @Summary      Update damage note
// @Description  Updates the custom damage or condition note text of a physical book copy.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "Book copy ID (UUID)"
// @Param        body  body      DamageNoteRequest  true  "Damage note payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/schadensnotiz [post]
func (s *Server) UpdateDamageNoteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		var req DamageNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			UPDATE buecher_exemplare
			SET zustand_notiz = $1, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $2
		`
		_, err := s.DB.Pool.Exec(ctx, query, req.Note, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// UpdateBarcodeRequest holds the payload for updating a copy's barcode.
type UpdateBarcodeRequest struct {
	Barcode string `json:"barcode"`
}

// UpdateCopyBarcodeHandler updates the barcode of a physical book copy.
// @Summary      Update copy barcode
// @Description  Updates the barcode of a physical book copy, replacing placeholders like AUTO-.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Book copy ID (UUID)"
// @Param        body  body      UpdateBarcodeRequest  true  "New barcode payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/barcode [put]
func (s *Server) UpdateCopyBarcodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		var req UpdateBarcodeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if req.Barcode == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("barcode cannot be empty"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			UPDATE buecher_exemplare
			SET barcode_id = $1, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $2
		`
		_, err := s.DB.Pool.Exec(ctx, query, req.Barcode, id)
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
				apierrors.SendHTTPError(w, http.StatusConflict, errors.New("dieser Barcode wird bereits von einem anderen Exemplar verwendet"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// UpdateStatusRequest holds the payload for updating a copy's status.
type UpdateStatusRequest struct {
	IstAusleihbar bool   `json:"ist_ausleihbar"`
	IstAusgesondert bool `json:"ist_ausgesondert"`
	ZustandNotiz  string `json:"zustand_notiz"`
}

// UpdateCopyStatusHandler updates the status of a physical book copy.
// @Summary      Update copy status
// @Description  Updates the status (ist_ausleihbar, ist_ausgesondert) and the condition note of a physical book copy.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Book copy ID (UUID)"
// @Param        body  body      UpdateStatusRequest   true  "New status payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/status [put]
func (s *Server) UpdateCopyStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		var req UpdateStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Wenn ein Buch manuell auf "Verfügbar" gesetzt wird, zwingend Notizen und Ausgesondert-Flag löschen
		if req.IstAusleihbar {
			req.ZustandNotiz = ""
			req.IstAusgesondert = false
		}

		query := `
			UPDATE buecher_exemplare
			SET ist_ausleihbar = $1, ist_ausgesondert = $2, zustand_notiz = $3, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $4
		`
		_, err := s.DB.Pool.Exec(ctx, query, req.IstAusleihbar, req.IstAusgesondert, req.ZustandNotiz, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

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

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := auditRepo.DeleteCopy(ctx, id, claims.UserID)
		if err != nil {
			if err.Error() == "Exemplar ist aktuell noch verliehen!" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			} else {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
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

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := auditRepo.DeleteTitle(ctx, id, claims.UserID)
		if err != nil {
			if len(err.Error()) > 21 && err.Error()[:22] == "Löschen fehlgeschlagen:" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			} else {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
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

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copies)
	}
}

// AussondernCopyHandler marks a physical copy as decommissioned (ausgesondert).
// Decommissioned copies are hidden from catalog, kiosk, and inventory but kept for statistics.
// @Summary      Decommission a book copy
// @Description  Marks a physical copy as decommissioned: sets ist_ausgesondert=true and ist_ausleihbar=false.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book copy ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/exemplare/{id}/aussondern [post]
func (s *Server) AussondernCopyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		_, err := s.DB.Pool.Exec(ctx, `
			UPDATE buecher_exemplare
			SET ist_ausgesondert = true, ist_ausleihbar = false, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1
		`, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// TitleBorrower represents a student who is currently borrowing a copy of a title.
type TitleBorrower struct {
	Vorname         string     `json:"schueler_name"`
	Nachname        string     `json:"schueler_nachname"`
	Klasse          string     `json:"klasse"`
	SchuelerBarcode string     `json:"schueler_barcode"`
	ExemplarBarcode string     `json:"exemplar_barcode"`
	AusgeliehenAm   time.Time  `json:"ausgeliehen_am"`
	RueckgabeFrist  time.Time  `json:"rueckgabe_frist"`
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

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(borrowers)
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

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(history)
	}
}
