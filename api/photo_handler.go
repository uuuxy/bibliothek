package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/internal/service"
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
			if errors.Is(err, service.ErrFotoLeserUnbekannt) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			if errors.Is(err, service.ErrFotoUnlesbar) {
				apierrors.SendHTTPErrorMitMeldung(w, http.StatusBadRequest,
					"Das Bild ließ sich nicht lesen. Bitte die Aufnahme wiederholen.", err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Über den JSON-Schreiber, nicht von Hand zusammengesetzt: Die Ausweisnummer steht in
		// der Adresse und darf Zeichen tragen, die in JSON maskiert werden müssen.
		RespondJSON(w, http.StatusOK, map[string]string{"status": "success", "url": photoURL})
	}
}
