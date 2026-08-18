package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Zwei Handlungen für den Fehlbestandsbericht (Migration 061). Vorher war er reine
// Anzeige — Peter (05.08.2026): „was mache ich [dann]? ich kann nicht weiter damit
// machen!" Beim Regal-Absuchen gibt es genau zwei Ausgänge je Zeile: das Buch taucht
// doch wieder auf, oder es bleibt endgültig weg. Beide brauchen einen Knopf.
//
// Transaktionsgrenze liegt bewusst NICHT hier: Wie FinishInventurSession erwarten
// diese Methoden ein bereits eröffnetes db (Pool oder Tx) — der Aufrufer (Handler)
// eröffnet die Transaktion, damit Snapshot, Update und Audit-Log atomar bleiben.

// MarkiereVerlustAlsGefunden bringt ein als Verlust gebuchtes Exemplar zurück in
// Umlauf und trägt den Fund im Fehlbestandsbericht ein. false ohne Fehler heisst: kein
// offener Verlust mit dieser ID (falsche ID oder schon endgültig gelöscht) — dann gibt
// es nichts zu tun.
func (r *InventoryRepository) MarkiereVerlustAlsGefunden(ctx context.Context, exemplarID, bearbeiterID string) (bool, error) {
	var barcode, titel string
	err := r.db.QueryRow(ctx,
		`SELECT e.barcode_id, t.titel FROM buecher_exemplare e
		 JOIN buecher_titel t ON t.id = e.titel_id
		 WHERE e.id = $1 AND e.ist_ausgesondert = true AND e.aussonderung_grund = 'VERLUST'`,
		exemplarID,
	).Scan(&barcode, &titel)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("exemplar für Fund-Meldung lesen fehlgeschlagen: %w", err)
	}

	if _, err := r.db.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = false, ist_ausleihbar = true, aussonderung_grund = NULL,
		    zustand_notiz = '', bestellstatus = NULL, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $1
	`, exemplarID); err != nil {
		return false, fmt.Errorf("exemplar wiederherstellen fehlgeschlagen: %w", err)
	}

	// Alle noch offenen Verlust-Zeilen dieses Exemplars schliessen, nicht nur die einer
	// bestimmten Session — ein Exemplar kann rein rechnerisch mehrfach als Verlust
	// gebucht worden sein, wenn ein Fund vor dem Klick hier nie eingetragen wurde.
	if _, err := r.db.Exec(ctx, `
		UPDATE inventur_verluste SET gefunden_am = CURRENT_TIMESTAMP
		WHERE exemplar_id = $1 AND gefunden_am IS NULL
	`, exemplarID); err != nil {
		return false, fmt.Errorf("verlustbericht aktualisieren fehlgeschlagen: %w", err)
	}

	if err := schreibeAuditLog(ctx, r.db, auditEntry{
		Tabelle: "buecher_exemplare", Aktion: "UPDATE", DatensatzID: exemplarID,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Kontext: strPtr("Als Verlust gebuchtes Exemplar beim Nachsuchen gefunden"),
		Details: map[string]any{"barcode_id": barcode, "titel": titel, "action": "verlust_gefunden"},
	}); err != nil {
		return false, err
	}
	return true, nil
}

// EndgueltigLoescheVerlustExemplare entfernt bereits als Verlust gebuchte Exemplare
// unwiderruflich aus dem Katalog. Der Fehlbestandsbericht bleibt lesbar (Migration
// 059: ON DELETE SET NULL, die Textspalten sind eine Abschrift) — nur das Löschen
// beendet den offenen Zustand endgültig.
//
// Bewusst eingeschränkt auf ist_ausgesondert=true (VERLUST): Diese Funktion ist NICHT
// der allgemeine Lösch-Weg für Exemplare (dafür DeleteCopyHandler, der soft löscht) —
// sie darf nur aufräumen, was der Fehlbestandsbericht selbst schon als verloren
// gebucht hat. Liefert die Zahl der tatsächlich gelöschten Exemplare (IDs, die schon
// vorher weg waren oder nicht VERLUST sind, werden still übersprungen).
func (r *InventoryRepository) EndgueltigLoescheVerlustExemplare(ctx context.Context, exemplarIDs []string, bearbeiterID string) (int, error) {
	if len(exemplarIDs) == 0 {
		return 0, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.barcode_id, t.titel
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.id = ANY($1) AND e.ist_ausgesondert = true AND e.aussonderung_grund = 'VERLUST'
	`, exemplarIDs)
	if err != nil {
		return 0, fmt.Errorf("zu löschende Verlust-Exemplare lesen fehlgeschlagen: %w", err)
	}
	type snapshot struct{ id, barcode, titel string }
	var treffer []snapshot
	for rows.Next() {
		var s snapshot
		if err := rows.Scan(&s.id, &s.barcode, &s.titel); err != nil {
			rows.Close()
			return 0, fmt.Errorf("verlust-exemplar lesen fehlgeschlagen: %w", err)
		}
		treffer = append(treffer, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(treffer) == 0 {
		return 0, nil
	}

	ids := make([]string, len(treffer))
	for i, s := range treffer {
		ids[i] = s.id
	}
	if _, err := r.db.Exec(ctx,
		`DELETE FROM buecher_exemplare WHERE id = ANY($1)`, ids,
	); err != nil {
		return 0, fmt.Errorf("verlust-exemplare endgültig löschen fehlgeschlagen: %w", err)
	}

	for _, s := range treffer {
		if err := schreibeAuditLog(ctx, r.db, auditEntry{
			Tabelle: "buecher_exemplare", Aktion: "DELETE", DatensatzID: s.id,
			BearbeiterID: &bearbeiterID, Akteur: "USER",
			Kontext: strPtr("Als Verlust gebuchtes Exemplar endgültig gelöscht"),
			Details: map[string]any{"barcode_id": s.barcode, "titel": s.titel, "action": "verlust_endgueltig_geloescht"},
		}); err != nil {
			return 0, err
		}
	}
	return len(treffer), nil
}

func strPtr(s string) *string { return &s }

// schreibeAuditLog schreibt EINEN audit_log-Eintrag über die generische DBQueryer-
// Schnittstelle. Eigene, kleine Kopie statt pgAuditRepository.insertAuditLog: Jene
// verlangt konkret pgx.Tx, hier liegt aber nur r.db (DBQueryer) vor — dieselbe
// Verengung, die schon FinishInventurSession umgeht, indem es Pool und Tx über
// dasselbe Interface anspricht. Direktes Schreiben in audit_log ist an mehreren
// Stellen im Projekt so üblich (z. B. api/student_update.go), nicht nur über die
// Kapselung in audit.go.
func schreibeAuditLog(ctx context.Context, db DBQueryer, e auditEntry) error {
	var detailsJSON []byte
	if e.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(e.Details)
		if err != nil {
			return fmt.Errorf("audit details serialization: %w", err)
		}
	}
	_, err := db.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, kontext, details)
		VALUES ($1, $2, $3::uuid, $4, $5, $6, $7)
	`, e.Tabelle, e.Aktion, e.DatensatzID, e.BearbeiterID, e.Akteur, e.Kontext,
		func() any {
			if detailsJSON == nil {
				return nil
			}
			return string(detailsJSON)
		}())
	return err
}
