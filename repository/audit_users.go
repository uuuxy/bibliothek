package repository

import (
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
	defer func() { _ = tx.Rollback(ctx) }()

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

	if err = r.insertAuditLog(ctx, tx, "benutzer", "DELETE", userID,
		&bearbeiterID, "USER", nil,
		map[string]any{"vorname": vorname, "nachname": nachname, "email": email, "rolle": rolle},
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteStudent löscht einen Schüler-Datensatz transaktionssicher und datenschutzkonform (DSGVO-konforme Hard-Delete).
// Der Aufruf ist sowohl über die HTTP-API (bearbeiterID = Benutzer-UUID) als auch über den DSGVO-Cronjob (bearbeiterID = "" → SYSTEM) möglich.
// Um personenbezogene Daten (PII) vollständig zu löschen, werden:
//   - Alle historischen Ausleihen anonymisiert (schueler_id = NULL gesetzt).
//   - Alle älteren Audit-Logs dieses Schülers anonymisiert (Details-Felder überschrieben).
//   - Bezahlte Schadensfälle gelöscht (unbezahlte Schadensfälle blockieren die Löschung).
func (r *pgAuditRepository) DeleteStudent(ctx context.Context, studentID string, bearbeiterID string, grund string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	// 1. Ausleihen anonymisieren: Setze die schueler_id bei allen zurückgegebenen Ausleihen auf NULL.
	// Das sorgt dafür, dass Statistiken erhalten bleiben, aber kein Personenbezug mehr existiert.
	if _, err = tx.Exec(ctx,
		`UPDATE ausleihen SET schueler_id = NULL WHERE schueler_id = $1 AND rueckgabe_am IS NOT NULL`,
		studentID,
	); err != nil {
		return fmt.Errorf("anonymizing returned loans: %w", err)
	}

	// 2. Audit-Logs anonymisieren: Überschreibe personenbezogene Felder in alten Log-Einträgen.
	if _, err = tx.Exec(ctx, `
		UPDATE audit_log
		SET details = details || '{"vorname":"Anonymisiert", "nachname":"Anonymisiert", "klasse":"Anonymisiert"}'::jsonb
		WHERE (datensatz_id = $1 OR (details->>'schueler_id') = $2) AND details ? 'vorname'
	`, studentID, studentID); err != nil {
		return fmt.Errorf("anonymizing past audit_logs: %w", err)
	}

	// 3. Schadensfälle bereinigen: Bezahlte Fälle werden gelöscht. Offene Fälle verhindern das Löschen.
	if _, err = tx.Exec(ctx,
		`DELETE FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = true`,
		studentID,
	); err != nil {
		return fmt.Errorf("deleting paid damages: %w", err)
	}

	// 4. Schüler-Datensatz physisch löschen
	tag, err := tx.Exec(ctx, `DELETE FROM schueler WHERE id = $1`, studentID)
	if err != nil {
		return fmt.Errorf("deleting student: %w", err)
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

	kontext := "DSGVO-Löschroutine"

	// Protokolleintrag schreiben – hierbei werden keine echten Namen, sondern "Anonymisiert"-Werte gespeichert.
	if err = r.insertAuditLog(ctx, tx, "schueler", "DELETE", studentID,
		bearbeiterPtr, akteur, &kontext,
		map[string]any{
			"vorname":        "Anonymisiert",
			"nachname":       "Anonymisiert",
			"klasse":         "Anonymisiert",
			"barcode_id":     barcodeID,
			"abgaenger_jahr": abgaengerJahr,
			"grund":          grund,
			"geloescht_am":   time.Now().UTC().Format(time.RFC3339),
		},
	); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}
