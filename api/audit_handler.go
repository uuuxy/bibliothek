package api

// audit_handler.go — Handler for the immutable audit log.
// The audit trail records all sensitive delete/cancel operations performed by staff.

import (
	"net/http"
	"strconv"
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
	// Akteur unterscheidet 'USER' von 'SYSTEM'. Ohne dieses Feld stünde eine
	// Systemaktion in der Anzeige als Eintrag ganz ohne Urheber da — nicht
	// unterscheidbar von einem Datenfehler.
	Akteur string `json:"akteur"`
}

// auditLogMaxZeilen begrenzt das Logbuch auf die jüngsten Einträge.
//
// Vorher holte die Abfrage die GESAMTE Tabelle. Auf dem Prüfstand mit 247.000 Zeilen
// waren das 72 MB JSON in einer Antwort: Der Server lieferte sie in 0,6 s, aber der
// Browser musste sie parsen und ebenso viele Tabellenzeilen bauen — die Seite kam nicht
// mehr zum Vorschein, der E2E-Test lief in den Timeout. Das Logbuch wächst im Betrieb
// unbegrenzt weiter, der Fall tritt also zwangsläufig ein.
//
// 1000 wie beim Schwester-Endpunkt GetAdminAuditLogsHandler — der hatte die Grenze von
// Anfang an, dieser war der Ausreißer.
const auditLogMaxZeilen = 1000

// GetAuditLogsHandler returns logs of immutable security events.
// @Summary      Get audit logs
// @Description  Retrieves the most recent 1000 records of the system's audit trail, newest first.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {array}   AuditLogEntry
// @Failure      500  {object}  map[string]string
// @Router       /audit [get]
func (s *Server) GetAuditLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// LEFT JOIN, nicht JOIN: Systemgesteuerte Vorgänge (DSGVO-Bereinigung, Backups,
		// automatische Sperren) schreiben ohne bearbeiter_id und mit akteur='SYSTEM'
		// (repository/audit_system.go). Der frühere Inner Join hat genau diese Zeilen
		// aussortiert — ein Prüfprotokoll, das ausgerechnet die unbeaufsichtigten
		// Aktionen verschweigt, ist das Gegenteil von dem, wofür es geführt wird.
		//
		// COALESCE ist dabei Pflicht, nicht Kosmetik: bearbeiter_id, vorname und nachname
		// sind ab jetzt nullbar, die Go-Felder sind es nicht — ohne COALESCE stünde hier
		// ein "cannot scan NULL" statt des Logbuchs.
		query := `
			SELECT l.id, l.tabelle, l.aktion, l.datensatz_id, l.timestamp,
			       COALESCE(l.bearbeiter_id::text, ''),
			       COALESCE(b.vorname, ''), COALESCE(b.nachname, ''),
			       l.akteur
			FROM audit_log l
			LEFT JOIN benutzer b ON l.bearbeiter_id = b.id
			ORDER BY l.timestamp DESC
			LIMIT ` + strconv.Itoa(auditLogMaxZeilen)
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		logs := []AuditLogEntry{}
		for rows.Next() {
			var l AuditLogEntry
			err := rows.Scan(&l.ID, &l.Tabelle, &l.Aktion, &l.DatensatzID, &l.Timestamp,
				&l.BearbeiterID, &l.BearbeiterVorname, &l.BearbeiterNachname, &l.Akteur)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			logs = append(logs, l)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, logs)
	}
}

