package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

// DeleteCopyHandler removes a physical copy from circulation.
func (s *Server) DeleteCopyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `DELETE FROM buecher_exemplare WHERE id = $1`
		_, err := s.DB.Pool.Exec(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// DeleteTitleHandler deletes a book title and all its physical copies from the database, creating an audit log.
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
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// DeleteUserHandler deletes a user and logs it in the audit log.
func (s *Server) DeleteUserHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing user ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := auditRepo.DeleteUser(ctx, id, claims.UserID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// GetGraduatesHandler lists graduating students with unreturned books.
func (s *Server) GetGraduatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Get students of graduating classes who currently hold open loans
		query := `
			SELECT DISTINCT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt
			FROM schueler s
			JOIN ausleihen a ON s.id = a.schueler_id
			WHERE s.klasse IN ('9h', '10r', '13')
			  AND a.rueckgabe_am IS NULL
			ORDER BY s.klasse, s.nachname
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		students := []any{}
		for rows.Next() {
			var id, barcode, vorname, nachname, klasse string
			var abgaengerJahr int
			var gesperrt bool
			if err := rows.Scan(&id, &barcode, &vorname, &nachname, &klasse, &abgaengerJahr, &gesperrt); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			students = append(students, map[string]any{
				"id":             id,
				"barcode_id":     barcode,
				"vorname":        vorname,
				"nachname":       nachname,
				"klasse":         klasse,
				"abgaenger_jahr": abgaengerJahr,
				"ist_gesperrt":   gesperrt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(students)
	}
}

// AuditLogEntry represents a joined row in the audit log table.
type AuditLogEntry struct {
	ID                string    `json:"id"`
	Tabelle           string    `json:"tabelle"`
	Aktion            string    `json:"aktion"`
	DatensatzID       string    `json:"datensatz_id"`
	Timestamp         time.Time `json:"timestamp"`
	BearbeiterID      string    `json:"bearbeiter_id"`
	BearbeiterVorname string    `json:"bearbeiter_vorname"`
	BearbeiterNachname string   `json:"bearbeiter_nachname"`
}

// GetAuditLogsHandler returns logs of immutable security events.
func (s *Server) GetAuditLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			SELECT l.id, l.tabelle, l.aktion, l.datensatz_id, l.timestamp, l.bearbeiter_id, b.vorname, b.nachname
			FROM audit_log l
			JOIN benutzer b ON l.bearbeiter_id = b.id
			ORDER BY l.timestamp DESC
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		logs := []AuditLogEntry{}
		for rows.Next() {
			var l AuditLogEntry
			err := rows.Scan(&l.ID, &l.Tabelle, &l.Aktion, &l.DatensatzID, &l.Timestamp, &l.BearbeiterID, &l.BearbeiterVorname, &l.BearbeiterNachname)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			logs = append(logs, l)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logs)
	}
}

// GetTitleCopiesHandler lists all physical copies belonging to a book title.
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
			SELECT id, barcode_id, coalesce(zustand_notiz, ''), ist_ausleihbar
			FROM buecher_exemplare
			WHERE titel_id = $1
			ORDER BY barcode_id
		`
		rows, err := s.DB.Pool.Query(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		type CopyResponse struct {
			ID           string `json:"id"`
			BarcodeID    string `json:"barcode_id"`
			ZustandNotiz string `json:"zustand_notiz"`
			IstAusleihbar bool   `json:"ist_ausleihbar"`
		}

		copies := []CopyResponse{}
		for rows.Next() {
			var cp CopyResponse
			if err := rows.Scan(&cp.ID, &cp.BarcodeID, &cp.ZustandNotiz, &cp.IstAusleihbar); err == nil {
				copies = append(copies, cp)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copies)
	}
}
