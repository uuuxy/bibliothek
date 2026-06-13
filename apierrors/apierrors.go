package apierrors

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// SendHTTPError logs the detailed internal error to the server console and returns a sanitized JSON error to the client.
func SendHTTPError(w http.ResponseWriter, status int, internalErr error) {
	if internalErr != nil {
		log.Printf("API Error [HTTP %d]: %v (path: %s)", status, internalErr, "unknown")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// Default to status text
	msg := http.StatusText(status)
	if msg == "" {
		msg = "Unknown Error"
	}

	// For user-facing errors (non-500), default to the internalErr string unless it contains SQL/DB details
	if status != http.StatusInternalServerError && internalErr != nil {
		msg = internalErr.Error()
	}

	// Check if the error is database-related (SQL structure, constraint violation, DB driver logs)
	isDBError := false
	if internalErr != nil {
		errMsg := strings.ToLower(internalErr.Error())
		if strings.Contains(errMsg, "pgx") ||
			strings.Contains(errMsg, "pgconn") ||
			strings.Contains(errMsg, "sql:") ||
			strings.Contains(errMsg, "unique constraint") ||
			strings.Contains(errMsg, "foreign key") ||
			strings.Contains(errMsg, "duplicate key") ||
			strings.Contains(errMsg, "violates") ||
			strings.Contains(errMsg, "insert into") ||
			strings.Contains(errMsg, "select ") ||
			strings.Contains(errMsg, "update ") ||
			strings.Contains(errMsg, "delete ") {
			isDBError = true
		}
	}

	// Strictly sanitize all internal server errors and database errors
	if status == http.StatusInternalServerError || isDBError {
		if internalErr != nil {
			errStr := internalErr.Error()
			if strings.Contains(errStr, "23505") || strings.Contains(errStr, "unique_violation") || strings.Contains(errStr, "duplicate key") {
				msg = "Ein Eintrag mit diesen eindeutigen Eigenschaften existiert bereits."
			} else if strings.Contains(errStr, "23503") || strings.Contains(errStr, "foreign_key_violation") {
				msg = "Diese Aktion kann nicht durchgeführt werden, da verknüpfte Daten existieren."
			} else {
				msg = "Ein interner Datenbankfehler ist aufgetreten."
			}
		} else {
			msg = "Internal Server Error"
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
