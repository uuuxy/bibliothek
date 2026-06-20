package api

import (
	"errors"
	"net/http"
	"strconv"
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

// UpdateSignatureHandler updates name and description of an existing signature.
// PUT /api/signatures/{id}
func (s *Server) UpdateSignatureHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültige ID"))
			return
		}

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

		res, err := s.DB.Pool.Exec(ctx,
			`UPDATE signatures SET name = $1, description = $2 WHERE id = $3`,
			req.Name, req.Description, id,
		)
		if err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "23505") {
				apierrors.SendHTTPError(w, http.StatusConflict, errors.New("eine Signatur mit diesem Namen existiert bereits"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if res.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Signatur nicht gefunden"))
			return
		}

		RespondJSON(w, http.StatusOK, Signature{ID: id, Name: req.Name, Description: req.Description})
	}
}

// DeleteSignatureHandler deletes a signature.
// Returns 409 Conflict if books are still assigned to this signature.
// DELETE /api/signatures/{id}
func (s *Server) DeleteSignatureHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültige ID"))
			return
		}

		ctx := r.Context()

		res, err := s.DB.Pool.Exec(ctx, `DELETE FROM signatures WHERE id = $1`, id)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "violates foreign key constraint") || strings.Contains(errStr, "23503") {
				apierrors.SendHTTPError(w, http.StatusConflict,
					errors.New("Signatur kann nicht gelöscht werden, da ihr noch Bücher zugeordnet sind"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if res.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Signatur nicht gefunden"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
