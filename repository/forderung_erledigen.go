package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Die Fehler beim Erledigen einer Forderung. Die Tür macht daraus „nicht gefunden" oder einen
// Konflikt; ein Fehler ohne Namen käme als „interner Fehler" beim Formular an.
var (
	ErrSchadensfallNichtGefunden = errors.New("schadensfall nicht gefunden")
	ErrSchadensfallErledigt      = errors.New("schadensfall schon bezahlt oder storniert")
)

// ladeBetragFuerErledigung liest den Betrag einer offenen Forderung und sperrt die Zeile bis zum
// Commit. Im Protokoll steht so der Betrag aus der Datenbank und nicht der aus der Anfrage.
// Eine unbekannte und eine schon erledigte Forderung meldet die Funktion je mit eigenem Fehler.
func ladeBetragFuerErledigung(ctx context.Context, tx pgx.Tx, schadensfallID string) (float64, error) {
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

// StorniereForderung setzt eine offene Forderung mit Begründung auf storniert und schreibt den
// Eintrag dazu; die Zeile bleibt als Beleg stehen. Den Betrag liest es aus der Datenbank, der
// Rufer übergibt ihn nicht. ist_bezahlt wird mitgesetzt: Wer offene Forderungen zählt, fragt nur
// ist_bezahlt = false, und bezahlt und storniert unterscheidet storniert_am.
func (a *SchadensAbfragen) StorniereForderung(ctx context.Context, schadensfallID string, bearbeiterID string, grund string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.Verwerfe(ctx, tx)

	betrag, err := ladeBetragFuerErledigung(ctx, tx, schadensfallID)
	if err != nil {
		return err
	}

	var leserID *string
	if err := tx.QueryRow(ctx, `
		UPDATE schadensfaelle
		SET ist_bezahlt = true,
		    storniert_am = NOW(),
		    storniert_von = $1,
		    stornierungsgrund = $2,
		    aktualisiert_am = NOW()
		WHERE id = $3
		RETURNING schueler_id::text
	`, bearbeiterID, grund, schadensfallID).Scan(&leserID); err != nil {
		return fmt.Errorf("stornierung schadensfaelle: %w", err)
	}

	// Der getippte Grund kann die Person nennen. Die Kennung des Lesers steht daneben, damit
	// die Tilgung den Eintrag findet, auch wenn die Forderung schon gelöscht ist.
	details := map[string]any{
		"betrag":        betrag,
		"grund":         grund,
		"storniert_am":  time.Now().UTC().Format(time.RFC3339),
		"bearbeiter_id": bearbeiterID,
	}
	if leserID != nil {
		details["schueler_id"] = *leserID
	}
	kontext := "Gebühr storniert"
	if err = schreibeProtokollEintrag(ctx, tx, protokollEintrag{
		Tabelle: "schadensfaelle", Aktion: "STORNIERUNG", DatensatzID: schadensfallID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: details,
	}); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// VerbucheZahlung verbucht die Zahlung einer offenen Forderung, im Alltag bar an der Theke, und
// schreibt den Eintrag dazu. Die Spalten der Stornierung bleiben leer; so bleiben bezahlt und
// storniert unterscheidbar.
func (a *SchadensAbfragen) VerbucheZahlung(ctx context.Context, schadensfallID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.Verwerfe(ctx, tx)

	betrag, err := ladeBetragFuerErledigung(ctx, tx, schadensfallID)
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
	if err = schreibeProtokollEintrag(ctx, tx, protokollEintrag{
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
