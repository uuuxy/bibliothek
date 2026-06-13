package api

import (
	"encoding/json"
	"net/http"

	"bibliothek/apierrors"
)

// DecodeJSON decodes the JSON request body into the provided target struct.
// It returns true if decoding was successful. If an error occurs, it automatically
// responds with a 400 Bad Request HTTP error and returns false.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request, target *T) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

// RespondJSON encodes the payload as JSON and sends it with the given HTTP status code.
// It sets the Content-Type header to application/json.
func RespondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// RespondSuccess is a convenience function that sends a {"status": "success"} JSON response.
func RespondSuccess(w http.ResponseWriter) {
	RespondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
