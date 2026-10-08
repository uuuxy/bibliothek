package api

import (
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// AdminAuditLogEntry ist eine Zeile des Admin-Prüfprotokolls, wie die Oberfläche sie
// anzeigt. AdminID ist bewusst nullbar: Systemgetriebene Einträge (Cron, Migration)
// haben keinen Bearbeiter, sollen aber trotzdem im Protokoll stehen.
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

		zeilen, err := repository.ListeAdminProtokoll(ctx, s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		logs := []AdminAuditLogEntry{}
		for _, z := range zeilen {
			logs = append(logs, AdminAuditLogEntry(z))
		}

		RespondJSON(w, http.StatusOK, logs)
	}
}
