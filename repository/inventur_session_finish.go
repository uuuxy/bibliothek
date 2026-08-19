package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// ErrInventurAbgeschlossen meldet, dass der Scan ankam, als die Session bereits
// abgeschlossen war. Der Handler übersetzt das in eine 409 — der Scan ist
// gegenstandslos, aber kein Fehler.
var ErrInventurAbgeschlossen = errors.New("inventur bereits abgeschlossen")

// RecordInventurScan verbucht ein Exemplar als in dieser Session erfasst. Ein erneuter
// Scan desselben Exemplars in derselben Session ist ein No-op (Primärschlüssel).
//
// Nebenläufigkeit (Multi-Scanner-Betrieb): Der Scan koordiniert über die Session-Zeile
// gegen einen gleichzeitigen Abschluss. Er hält sie mit FOR SHARE, solange er die
// Erfassung schreibt; FinishInventurSession nimmt sie mit FOR UPDATE (siehe dort) und
// kann daher nicht committen, bevor dieser Scan fertig ist — der Verlust-Snapshot des
// Abschlusses sieht die Erfassung also garantiert. Ist die Session bereits abgeschlossen,
// findet der FOR-SHARE-Read keine offene Zeile und der Scan wird abgewiesen
// (ErrInventurAbgeschlossen), statt ins Leere zu laufen und ein physisch vorliegendes
// Buch als Verlust zurückzulassen.
func (r *InventoryRepository) RecordInventurScan(ctx context.Context, sessionID, exemplarID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("scan-transaktion konnte nicht gestartet werden: %w", err)
	}
	defer db.SafeRollback(ctx, tx) // No-op nach Commit

	var id string
	if err := tx.QueryRow(ctx,
		`SELECT id FROM inventur_sessions WHERE id = $1 AND abgeschlossen_am IS NULL FOR SHARE`,
		sessionID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInventurAbgeschlossen
		}
		return fmt.Errorf("session für scan sperren fehlgeschlagen: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO inventur_erfassungen (session_id, exemplar_id)
		VALUES ($1, $2)
		ON CONFLICT (session_id, exemplar_id) DO NOTHING
	`, sessionID, exemplarID); err != nil {
		return fmt.Errorf("scan verbuchen fehlgeschlagen: %w", err)
	}
	return tx.Commit(ctx)
}

// SperreInventurSessionFuerAbschluss lädt die offene Session und sperrt ihre Zeile
// EXKLUSIV (FOR UPDATE) — die Gegenseite zum FOR SHARE des Scans. Der Abschluss wartet
// damit auf alle noch schreibenden Scans und zieht seinen Verlust-Snapshot erst danach.
// Muss innerhalb der Abschluss-Transaktion laufen (der Handler übergibt die Tx als db).
func (r *InventoryRepository) SperreInventurSessionFuerAbschluss(ctx context.Context, id string) (*InventurSession, error) {
	var s InventurSession
	err := r.db.QueryRow(ctx, `
		SELECT id, scope_type, scope_signatur, scope_subject, scope_grade, scope_label,
		       gestartet_von::text, gestartet_am::text,
		       (SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1)
		FROM inventur_sessions
		WHERE id = $1 AND abgeschlossen_am IS NULL
		FOR UPDATE
	`, id).Scan(&s.ID, &s.ScopeType, &s.Signatur, &s.Subject, &s.Grade, &s.ScopeLabel,
		&s.GestartetVon, &s.GestartetAm, &s.Erfasst)
	if err != nil {
		return nil, err
	}
	return &s, nil
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
	// Die Signatur kam zunaechst per Join auf die Tabelle `signatures` ueber
	// t.signature_id — eine Spalte, die ausser Migration 021 nie jemand geschrieben hat.
	// Die Abschrift war damit IMMER leer, und ausgerechnet die Sortierung "nach Signatur",
	// mit der man mit dem Zettel durchs Regal laeuft, sortierte nach nichts. Quelle ist
	// jetzt t.signatur, der Text vom Buchruecken (Migration 060).
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
			RETURNING e.id, e.barcode_id, t.titel, coalesce(t.autor, '') AS autor,
			          coalesce(btrim(t.signatur), '') AS signatur
		), mitschrift AS (
			INSERT INTO inventur_verluste (session_id, exemplar_id, barcode_id, titel, autor, signatur)
			SELECT $%d, v.id, v.barcode_id, v.titel, v.autor, v.signatur
			FROM verloren v
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
	// ExemplarID kann fehlen (leerer String): Das FK-Feld geht bei einer endgültigen
	// Löschung des Exemplars auf NULL (Migration 059, ON DELETE SET NULL) — die
	// Textspalten bleiben trotzdem lesbar. Ohne ExemplarID sind "Gefunden" und
	// "endgültig löschen" nicht mehr möglich, das Exemplar ist ja schon weg.
	ExemplarID string `json:"exemplar_id,omitempty"`
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	Signatur   string `json:"signatur"`
	GebuchtAm  string `json:"gebucht_am"`
	// GefundenAm ist gesetzt, sobald das Exemplar beim Nachsuchen wiedergefunden und
	// über den "Gefunden"-Knopf zurück in Umlauf gebracht wurde (Migration 061).
	GefundenAm string `json:"gefunden_am,omitempty"`
}

// LadeInventurVerluste liefert den Fehlbestand einer abgeschlossenen Session.
//
// Sortiert nach Signatur und Titel — das ist die Reihenfolge, in der man mit der Liste
// durchs Regal geht. Nach Barcode oder Buchungszeit sortiert wäre sie zum Nachsuchen
// unbrauchbar, weil man dann kreuz und quer laufen müsste.
func (r *InventoryRepository) LadeInventurVerluste(ctx context.Context, sessionID string) ([]InventurVerlust, error) {
	rows, err := r.db.Query(ctx, `
		SELECT coalesce(exemplar_id::text, ''), barcode_id, titel, autor, signatur,
		       to_char(gebucht_am, 'YYYY-MM-DD"T"HH24:MI:SSOF'),
		       coalesce(to_char(gefunden_am, 'YYYY-MM-DD"T"HH24:MI:SSOF'), '')
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
		if err := rows.Scan(&v.ExemplarID, &v.BarcodeID, &v.Titel, &v.Autor, &v.Signatur, &v.GebuchtAm, &v.GefundenAm); err != nil {
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
