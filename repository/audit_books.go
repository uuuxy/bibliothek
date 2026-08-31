package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrTitelHatAktiveAusleihen meldet, dass ein Titel nicht gelöscht werden kann, weil
// noch Exemplare verliehen sind. Das ist ein Bedienfehler (HTTP 400), keine Störung.
//
// Als Sentinel und nicht als Textvergleich: Der Handler prüfte die Meldung früher per
// err.Error()[:22] gegen "Löschen fehlgeschlagen:" — und traf nie. Der Text hier begann
// klein, das Literal groß, und 22 Bytes können ein 24-Byte-Literal ohnehin nicht
// treffen (die Umlaute zählen doppelt). Jeder blockierte Löschversuch wurde damit zu
// einem HTTP 500, und die Liste der noch verliehenen Barcodes — genau die Auskunft, die
// weiterhilft — verschwand hinter „Serverfehler".
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, nutzer-sichtbare Meldung
var ErrTitelHatAktiveAusleihen = errors.New("Löschen fehlgeschlagen: folgende Exemplare sind noch verliehen")

// ErrExemplarNochVerliehen / ErrExemplarNichtGefunden: dieselbe Sentinel-Bauart für das
// EINZELNE Exemplar. Der Handler entschied hier bis zum 31.08.2026 per Textvergleich —
// „Exemplar ist aktuell noch verliehen!" gegen „exemplar ist aktuell noch verliehen"
// (Großschreibung + Ausrufezeichen): Der Vergleich traf nie, ein verliehenes Buch endete
// als 500 mit Sanitizer-Text statt als 400 mit Auskunft.
var (
	ErrExemplarNochVerliehen = errors.New("exemplar ist aktuell noch verliehen")
	ErrExemplarNichtGefunden = errors.New("exemplar nicht gefunden oder bereits ausgebucht")
	// ErrTitelNichtGefunden: unbekannte Titel-ID beim Löschen (Phantom-Erfolg-Sweep 31.08.2026).
	ErrTitelNichtGefunden = errors.New("titel nicht gefunden")
)

// DeleteTitle entfernt einen Buchtitel vollständig aus dem Katalog und erstellt einen revisionssicheren Audit-Eintrag.
// Vor dem Löschen wird geprüft, ob noch Exemplare dieses Titels verliehen sind (was das Löschen blockiert).
// Historische Ausleihen und abgeschlossene Schadensfälle werden bereinigt, um Fremdschlüssel-Fehler zu vermeiden.
func (r *pgAuditRepository) DeleteTitle(ctx context.Context, titleID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Metadaten vor dem Löschen für das Audit-Log sichern
	var titel, autor, isbn string
	err = tx.QueryRow(ctx,
		`SELECT coalesce(titel,''), coalesce(autor,''), coalesce(isbn,'') FROM buecher_titel WHERE id = $1`,
		titleID,
	).Scan(&titel, &autor, &isbn)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot title for audit: %w", err)
	}

	// Sicherheitsschranke: Prüfen, ob irgendein Exemplar dieses Titels aktuell ausgeliehen ist
	var activeLoans []string
	rows, err := tx.Query(ctx, `
		SELECT e.barcode_id 
		FROM ausleihen a 
		JOIN buecher_exemplare e ON a.exemplar_id = e.id 
		WHERE e.titel_id = $1 AND a.rueckgabe_am IS NULL
	`, titleID)
	if err != nil {
		return fmt.Errorf("failed to check active loans for title: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var barcode string
		if err := rows.Scan(&barcode); err == nil {
			activeLoans = append(activeLoans, barcode)
		}
	}
	// Bricht die Iteration durch einen Verbindungsfehler vorzeitig ab, dürfen wir NICHT
	// von "keine aktiven Ausleihen" ausgehen und den Titel löschen.
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read active loans for title: %w", err)
	}
	if len(activeLoans) > 0 {
		return fmt.Errorf("%w: %v", ErrTitelHatAktiveAusleihen, activeLoans)
	}

	// Verknüpfte Einträge (Schadensfälle, alte Rückgaben) löschen, um ON DELETE RESTRICT Fehler zu vermeiden
	if _, err = tx.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)", titleID); err != nil {
		return fmt.Errorf("failed to delete damage records for title: %w", err)
	}
	if _, err = tx.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1) AND rueckgabe_am IS NOT NULL", titleID); err != nil {
		return fmt.Errorf("failed to delete past loans for title: %w", err)
	}

	// Alle zugehörigen Exemplare löschen
	if _, err = tx.Exec(ctx, "DELETE FROM buecher_exemplare WHERE titel_id = $1", titleID); err != nil {
		return fmt.Errorf("failed to delete associated copies: %w", err)
	}

	// Eigentlichen Titel-Datensatz löschen. 0 Zeilen = unbekannte ID: Vorher lief eine
	// unbekannte Titel-ID glatt durch (Snapshot toleriert ErrNoRows, 0 aktive Ausleihen)
	// und hinterließ einen DELETE-Audit-Eintrag über einen Titel, den es nie gab
	// (Phantom-Erfolg-Sweep 31.08.2026; die Exemplar-Schwester unten prüft längst).
	tag, err := tx.Exec(ctx, "DELETE FROM buecher_titel WHERE id = $1", titleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTitelNichtGefunden
	}

	// Löschung im Audit-Log vermerken
	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "buecher_titel", Aktion: "DELETE", DatensatzID: titleID,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Details: map[string]any{"titel": titel, "autor": autor, "isbn": isbn},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteCopy bucht ein physisches Exemplar aus dem System aus (Soft-Delete) und protokolliert dies im Audit-Log.
