package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
)

// Signature represents a library signature (category/location) record.
type Signature struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetSignaturesHandler returns all signatures, sorted alphabetically.
// GET /api/signatures
func (s *Server) GetSignaturesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := s.DB.Pool.Query(ctx,
			`SELECT id, name, COALESCE(description, '') FROM signatures ORDER BY name ASC`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		result := []Signature{}
		for rows.Next() {
			var sig Signature
			if err := rows.Scan(&sig.ID, &sig.Name, &sig.Description); err != nil {
				continue
			}
			result = append(result, sig)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, result)
	}
}

// CreateSignatureHandler creates a new signature.
// POST /api/signatures  { "name": "...", "description": "..." }
func (s *Server) CreateSignatureHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("name ist erforderlich"))
			return
		}

		ctx := r.Context()

		var newID int
		err := s.DB.Pool.QueryRow(ctx,
			`INSERT INTO signatures (name, description) VALUES ($1, $2) RETURNING id`,
			req.Name, req.Description,
		).Scan(&newID)
		if err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "23505") {
				apierrors.SendHTTPError(w, http.StatusConflict, errors.New("eine Signatur mit diesem Namen existiert bereits"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusCreated, Signature{ID: newID, Name: req.Name, Description: req.Description})
	}
}
