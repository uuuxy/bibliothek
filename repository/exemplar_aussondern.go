package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Die Fehler beim Aussondern eines Exemplars. Fehlerwerte und kein Vergleich am Text der
// Meldung: Der traf nie, und ein verliehenes Buch endete als Serverfehler statt als Auskunft.
var (
	ErrExemplarNochVerliehen = errors.New("exemplar ist aktuell noch verliehen")
	ErrExemplarNichtGefunden = errors.New("exemplar nicht gefunden oder bereits ausgebucht")
)

// SondereExemplarAus bucht ein Exemplar aus und schreibt den Eintrag dazu. Die Zeile bleibt in
// der Tabelle und gilt als ausgesondert, weil Ausleihen und Schadensfälle an ihr hängen. Ein
// verliehenes Exemplar lehnt es ab.
func (a *pgProtokollAbfragen) SondereExemplarAus(ctx context.Context, exemplarID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Was der Eintrag nennt, wird vor dem Ändern gelesen. Ein unbekanntes Exemplar ist ein
	// Fehler: Sonst entstünde ein Eintrag über ein Aussondern, das es nicht gab.
	var barcode, zustandNotiz, titel string
	var titelID string
	err = tx.QueryRow(ctx,
		`SELECT e.barcode_id, coalesce(e.zustand_notiz,''), e.titel_id, t.titel
		 FROM buecher_exemplare e
		 JOIN buecher_titel t ON e.titel_id = t.id
		 WHERE e.id = $1`,
		exemplarID,
	).Scan(&barcode, &zustandNotiz, &titelID, &titel)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExemplarNichtGefunden
	}
	if err != nil {
		return fmt.Errorf("failed to snapshot copy for audit: %w", err)
	}

	var offeneAusleihen int
	err = tx.QueryRow(ctx, "SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL", exemplarID).Scan(&offeneAusleihen)
	if err != nil {
		return fmt.Errorf("failed to check active loans for copy: %w", err)
	}
	if offeneAusleihen > 0 {
		return ErrExemplarNochVerliehen
	}

	// Die Bedingung ist_ausgesondert = false macht den zweiten Klick zu „nicht gefunden" statt
	// zu einem zweiten Erfolg und verhindert, dass er die Notiz eines längst ausgebuchten
	// Exemplars überschreibt.
	tag, err := tx.Exec(ctx, "UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'AUSSORTIERT', zustand_notiz = 'Systematisch gelöscht', bestellstatus = NULL, letzte_bewegung_am = "+sqlStempelJetzt+" WHERE id = $1 AND ist_ausgesondert = false", exemplarID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}

	kontext := "Buch ausgebuchen (Soft-Delete)"
	if err = a.schreibeEintrag(ctx, tx, protokollEintrag{
		Tabelle: "buecher_exemplare", Aktion: "UPDATE", DatensatzID: exemplarID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{"barcode_id": barcode, "zustand_notiz": zustandNotiz, "titel_id": titelID, "titel": titel, "action": "soft_delete"},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
