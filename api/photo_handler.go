package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bibliothek/apierrors"
	"github.com/jackc/pgx/v5"
)

// UploadPhotoRequest holds the base64 encoded photo payload.
type UploadPhotoRequest struct {
	PhotoData string `json:"photo_data"` // data:image/jpeg;base64,...
}

// UploadStudentPhotoHandler decodes and registers a student's webcam passport photo.
func (s *Server) UploadStudentPhotoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing student ID parameter"))
			return
		}

		var req UploadPhotoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if !strings.HasPrefix(req.PhotoData, "data:image/jpeg;base64,") {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("payload must be a base64 encoded JPEG data URL"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		// 1. Resolve student's barcode ID from database
		var barcodeID string
		err := s.DB.Pool.QueryRow(ctx, "SELECT barcode_id FROM schueler WHERE id = $1", id).Scan(&barcodeID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. Decode base64 image data
		base64Data := strings.TrimPrefix(req.PhotoData, "data:image/jpeg;base64,")
		imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		// 3. Save to disk folder uploads/fotos/{barcodeID}.jpg
		dir := filepath.Join("uploads", "fotos")
		if err := os.MkdirAll(dir, 0750); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		fileName := filepath.Join(dir, fmt.Sprintf("%s.jpg", barcodeID))
		if err := os.WriteFile(fileName, imgBytes, 0600); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","url":"/uploads/fotos/` + barcodeID + `.jpg"}`))
	}
}
