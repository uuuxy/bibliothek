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

// RecentTransaction represents a recent checkout or return.
type RecentTransaction struct {
	Aktion          string    `json:"aktion"`
	SchuelerVorname string    `json:"schueler_vorname"`
	SchuelerNachname string   `json:"schueler_nachname"`
	SchuelerBarcode string    `json:"schueler_barcode"`
	Buchtitel       string    `json:"buchtitel"`
	Timestamp       time.Time `json:"timestamp"`
}

// GetRecentTransactionsHandler returns the 15 most recent checkouts/returns.
// @Router       /transactions/recent [get]
func (s *Server) GetRecentTransactionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			SELECT 
				a.aktion,
				COALESCE(s.vorname, b.vorname, 'Unbekannt'),
				COALESCE(s.nachname, b.nachname, 'Unbekannt'),
				COALESCE(s.barcode_id, b.barcode_id, ''),
				COALESCE(t.titel, 'Unbekanntes Buch'),
				a.timestamp
			FROM audit_log a
			LEFT JOIN schueler s ON (a.details->>'schueler_id') IS NOT NULL AND (a.details->>'schueler_id') != '' AND s.id = (a.details->>'schueler_id')::uuid
			LEFT JOIN benutzer b ON (a.details->>'benutzer_id') IS NOT NULL AND (a.details->>'benutzer_id') != '' AND b.id = (a.details->>'benutzer_id')::uuid
			LEFT JOIN buecher_exemplare e ON (a.details->>'exemplar_id') IS NOT NULL AND (a.details->>'exemplar_id') != '' AND e.id = (a.details->>'exemplar_id')::uuid
			LEFT JOIN buecher_titel t ON t.id = e.titel_id
			WHERE a.tabelle = 'ausleihen' AND a.aktion IN ('CHECKOUT', 'RETURN')
			ORDER BY a.timestamp DESC
			LIMIT 15
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		txs := []RecentTransaction{}
		for rows.Next() {
			var tx RecentTransaction
			if err := rows.Scan(&tx.Aktion, &tx.SchuelerVorname, &tx.SchuelerNachname, &tx.SchuelerBarcode, &tx.Buchtitel, &tx.Timestamp); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			txs = append(txs, tx)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(txs)
	}
}
