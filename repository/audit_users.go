package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// DeleteUser löscht einen Systembenutzer endgültig aus der Datenbank und erfasst die Löschung im Audit-Log.
func (r *pgAuditRepository) DeleteUser(ctx context.Context, userID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Benutzerdaten vor dem Löschen sichern
	var vorname, nachname, email, rolle string
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(email,''), coalesce(rolle::text,'')
		 FROM benutzer WHERE id = $1`,
		userID,
	).Scan(&vorname, &nachname, &email, &rolle)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot user for audit: %w", err)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM benutzer WHERE id = $1", userID); err != nil {
		return err
	}

	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "benutzer", Aktion: "DELETE", DatensatzID: userID,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Details: map[string]any{"vorname": vorname, "nachname": nachname, "email": email, "rolle": rolle},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteStudent verschiebt einen Schüler in den Papierkorb (Soft-Delete): deleted_at
// wird gesetzt und der Datensatz gesperrt. Die personenbezogenen Daten (PII) bleiben
// zunächst erhalten, damit ein versehentliches Löschen per RestoreStudentHandler
// rückgängig gemacht werden kann.
//
// ACHTUNG: Dies ist KEINE DSGVO-Löschung — die PII (Name, Adresse, Ausleihhistorie,
// Audit-Logs, Schadensfälle) bleibt bestehen. Die endgültige Anonymisierung/Löschung
// macht PurgeStudent (endgültiges Entfernen aus dem Papierkorb).
//
// Der Aufruf ist über die HTTP-API (bearbeiterID = Benutzer-UUID) und den Cronjob
// (bearbeiterID = "" → SYSTEM) möglich.
func (r *pgAuditRepository) DeleteStudent(ctx context.Context, studentID string, bearbeiterID string, grund string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Daten für das Audit-Log vor dem Löschen sichern
	var vorname, nachname, klasse, barcodeID string
	var abgaengerJahr int
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(klasse,''),
		        coalesce(barcode_id,''), coalesce(abgaenger_jahr, 0)
		 FROM schueler WHERE id = $1`,
		studentID,
	).Scan(&vorname, &nachname, &klasse, &barcodeID, &abgaengerJahr)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot student for audit: %w", err)
	}

	// Soft-Delete durchführen anstatt physisch zu löschen
	tag, err := tx.Exec(ctx, `UPDATE schueler SET deleted_at = CURRENT_TIMESTAMP, ist_gesperrt = true, block_reason = 'Systematisch gelöscht' WHERE id = $1`, studentID)
	if err != nil {
		return fmt.Errorf("soft-deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("student %s not found", studentID)
	}

	// Akteur ermitteln (entweder manueller Admin-User oder automatische System-Bereinigung)
	var akteur string
	var bearbeiterPtr *string
	if bearbeiterID != "" {
		akteur = "USER"
		bearbeiterPtr = &bearbeiterID
	} else {
		akteur = "SYSTEM"
	}

	kontext := "Soft-Delete Routine"

	// Protokolleintrag schreiben
	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "schueler", Aktion: "UPDATE", DatensatzID: studentID,
		BearbeiterID: bearbeiterPtr, Akteur: akteur, Kontext: &kontext,
		Details: map[string]any{
			"vorname":        vorname,
			"nachname":       nachname,
			"klasse":         klasse,
			"barcode_id":     barcodeID,
			"abgaenger_jahr": abgaengerJahr,
			"grund":          grund,
			"geloescht_am":   time.Now().UTC().Format(time.RFC3339),
			"action":         "soft_delete",
		},
	}); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// blockiereBeiOffenenVorgaengen verhindert das endgültige Löschen, solange Bücher
