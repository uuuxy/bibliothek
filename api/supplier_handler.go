package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
)

// SupplierResponse represents the supplier data sent to the client.
type SupplierResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	CustomerNumber string    `json:"customerNumber"`
	ErstelltAm     time.Time `json:"erstellt_am"`
}

// CreateSupplierRequest holds the payload for creating a new supplier.
type CreateSupplierRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	CustomerNumber string `json:"customerNumber"`
}

// ListSuppliersHandler returns a list of all suppliers.
func (s *Server) ListSuppliersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rows, err := s.DB.Pool.Query(ctx, "SELECT id, name, email, kundennummer, erstellt_am FROM lieferanten ORDER BY name ASC")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		suppliers := []SupplierResponse{}
		for rows.Next() {
			var sup SupplierResponse
			if err := rows.Scan(&sup.ID, &sup.Name, &sup.Email, &sup.CustomerNumber, &sup.ErstelltAm); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			suppliers = append(suppliers, sup)
		}

		RespondJSON(w, http.StatusOK, suppliers)
	}
}

// CreateSupplierHandler adds a new supplier.
func (s *Server) CreateSupplierHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSupplierRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		if req.Name == "" || req.Email == "" || req.CustomerNumber == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("name, email and customerNumber are required"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var newID string
		var erstelltAm time.Time
		err := s.DB.Pool.QueryRow(ctx, `
			INSERT INTO lieferanten (name, email, kundennummer)
			VALUES ($1, $2, $3)
			RETURNING id, erstellt_am
		`, req.Name, req.Email, req.CustomerNumber).Scan(&newID, &erstelltAm)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusCreated, SupplierResponse{
			ID:             newID,
			Name:           req.Name,
			Email:          req.Email,
			CustomerNumber: req.CustomerNumber,
			ErstelltAm:     erstelltAm,
		})
	}
}

// DeleteSupplierHandler removes a supplier.
func (s *Server) DeleteSupplierHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Go 1.22+ routing path parameter resolution
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing supplier ID"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		tag, err := s.DB.Pool.Exec(ctx, "DELETE FROM lieferanten WHERE id = $1", id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if tag.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
