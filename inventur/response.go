package inventur

import (
	"encoding/json"
	"errors"
	"net/http"

	"bibliothek/apierrors"
)

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload) //nolint:errcheck
}

func writeError(writer http.ResponseWriter, status int, message string) {
	apierrors.SendHTTPError(writer, status, errors.New(message))
}
