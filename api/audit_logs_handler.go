package api

import (
	"net/http"
	"time"

	"bibliothek/apierrors"
)

type AdminAuditLogEntry struct {
	ID          string    `json:"id"`
	AdminID     *string   `json:"admin_id"`
	AdminName   string    `json:"admin_name"`
	Aktion      string    `json:"aktion"`
	Details     any       `json:"details"`
	IpAdresse   string    `json:"ip_adresse"`
	Zeitstempel time.Time `json:"zeitstempel"`
}

// GetAdminAuditLogsHandler fetches the latest 1000 admin audit logs in descending order.
func (s *Server) GetAdminAuditLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		query := `
			SELECT 
				a.id, a.admin_id, coalesce(b.vorname || ' ' || b.nachname, 'System/Unbekannt'),
				a.aktion, a.details, coalesce(a.ip_adresse, ''), a.zeitstempel
			FROM audit_logs a
			LEFT JOIN benutzer b ON a.admin_id = b.id
			ORDER BY a.zeitstempel DESC
			LIMIT 1000
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		var logs []AdminAuditLogEntry
		for rows.Next() {
			var l AdminAuditLogEntry
			if err := rows.Scan(&l.ID, &l.AdminID, &l.AdminName, &l.Aktion, &l.Details, &l.IpAdresse, &l.Zeitstempel); err != nil {
				continue
			}
			logs = append(logs, l)
		}

		if logs == nil {
			logs = []AdminAuditLogEntry{}
		}

		RespondJSON(w, http.StatusOK, logs)
	}
}
