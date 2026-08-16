package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Sentinel-Fehler der Gebühren-Erledigung — die Handler mappen sie auf 404/409.
// Ein generischer Fehler würde vom 500-Sanitizer zu "interner Datenbankfehler"
// verschluckt und erreichte das Formular nie.
var (
	ErrSchadensfallNichtGefunden = errors.New("Schadensfall nicht gefunden")
	ErrSchadensfallErledigt      = errors.New("Schadensfall wurde bereits bezahlt oder storniert")
)

// betragFuerErledigung liest den Betrag eines noch offenen Schadensfalls und sperrt
// die Zeile (FOR UPDATE) bis zum Commit. Der Betrag steht damit IMMER aus der
// Datenbank im Audit-Protokoll — nie aus dem Request, wo ihn ein Client fälschen
// könnte. Gleichzeitig liefert die Abfrage die 404/409-Unterscheidung.
func betragFuerErledigung(ctx context.Context, tx pgx.Tx, schadensfallID string) (float64, error) {
	var betrag float64
	var istBezahlt bool
	err := tx.QueryRow(ctx,
		`SELECT betrag, ist_bezahlt FROM schadensfaelle WHERE id = $1 FOR UPDATE`,
		schadensfallID).Scan(&betrag, &istBezahlt)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrSchadensfallNichtGefunden
	}
	if err != nil {
		return 0, fmt.Errorf("schadensfall laden: %w", err)
	}
	if istBezahlt {
		return 0, ErrSchadensfallErledigt
	}
	return betrag, nil
}

// StornierungGebuehr markiert eine ausstehende Gebühr (Schadensfall) als storniert und schreibt einen
// revisionssicheren Audit-Eintrag. Dies ersetzt das physische Löschen von Schadensfällen, um die Historie zu wahren.
// ist_bezahlt wird bewusst mitgesetzt: Alle Offen-Filter hängen an ist_bezahlt = false;
// die Lesepfade unterscheiden bezahlt/storniert über storniert_am.
func (r *pgAuditRepository) StornierungGebuehr(ctx context.Context, schadensfallID string, bearbeiterID string, grund string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	betrag, err := betragFuerErledigung(ctx, tx, schadensfallID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE schadensfaelle
		SET ist_bezahlt = true,
		    storniert_am = NOW(),
		    storniert_von = $1,
		    stornierungsgrund = $2,
		    aktualisiert_am = NOW()
		WHERE id = $3
	`, bearbeiterID, grund, schadensfallID); err != nil {
		return fmt.Errorf("stornierung schadensfaelle: %w", err)
	}

	kontext := "Gebühr storniert"
	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "schadensfaelle", Aktion: "STORNIERUNG", DatensatzID: schadensfallID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{
			"betrag":        betrag,
			"grund":         grund,
			"storniert_am":  time.Now().UTC().Format(time.RFC3339),
			"bearbeiter_id": bearbeiterID,
		},
	}); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// BezahltGebuehr verbucht die Zahlung einer ausstehenden Gebühr (der Alltagsfall:
// Barzahlung am Tresen) und schreibt den Audit-Eintrag. Die Storno-Spalten bleiben
// unberührt — bezahlt und storniert bleiben so unterscheidbar.
func (r *pgAuditRepository) BezahltGebuehr(ctx context.Context, schadensfallID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	betrag, err := betragFuerErledigung(ctx, tx, schadensfallID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE schadensfaelle
		SET ist_bezahlt = true, aktualisiert_am = NOW()
		WHERE id = $1
	`, schadensfallID); err != nil {
		return fmt.Errorf("zahlung schadensfaelle: %w", err)
	}

	kontext := "Gebühr bezahlt"
	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "schadensfaelle", Aktion: "ZAHLUNG", DatensatzID: schadensfallID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{
			"betrag":        betrag,
			"bearbeiter_id": bearbeiterID,
		},
	}); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// LogSystemAktion schreibt einen Audit-Eintrag für eine rein systemgesteuerte Aktion (ohne Bearbeiter-ID).
// Dies wird typischerweise von Cronjobs, DSGVO-Routinebereinigungen oder Datensicherungen verwendet.
func (r *pgAuditRepository) LogSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Systemaktionen verwenden eine Standard-Null-UUID als Datensatz-ID
	const systemSentinelID = "00000000-0000-0000-0000-000000000000"

	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: tabelle, Aktion: aktion, DatensatzID: systemSentinelID,
		Akteur: "SYSTEM", Kontext: &kontext, Details: details,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
