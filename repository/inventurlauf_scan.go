package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// InventurExemplar ist, was ein Scan der Inventur vom Exemplar braucht.
type InventurExemplar struct {
	ExemplarID      string
	Titel           string
	CoverURL        string
	IstAusgesondert bool
	IstVerliehen    bool
}

// LadeExemplar lädt das Exemplar zu einem Barcode für den Scan der Inventur.
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

// ErrInventurAbgeschlossen meldet, dass der Scan ankam, als der Lauf schon abgeschlossen war.
// Die Tür macht daraus 409: Der Scan ist gegenstandslos, aber kein Fehler.
var ErrInventurAbgeschlossen = errors.New("inventur bereits abgeschlossen")

// VerbucheScan verbucht ein Exemplar als in diesem Lauf erfasst. Ein zweiter Scan desselben
// Exemplars im selben Lauf ändert nichts (Primärschlüssel).
//
// Scannen mehrere Geräte, stimmt sich der Scan über die Zeile des Laufs mit einem
// gleichzeitigen Abschluss ab: Er hält sie mit FOR SHARE, solange er die Erfassung schreibt.
// Der Abschluss nimmt sie mit FOR UPDATE (SperreLaufFuerAbschluss), wartet also auf den Scan
// und sieht dessen Erfassung sicher. Ist der Lauf schon abgeschlossen, findet der Scan keine
// offene Zeile und wird abgewiesen, statt ein Buch, das vorliegt, als Verlust zurückzulassen.
func (a *InventurAbfragen) VerbucheScan(ctx context.Context, laufID, exemplarID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("scan-transaktion konnte nicht gestartet werden: %w", err)
	}
	defer db.Verwerfe(ctx, tx) // nach dem Commit ohne Wirkung

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

// IstExemplarImBereich prüft, ob ein Exemplar zu Signatur, Fach und Jahrgang eines Laufs
// gehört; ohne Einschränkung gehört jedes dazu. Es dient nur der Warnung beim Scan („gehört
// nicht zum Bereich"), die den Scan nicht aufhält. Ob das Buch verliehen ist, spielt hier keine
// Rolle: Es liegt gescannt in der Hand.
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
