package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// InventurExemplar contains the book data retrieved during a barcode scan.
type InventurExemplar struct {
	ExemplarID      string
	Titel           string
	CoverURL        string
	IstAusgesondert bool
	IstVerliehen    bool
}

// LadeExemplar retrieves the details of a copy by its barcode
// to determine if it is eligible for inventory check-in.
func (a *InventurAbfragen) LadeExemplar(ctx context.Context, barcodeID string) (*InventurExemplar, error) {
	var exemplar InventurExemplar
	query := `
		SELECT e.id, t.titel, coalesce(t.cover_url, ''), e.ist_ausgesondert, EXISTS (
			SELECT 1 FROM ausleihen a
			WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
		) AS is_lent
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.barcode_id = $1
		LIMIT 1
	`
	err := a.db.QueryRow(ctx, query, barcodeID).Scan(&exemplar.ExemplarID, &exemplar.Titel, &exemplar.CoverURL, &exemplar.IstAusgesondert, &exemplar.IstVerliehen)
	if err != nil {
		return nil, fmt.Errorf("exemplar für inventur-scan nicht ladbar: %w", err)
	}
	return &exemplar, nil
}

// ErrInventurAbgeschlossen meldet, dass der Scan ankam, als die Session bereits
// abgeschlossen war. Der Handler übersetzt das in eine 409 — der Scan ist
// gegenstandslos, aber kein Fehler.
var ErrInventurAbgeschlossen = errors.New("inventur bereits abgeschlossen")

// VerbucheScan verbucht ein Exemplar als in dieser Session erfasst. Ein erneuter
// Scan desselben Exemplars in derselben Session ist ein No-op (Primärschlüssel).
//
// Nebenläufigkeit (Multi-Scanner-Betrieb): Der Scan koordiniert über die Session-Zeile
// gegen einen gleichzeitigen Abschluss. Er hält sie mit FOR SHARE, solange er die
// Erfassung schreibt; SchliesseLaufAb nimmt sie mit FOR UPDATE (siehe dort) und
// kann daher nicht committen, bevor dieser Scan fertig ist — der Verlust-Snapshot des
// Abschlusses sieht die Erfassung also garantiert. Ist die Session bereits abgeschlossen,
// findet der FOR-SHARE-Read keine offene Zeile und der Scan wird abgewiesen
// (ErrInventurAbgeschlossen), statt ins Leere zu laufen und ein physisch vorliegendes
// Buch als Verlust zurückzulassen.
func (a *InventurAbfragen) VerbucheScan(ctx context.Context, laufID, exemplarID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("scan-transaktion konnte nicht gestartet werden: %w", err)
	}
	defer db.SafeRollback(ctx, tx) // No-op nach Commit

	var id string
	if err := tx.QueryRow(ctx,
		`SELECT id FROM inventur_sessions WHERE id = $1 AND abgeschlossen_am IS NULL FOR SHARE`,
		laufID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInventurAbgeschlossen
		}
		return fmt.Errorf("session für scan sperren fehlgeschlagen: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO inventur_erfassungen (session_id, exemplar_id)
		VALUES ($1, $2)
		ON CONFLICT (session_id, exemplar_id) DO NOTHING
	`, laufID, exemplarID); err != nil {
		return fmt.Errorf("scan verbuchen fehlgeschlagen: %w", err)
	}
	return tx.Commit(ctx)
}

// IstExemplarImBereich prüft, ob ein Exemplar zu den Scope-Dimensionen einer Session gehört
// (Signatur/Fach/Klasse). Für globale Sessions ist immer true. Dient nur der
// nicht-blockierenden Scan-Warnung ("gehört nicht zum Scope") — physische Bedingungen
// (verliehen etc.) spielen hier keine Rolle, das Buch liegt ja gescannt in der Hand.
func (a *InventurAbfragen) IstExemplarImBereich(ctx context.Context, exemplarID string, bereich InventurBereich) (bool, error) {
	dimBedingung, dimArgs := bereich.dimensionBedingung(2)
	if dimBedingung == "TRUE" {
		return true, nil
	}
	args := append([]any{exemplarID}, dimArgs...)
	var vorhanden bool
	err := a.db.QueryRow(ctx, `
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
