package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/internal/crypto"

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
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if !strings.HasPrefix(req.PhotoData, "data:image/jpeg;base64,") {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("payload must be a base64 encoded JPEG data URL"))
			return
		}

		ctx := r.Context()

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

		// 3. Verschlüsseln der Foto-Bytes
		encryptedData, err := crypto.Encrypt(imgBytes)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der fotostrukturierung: %v", err))
			return
		}

		// 4. In der Datenbank abspeichern (Upsert in schueler_fotos)
		query := `
			INSERT INTO schueler_fotos (schueler_id, foto_encrypted)
			VALUES ($1, $2)
			ON CONFLICT (schueler_id) DO UPDATE SET 
				foto_encrypted = EXCLUDED.foto_encrypted,
				aktualisiert_am = CURRENT_TIMESTAMP
		`
		_, err = s.DB.Pool.Exec(ctx, query, id, encryptedData)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim speichern des fotos in der db: %v", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		photoURL := fmt.Sprintf("/api/schueler/%s/photo", barcodeID)
		_, _ = w.Write([]byte(`{"status":"success","url":"` + photoURL + `"}`))
	}
}
