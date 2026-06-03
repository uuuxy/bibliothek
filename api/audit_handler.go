package api

// audit_handler.go — Handler for the immutable audit log.
// The audit trail records all sensitive delete/cancel operations performed by staff.

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"bibliothek/apierrors"
)

// AuditLogEntry represents a joined row in the audit log table.
type AuditLogEntry struct {
	ID                 string    `json:"id"`
	Tabelle            string    `json:"tabelle"`
	Aktion             string    `json:"aktion"`
	DatensatzID        string    `json:"datensatz_id"`
	Timestamp          time.Time `json:"timestamp"`
	BearbeiterID       string    `json:"bearbeiter_id"`
	BearbeiterVorname  string    `json:"bearbeiter_vorname"`
	BearbeiterNachname string    `json:"bearbeiter_nachname"`
}

// GetAuditLogsHandler returns logs of immutable security events.
// @Summary      Get audit logs
// @Description  Retrieves all immutable records in the system's audit trail, including deletions and cancellations.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {array}   AuditLogEntry
// @Failure      500  {object}  map[string]string
// @Router       /audit [get]
func (s *Server) GetAuditLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			SELECT l.id, l.tabelle, l.aktion, l.datensatz_id, l.timestamp, l.bearbeiter_id, b.vorname, b.nachname
			FROM audit_log l
			JOIN benutzer b ON l.bearbeiter_id = b.id
			ORDER BY l.timestamp DESC
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		logs := []AuditLogEntry{}
		for rows.Next() {
			var l AuditLogEntry
			err := rows.Scan(&l.ID, &l.Tabelle, &l.Aktion, &l.DatensatzID, &l.Timestamp, &l.BearbeiterID, &l.BearbeiterVorname, &l.BearbeiterNachname)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			logs = append(logs, l)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logs)
	}
}
