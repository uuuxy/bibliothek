package repository

import (
	"bibliothek/db"
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// AuditRepository verwaltet revisionssichere Protokollierungen (Audit-Logs)
// sowie administrative Löschungen von Systemressourcen unter Einhaltung von Datenschutzvorgaben (DSGVO).
// Alle Log-Einträge sind schreibgeschützt (Append-Only), um Manipulationen auszuschließen.
type AuditRepository interface {
	// DeleteTitle protokolliert die administrative Löschung eines Buchtitels.
	DeleteTitle(ctx context.Context, titleID string, bearbeiterID string) error
	// DeleteCopy protokolliert die administrative Löschung eines konkreten Buchexemplars.
	DeleteCopy(ctx context.Context, copyID string, bearbeiterID string) error
	// DeleteUser protokolliert die administrative Löschung eines Systembenutzers.
	DeleteUser(ctx context.Context, userID string, bearbeiterID string) error

	// DeleteStudent verschiebt einen Schüler in den Papierkorb (Soft-Delete, wiederherstellbar).
	// Die PII bleibt erhalten — die endgültige DSGVO-Anonymisierung macht PurgeStudent.
	DeleteStudent(ctx context.Context, studentID string, bearbeiterID string, grund string) error

	// PurgeStudent entfernt einen bereits im Papierkorb liegenden Schüler endgültig und
	// DSGVO-konform: Ausleihhistorie anonymisiert, Audit-Logs anonymisiert, bezahlte
	// Schadensfälle gelöscht, Schüler-Datensatz entfernt. Offene Ausleihen oder
	// unbezahlte Schadensfälle blockieren die Löschung (liefern einen Fehler).
	PurgeStudent(ctx context.Context, studentID string, bearbeiterID string) error

	// PurgeAbgaenger ist das Cronjob-Pendant für ehemalige Schüler (nicht im Papierkorb,
	// sondern ist_abgaenger=true). Gleiche DSGVO-Löschung wie PurgeStudent; offene
	// Vorgänge blockieren ebenfalls.
	PurgeAbgaenger(ctx context.Context, studentID string, bearbeiterID string) error

	// StornierungGebuehr protokolliert den Erlass oder die Stornierung einer ausstehenden Gebühr mit Begründung.
	StornierungGebuehr(ctx context.Context, schadensfallID string, bearbeiterID string, betrag float64, grund string) error

	// LogAusleihe protokolliert die erfolgreiche Ausleihe eines Exemplars an einen Schüler oder Lehrer.
	LogAusleihe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error
	// LogRueckgabe protokolliert die Rückgabe eines Exemplars inklusive des bearbeitenden Mitarbeiters.
	LogRueckgabe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error

	// LogSystemAktion protokolliert systemgesteuerte Batch-Prozesse (z. B. automatische Sperrungen oder Bereinigungen).
	LogSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error

	// LogAdminAktion protokolliert kritische Admin-Eingriffe in der Tabelle audit_logs.
	LogAdminAktion(ctx context.Context, adminID string, aktion string, ip string, details map[string]any) error
}

// pgAuditRepository implementiert das AuditRepository für PostgreSQL.
type pgAuditRepository struct {
	db db.PgxPoolIface
}

// NewAuditRepository erzeugt eine neue Instanz des PostgreSQL-basierten Audit-Repositorys.
func NewAuditRepository(db db.PgxPoolIface) AuditRepository {
	return &pgAuditRepository{db: db}
}

// insertAuditLog ist die zentrale Hilfsfunktion, die alle Logeinträge in die Tabelle `audit_log` schreibt.
// Durch die Kapselung in einer Funktion wird ein konsistentes Datenbankschema und eine Append-Only-Semantik erzwungen.
// auditEntry bündelt die fachlichen Felder eines Audit-Log-Eintrags (ohne die
// Infrastruktur-Parameter ctx/tx), damit insertAuditLog nicht neun Einzelargumente führt.
type auditEntry struct {
	Tabelle      string
	Aktion       string
	DatensatzID  string
	BearbeiterID *string
	Akteur       string
	Kontext      *string
	Details      map[string]any
}

func (r *pgAuditRepository) insertAuditLog(ctx context.Context, tx pgx.Tx, e auditEntry) error {
	var detailsJSON []byte
	if e.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(e.Details)
		if err != nil {
			return fmt.Errorf("audit details serialization: %w", err)
		}
	}

	const q = `
		INSERT INTO audit_log
		  (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, kontext, details)
		VALUES ($1, $2, $3::uuid, $4, $5, $6, $7)
	`
	_, err := tx.Exec(ctx, q,
		e.Tabelle, e.Aktion, e.DatensatzID,
		e.BearbeiterID, e.Akteur, e.Kontext,
		func() interface{} {
			if detailsJSON == nil {
				return nil
			}
			return string(detailsJSON)
		}(),
	)
	return err
}

// LogAdminAktion schreibt einen Eintrag in die Tabelle audit_logs für systemweite oder kritische Admin-Eingriffe.
func (r *pgAuditRepository) LogAdminAktion(ctx context.Context, adminID string, aktion string, ip string, details map[string]any) error {
	var detailsJSON []byte
	if details != nil {
		var err error
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			detailsJSON = []byte("{}")
		}
	} else {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse, zeitstempel) 
		VALUES ($1, $2, $3::jsonb, $4, CURRENT_TIMESTAMP)
	`
	var adminPtr *string
	if adminID != "" {
		adminPtr = &adminID
	}
	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}

	_, err := r.db.Exec(ctx, query, adminPtr, aktion, string(detailsJSON), ipPtr)
	return err
}
