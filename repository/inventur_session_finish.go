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

// ExemplarImScope prüft, ob ein Exemplar zu den Scope-Dimensionen einer Session gehört
// (Signatur/Fach/Klasse). Für globale Sessions ist immer true. Dient nur der
// nicht-blockierenden Scan-Warnung ("gehört nicht zum Scope") — physische Bedingungen
// (verliehen etc.) spielen hier keine Rolle, das Buch liegt ja gescannt in der Hand.
func (r *InventoryRepository) ExemplarImScope(ctx context.Context, exemplarID string, scope InventurScope) (bool, error) {
	dimBedingung, dimArgs := scope.DimensionBedingung(2)
	if dimBedingung == "TRUE" {
		return true, nil
	}
	args := append([]any{exemplarID}, dimArgs...)
	var vorhanden bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE e.id = $1 AND `+dimBedingung+`
		)
	`, args...).Scan(&vorhanden)
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
func (r *InventoryRepository) FinishInventurSession(ctx context.Context, sessionID string, scope InventurScope) (int, error) {
	// Scope-Prädikat (physisch + Dimensionen) aus der einen Quelle; die Session-ID hängt
	// als letzter Platzhalter hinten dran.
	bedingung, args := scope.Bedingung(1)
	sessionIdx := len(args) + 1
	args = append(args, sessionID)
	// Die gebuchten Verluste werden zugleich MITGESCHRIEBEN, nicht nur gezählt.
	//
	// Vorher gab dieser Aufruf allein eine Zahl zurück: „47 Verluste". Welche 47, erfuhr
	// niemand — und danach war es auch nicht mehr herleitbar, weil die Exemplare durch die
	// Aussonderung aus der Scope-Bedingung fallen. Damit liess sich weder nachsehen, ob ein
	// Buch wirklich fehlt oder nur im falschen Regal stand, noch eine Liste zum Nachsuchen
	// ausdrucken.
	//
	// Titel, Autor und Signatur werden als ABSCHRIFT übernommen (siehe Migration 059):
	// Der Bericht muss auch dann noch lesbar sein, wenn Exemplar oder Titel später
	// endgültig gelöscht werden — und genau dann braucht man ihn, wenn jemand nachfragt.
	// Gezaehlt wird die AUSSONDERUNG, nicht die Mitschrift.
	//
	// Der erste Entwurf las tag.RowsAffected() des Gesamt-Statements — das ist bei einem
	// abschliessenden INSERT dessen Zeilenzahl, nicht die des UPDATE. Solange beide gleich
	// sind, faellt das nicht auf; ueberspringt ON CONFLICT je eine Zeile, meldete die
	// Inventur weniger Verluste, als sie tatsaechlich gebucht hat. Deshalb liefert das
	// abschliessende SELECT die Zahl aus dem UPDATE-CTE.
	query := fmt.Sprintf(`
		WITH verloren AS (
			UPDATE buecher_exemplare e
			SET ist_ausleihbar = false,
			    ist_ausgesondert = true,
			    aussonderung_grund = 'VERLUST',
			    zustand_notiz = 'Verlust bei Inventur',
			    aktualisiert_am = CURRENT_TIMESTAMP
			FROM buecher_titel t
			WHERE e.titel_id = t.id
			  AND %s
			  AND NOT EXISTS (
			      SELECT 1 FROM inventur_erfassungen ie
			      WHERE ie.session_id = $%d AND ie.exemplar_id = e.id
			  )
			RETURNING e.id, e.barcode_id, t.titel, coalesce(t.autor, '') AS autor, t.signature_id
		), mitschrift AS (
			INSERT INTO inventur_verluste (session_id, exemplar_id, barcode_id, titel, autor, signatur)
			SELECT $%d, v.id, v.barcode_id, v.titel, v.autor, coalesce(s.name, '')
			FROM verloren v
			LEFT JOIN signatures s ON s.id = v.signature_id
			ON CONFLICT DO NOTHING
			RETURNING 1
		)
		SELECT (SELECT count(*) FROM verloren)
	`, bedingung, sessionIdx, sessionIdx)
	var verloren int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&verloren); err != nil {
		return 0, fmt.Errorf("verluste markieren fehlgeschlagen: %w", err)
	}

	if _, err := r.db.Exec(ctx, `
		UPDATE inventur_sessions
		SET abgeschlossen_am = now(), verloren_gemeldet = $2
		WHERE id = $1 AND abgeschlossen_am IS NULL
	`, sessionID, verloren); err != nil {
		return 0, fmt.Errorf("session abschliessen fehlgeschlagen: %w", err)
	}
	return verloren, nil
}

// InventurVerlust ist eine Zeile des Fehlbestandsberichts.
type InventurVerlust struct {
	BarcodeID string `json:"barcode_id"`
	Titel     string `json:"titel"`
	Autor     string `json:"autor"`
	Signatur  string `json:"signatur"`
	GebuchtAm string `json:"gebucht_am"`
}

// LadeInventurVerluste liefert den Fehlbestand einer abgeschlossenen Session.
//
// Sortiert nach Signatur und Titel — das ist die Reihenfolge, in der man mit der Liste
// durchs Regal geht. Nach Barcode oder Buchungszeit sortiert wäre sie zum Nachsuchen
// unbrauchbar, weil man dann kreuz und quer laufen müsste.
func (r *InventoryRepository) LadeInventurVerluste(ctx context.Context, sessionID string) ([]InventurVerlust, error) {
	rows, err := r.db.Query(ctx, `
		SELECT barcode_id, titel, autor, signatur, to_char(gebucht_am, 'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM inventur_verluste
		WHERE session_id = $1
		ORDER BY signatur, titel, barcode_id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("fehlbestand laden fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	// Nie nil: Eine leere Liste muss beim Client als [] ankommen.
	liste := make([]InventurVerlust, 0)
	for rows.Next() {
		var v InventurVerlust
		if err := rows.Scan(&v.BarcodeID, &v.Titel, &v.Autor, &v.Signatur, &v.GebuchtAm); err != nil {
			return nil, fmt.Errorf("fehlbestand lesen fehlgeschlagen: %w", err)
		}
		liste = append(liste, v)
	}
	return liste, rows.Err()
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
