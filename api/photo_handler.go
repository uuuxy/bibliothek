package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/internal/service"
	"bibliothek/pkg/httpresp"
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

		photoURL, err := service.UploadStudentPhoto(ctx, s.DB.Pool, id, req.PhotoData)
		if err != nil {
			if err.Error() == "schüler nicht gefunden" {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Write(w, []byte(`{"status":"success","url":"`+photoURL+`"}`))
	}
}
