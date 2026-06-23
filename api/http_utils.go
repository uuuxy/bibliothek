package api

import (
	"encoding/json"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"

	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

// DecodeAndValidate decodes the JSON request body and validates the struct.
func DecodeAndValidate[T any](w http.ResponseWriter, r *http.Request, target *T) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return false
	}
	if err := Validate.Struct(target); err != nil {
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
	httpresp.Encode(w, payload)
}

// RespondSuccess is a convenience function that sends a {"status": "success"} JSON response.
func RespondSuccess(w http.ResponseWriter) {
	RespondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