// Da historische Ausleihdaten und Schadensfälle für statistische Zwecke erhalten bleiben müssen, wird das Exemplar
// nicht physisch aus der Tabelle gelöscht, sondern als ausgesondert markiert.
func (r *pgAuditRepository) DeleteCopy(ctx context.Context, copyID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
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
		copyID,
	).Scan(&barcode, &zustandNotiz, &titelID, &titel)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExemplarNichtGefunden
	}
	if err != nil {
		return fmt.Errorf("failed to snapshot copy for audit: %w", err)
	}

	// Sicherheitsschranke: Ist das Exemplar aktuell noch verliehen?
	var activeLoanCount int
	err = tx.QueryRow(ctx, "SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL", copyID).Scan(&activeLoanCount)
	if err != nil {
		return fmt.Errorf("failed to check active loans for copy: %w", err)
	}
	if activeLoanCount > 0 {
		return ErrExemplarNochVerliehen
	}

	// Soft-Delete durchführen: Exemplar sperren und Zustand auf "Systematisch gelöscht"
	// setzen. Der Guard ist_ausgesondert = false macht den Doppelklick zum 404 statt zum
	// zweiten „Erfolg" — und verhindert, dass ein Wiederholungsklick die Zustandsnotiz
	// eines längst ausgebuchten Exemplars erneut überschreibt.
	tag, err := tx.Exec(ctx, "UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'AUSSORTIERT', zustand_notiz = 'Systematisch gelöscht' WHERE id = $1 AND ist_ausgesondert = false", copyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}

	kontext := "Buch ausgebuchen (Soft-Delete)"
	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "buecher_exemplare", Aktion: "UPDATE", DatensatzID: copyID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{"barcode_id": barcode, "zustand_notiz": zustandNotiz, "titel_id": titelID, "titel": titel, "action": "soft_delete"},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// LogAusleihe schreibt einen neuen Ausleiheintrag (CHECKOUT) in das Audit-Log.
func (r *pgAuditRepository) LogAusleihe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return r.logLoanEvent(ctx, "ausleihen", "CHECKOUT", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// LogRueckgabe schreibt einen neuen Rückgabeeintrag (RETURN) in das Audit-Log.
func (r *pgAuditRepository) LogRueckgabe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return r.logLoanEvent(ctx, "ausleihen", "RETURN", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// logLoanEvent ist die interne Hilfsfunktion zur Erstellung von Transaktionsprotokollen für Ausleihen und Rückgaben.
func (r *pgAuditRepository) logLoanEvent(ctx context.Context, tabelle, aktion, exemplarID, schuelerID, benutzerID, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	var bearbeiterPtr *string
	if bearbeiterID != "" {
		bearbeiterPtr = &bearbeiterID
	}

	details := map[string]any{
		"exemplar_id": exemplarID,
		"zeitpunkt":   time.Now().UTC().Format(time.RFC3339),
	}
	if schuelerID != "" {
		details["schueler_id"] = schuelerID
	}
	if benutzerID != "" {
		details["benutzer_id"] = benutzerID
	}

	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: tabelle, Aktion: aktion, DatensatzID: exemplarID,
		BearbeiterID: bearbeiterPtr, Akteur: "USER",
		Details: details,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
