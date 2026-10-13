package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrExemplarNochVerliehen / ErrExemplarNichtGefunden: dieselbe Sentinel-Bauart für das
// EINZELNE Exemplar. Der Handler entschied hier bis zum 31.08.2026 per Textvergleich —
// „Exemplar ist aktuell noch verliehen!" gegen „exemplar ist aktuell noch verliehen"
// (Großschreibung + Ausrufezeichen): Der Vergleich traf nie, ein verliehenes Buch endete
// als 500 mit Sanitizer-Text statt als 400 mit Auskunft.
var (
	ErrExemplarNochVerliehen = errors.New("exemplar ist aktuell noch verliehen")
	ErrExemplarNichtGefunden = errors.New("exemplar nicht gefunden oder bereits ausgebucht")
)

func (a *pgProtokollAbfragen) SondereExemplarAus(ctx context.Context, exemplarID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Exemplardaten vor dem Aussondern für das Audit-Log sichern.
	// ErrNoRows ist hier KEIN tolerierbarer Fall (bis 31.08.2026 war er es): Eine
	// unbekannte ID endete sonst als Phantom-Erfolg — 200 samt Audit-Eintrag über eine
	// Löschung, die nie stattfand.
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

	// Sicherheitsschranke: Ist das Exemplar aktuell noch verliehen?
	var offeneAusleihen int
	err = tx.QueryRow(ctx, "SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL", exemplarID).Scan(&offeneAusleihen)
	if err != nil {
		return fmt.Errorf("failed to check active loans for copy: %w", err)
	}
	if offeneAusleihen > 0 {
		return ErrExemplarNochVerliehen
	}

	// Soft-Delete durchführen: Exemplar sperren und Zustand auf "Systematisch gelöscht"
	// setzen. Der Guard ist_ausgesondert = false macht den Doppelklick zum 404 statt zum
	// zweiten „Erfolg" — und verhindert, dass ein Wiederholungsklick die Zustandsnotiz
	// eines längst ausgebuchten Exemplars erneut überschreibt.
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