// draußen sind oder eine Gebühr offen ist — in beiden Fällen läuft noch ein
// berechtigtes Interesse, das die Aufbewahrung rechtfertigt.
func blockiereBeiOffenenVorgaengen(ctx context.Context, tx pgx.Tx, studentID string) error {
	var offeneAusleihen int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, studentID).Scan(&offeneAusleihen); err != nil {
		return fmt.Errorf("checking open loans: %w", err)
	}
	if offeneAusleihen > 0 {
		return fmt.Errorf("endgültiges Löschen blockiert: %d offene Ausleihe(n)", offeneAusleihen)
	}
	var offeneSchaeden int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`, studentID).Scan(&offeneSchaeden); err != nil {
		return fmt.Errorf("checking unpaid damages: %w", err)
	}
	if offeneSchaeden > 0 {
		return fmt.Errorf("endgültiges Löschen blockiert: %d unbezahlte(r) Schadensfall/-fälle", offeneSchaeden)
	}
	return nil
}

// entferneSchuelerPIIUndLoesche ist die gemeinsame DSGVO-Löschung für PurgeStudent
// (manueller Papierkorb) und PurgeAbgaenger (Cronjob): Ausleihhistorie anonymisieren
// (schueler_id = NULL — beide Entleiher NULL ist laut check_loan_borrower erlaubt),
// bezahlte Schadensfälle löschen, Schüler-Audit-Details anonymisieren, Datensatz
// entfernen (FK-CASCADE räumt Fotos + Vormerkungen), Löschung ohne PII protokollieren.
func (r *pgAuditRepository) entferneSchuelerPIIUndLoesche(ctx context.Context, tx pgx.Tx, studentID, bearbeiterID, kontextText string) error {
	if _, err := tx.Exec(ctx, `UPDATE ausleihen SET schueler_id = NULL WHERE schueler_id = $1`, studentID); err != nil {
		return fmt.Errorf("anonymizing loans: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schadensfaelle WHERE schueler_id = $1`, studentID); err != nil {
		return fmt.Errorf("deleting damages: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE audit_log
		SET details = jsonb_build_object('anonymisiert', true, 'grund', 'DSGVO-Löschung')
		WHERE tabelle = 'schueler' AND datensatz_id = $1`, studentID); err != nil {
		return fmt.Errorf("anonymizing audit logs: %w", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM schueler WHERE id = $1`, studentID)
	if err != nil {
		return fmt.Errorf("deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("student %s not found", studentID)
	}

	var bearbeiterPtr *string
	akteur := "SYSTEM"
	if bearbeiterID != "" {
		akteur = "USER"
		bearbeiterPtr = &bearbeiterID
	}
	if err := r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "schueler", Aktion: "DELETE", DatensatzID: studentID,
		BearbeiterID: bearbeiterPtr, Akteur: akteur, Kontext: &kontextText,
		Details: map[string]any{"action": "purge", "geloescht_am": time.Now().UTC().Format(time.RFC3339)},
	}); err != nil {
		return fmt.Errorf("writing purge audit log: %w", err)
	}
	return nil
}

// PurgeStudent entfernt einen im Papierkorb liegenden Schüler endgültig und
// DSGVO-konform. Nur aus dem Papierkorb; offene Ausleihen/unbezahlte Schäden blockieren.
func (r *pgAuditRepository) PurgeStudent(ctx context.Context, studentID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Nur bereits weichgelöschte Schüler (Papierkorb) dürfen endgültig entfernt werden.
	var imPapierkorb bool
	err = tx.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM schueler WHERE id = $1`, studentID).Scan(&imPapierkorb)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("student %s not found", studentID)
	}
	if err != nil {
		return fmt.Errorf("checking trash state: %w", err)
	}
	if !imPapierkorb {
		return fmt.Errorf("student %s ist nicht im Papierkorb — erst löschen, dann endgültig entfernen", studentID)
	}

	if err = blockiereBeiOffenenVorgaengen(ctx, tx, studentID); err != nil {
		return err
	}
	if err = r.entferneSchuelerPIIUndLoesche(ctx, tx, studentID, bearbeiterID, "DSGVO-Löschung (Purge)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PurgeAbgaenger entfernt einen ehemaligen Schüler (Abgänger) endgültig und
// DSGVO-konform — der Cronjob-Pendant zu PurgeStudent. Anders als PurgeStudent ist der
// Abgänger NICHT im Papierkorb (ist_abgaenger=true, deleted_at IS NULL); die Auswahl
// (Karenzzeit, ist_abgaenger) trifft der Aufrufer. Die Blockade bei offenen Vorgängen
// wird hier dennoch geprüft — als Sicherheitsnetz gegen einen Race zwischen Auswahl
// und Löschung.
func (r *pgAuditRepository) PurgeAbgaenger(ctx context.Context, studentID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	if err = blockiereBeiOffenenVorgaengen(ctx, tx, studentID); err != nil {
		return err
	}
	if err = r.entferneSchuelerPIIUndLoesche(ctx, tx, studentID, bearbeiterID, "DSGVO-Löschung (Abgänger-Cronjob)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
