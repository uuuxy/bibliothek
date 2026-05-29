package apierrors

import (
	"encoding/json"
	"log"
	"net/http"
)

// SendHTTPError logs the detailed internal error to the server console and returns a sanitized JSON error to the client.
func SendHTTPError(w http.ResponseWriter, status int, internalErr error) {
	if internalErr != nil {
		log.Printf("API Error [HTTP %d]: %v", status, internalErr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// Resolve the standard status text for the status code (e.g. "Bad Request", "Unauthorized")
	msg := http.StatusText(status)
	if msg == "" {
		msg = "Unknown Error"
	}

	// Ensure all internal server errors are strictly sanitized
	if status == http.StatusInternalServerError {
		msg = "Internal Server Error"
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
