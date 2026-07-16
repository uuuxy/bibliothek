package repository

import (
	"context"
	"fmt"
)

// RecordInventurScan verbucht ein Exemplar als in dieser Session erfasst. Ein erneuter
// Scan desselben Exemplars in derselben Session ist ein No-op (Primärschlüssel).
func (r *InventoryRepository) RecordInventurScan(ctx context.Context, sessionID, exemplarID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO inventur_erfassungen (session_id, exemplar_id)
		VALUES ($1, $2)
		ON CONFLICT (session_id, exemplar_id) DO NOTHING
	`, sessionID, exemplarID)
	if err != nil {
		return fmt.Errorf("scan verbuchen fehlgeschlagen: %w", err)
	}
	return nil
}

// ExemplarImScope prüft, ob ein Exemplar zum Scope einer Signatur-Session gehört.
// Für globale Sessions (signatureID == nil) ist immer true. Dient nur der
// nicht-blockierenden Scan-Warnung ("gehört nicht zum Scope").
func (r *InventoryRepository) ExemplarImScope(ctx context.Context, exemplarID string, signatureID *int) (bool, error) {
	if signatureID == nil {
		return true, nil
	}
	var vorhanden bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE e.id = $1 AND t.signature_id = $2
		)
	`, exemplarID, *signatureID).Scan(&vorhanden)
	if err != nil {
		return false, fmt.Errorf("scope-prüfung fehlgeschlagen: %w", err)
	}
	return vorhanden, nil
}

// FinishInventurSession schließt eine Session ab: Alle physisch erwartbaren Exemplare
// im Scope, die in DIESER Session nicht erfasst wurden, werden als Verlust markiert.
// Weil "im Scope" verliehene Bücher ausschliesst (inventurScopeBedingung), wird ein
// beim Schüler befindliches Buch nie fälschlich als verloren gebucht.
//
// Entscheidend gegenüber dem alten Modell: Nur die NICHT in dieser Session erfassten
// Exemplare gelten als vermisst — der Fortschritt einer parallelen Session bleibt
// unberührt, weil er session-gebunden in inventur_erfassungen liegt.
func (r *InventoryRepository) FinishInventurSession(ctx context.Context, sessionID string, signatureID *int) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE buecher_exemplare e
		SET ist_ausleihbar = false,
		    ist_ausgesondert = true,
		    aussonderung_grund = 'VERLUST',
		    zustand_notiz = 'Verlust bei Inventur',
		    aktualisiert_am = CURRENT_TIMESTAMP
		FROM buecher_titel t
		WHERE e.titel_id = t.id
		  AND e.ist_ausgesondert = false
		  AND e.ist_ausleihbar = true
		  AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
		  AND ($1::int IS NULL OR t.signature_id = $1)
		  AND NOT EXISTS (
		      SELECT 1 FROM inventur_erfassungen ie
		      WHERE ie.session_id = $2 AND ie.exemplar_id = e.id
		  )
	`, signatureID, sessionID)
	if err != nil {
		return 0, fmt.Errorf("verluste markieren fehlgeschlagen: %w", err)
	}
	verloren := int(tag.RowsAffected())

	if _, err := r.db.Exec(ctx, `
		UPDATE inventur_sessions
		SET abgeschlossen_am = now(), verloren_gemeldet = $2
		WHERE id = $1 AND abgeschlossen_am IS NULL
	`, sessionID, verloren); err != nil {
		return 0, fmt.Errorf("session abschliessen fehlgeschlagen: %w", err)
	}
	return verloren, nil
}

// AbortInventurSession verwirft eine Session ohne Verlustbuchung — für abgebrochene
// oder hängengebliebene Inventuren. Die Erfassungen bleiben (CASCADE räumt sie erst
// beim echten Löschen); der Scope wird dadurch wieder frei für einen Neustart.
func (r *InventoryRepository) AbortInventurSession(ctx context.Context, sessionID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE inventur_sessions
		SET abgeschlossen_am = now(), verloren_gemeldet = 0
		WHERE id = $1 AND abgeschlossen_am IS NULL
	`, sessionID)
	if err != nil {
		return fmt.Errorf("session abbrechen fehlgeschlagen: %w", err)
	}
	return nil
}
