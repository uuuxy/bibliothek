package apierrors

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// APIError is a structured error for the HTTP API that implements the error interface.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"error"`
	Err        error  `json:"-"` // Internal error for logging
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap allows unwrapping the internal error.
func (e *APIError) Unwrap() error {
	return e.Err
}

// New creates a new APIError.
func New(statusCode int, message string, err error) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// Common error constructors
func NotFound(message string, err error) *APIError {
	if message == "" {
		message = "Ressource nicht gefunden"
	}
	return New(http.StatusNotFound, message, err)
}

func BadRequest(message string, err error) *APIError {
	if message == "" {
		message = "Ungültige Anfrage"
	}
	return New(http.StatusBadRequest, message, err)
}

func Internal(message string, err error) *APIError {
	if message == "" {
		message = "Ein interner Serverfehler ist aufgetreten"
	}
	return New(http.StatusInternalServerError, message, err)
}

func Unauthorized(message string, err error) *APIError {
	if message == "" {
		message = "Nicht autorisiert"
	}
	return New(http.StatusUnauthorized, message, err)
}

// Conflict signalisiert einen Zustandskonflikt (409): Die Anfrage war gültig, kollidiert
// aber mit dem inzwischen geänderten Serverzustand (z. B. ein zwischenzeitlich neu
// ausgeliehenes Exemplar). Der Client soll neu laden, nicht wiederholen.
func Conflict(message string, err error) *APIError {
	if message == "" {
		message = "Konflikt mit dem aktuellen Stand"
	}
	return New(http.StatusConflict, message, err)
}

// APIHandler is a signature for HTTP handlers that return an error.
type APIHandler func(w http.ResponseWriter, r *http.Request) error

// Wrap converts an APIHandler into a standard http.HandlerFunc.
// If the handler returns an error, it is properly formatted and sent to the client.
func Wrap(h APIHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				if apiErr.StatusCode >= 500 {
					log.Printf("API Error [HTTP %d]: %v", apiErr.StatusCode, apiErr.Err)
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(apiErr.StatusCode)
				_ = json.NewEncoder(w).Encode(apiErr) //nolint:errcheck
			} else {
				// Fallback to the existing SendHTTPError logic for generic errors
				SendHTTPError(w, http.StatusInternalServerError, err)
			}
		}
	}
}

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
	isDBError := internalErr != nil && istDatenbankFehler(strings.ToLower(internalErr.Error()))

	// Strictly sanitize all internal server errors and database errors
	if status == http.StatusInternalServerError || isDBError {
		msg = sanitizeInternalError(internalErr)
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

// istDatenbankFehler erkennt DB-nahe Fehlermeldungen (SQL, Constraints, Treiber-Logs),
// die niemals ungefiltert an den Client gelangen dürfen. errMsg muss lowercase sein.
func istDatenbankFehler(errMsg string) bool {
	return strings.Contains(errMsg, "pgx") ||
		strings.Contains(errMsg, "pgconn") ||
		strings.Contains(errMsg, "sql:") ||
		strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "foreign key") ||
		strings.Contains(errMsg, "duplicate key") ||
		strings.Contains(errMsg, "violates") ||
		strings.Contains(errMsg, "insert into") ||
		strings.Contains(errMsg, "select ") ||
		strings.Contains(errMsg, "update ") ||
		strings.Contains(errMsg, "delete ")
}

// sanitizeInternalError liefert die neutrale, client-sichere Meldung für interne bzw.
// DB-Fehler (bekannte Constraint-Verletzungen werden fachlich übersetzt).
func sanitizeInternalError(internalErr error) string {
	if internalErr == nil {
		return "Internal Server Error"
	}
	errStr := internalErr.Error()
	switch {
	case strings.Contains(errStr, "23505") || strings.Contains(errStr, "unique_violation") || strings.Contains(errStr, "duplicate key"):
		return "Ein Eintrag mit diesen eindeutigen Eigenschaften existiert bereits."
	case strings.Contains(errStr, "23503") || strings.Contains(errStr, "foreign_key_violation"):
		return "Diese Aktion kann nicht durchgeführt werden, da verknüpfte Daten existieren."
	default:
		return "Ein interner Datenbankfehler ist aufgetreten."
	}
}
