package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// FindeFremdrueckgabeZeitpunkt liefert den Zeitpunkt der Rückgabe einer Ausleihe, die als
// Fremdrückgabe dieses Exemplars beendet wurde; nil, wenn es sie so nicht gibt.
//
// Das Nachbuchen fragt das für einen Eintrag, unter dessen Schlüssel der Scan an der Theke nur
// die Fremdrückgabe gebucht hat: Es legt den Scan auf diesen Zeitpunkt, und der Wächter
// entscheidet, ob sich das Exemplar seitdem bewegt hat (internal/service/nachbuchen.go). Ohne
// ist_fremdrueckgabe genügte der Verweis auf irgendeine beendete Ausleihe desselben Exemplars.
func FindeFremdrueckgabeZeitpunkt(ctx context.Context, q Datenbank, ausleiheID, exemplarID string) (*time.Time, error) {
	var rueckgabe *time.Time
	err := q.QueryRow(ctx, `
		SELECT rueckgabe_am FROM ausleihen
		WHERE id = $1 AND exemplar_id = $2 AND ist_fremdrueckgabe`, ausleiheID, exemplarID).Scan(&rueckgabe)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fremdrückgabe lesen: %w", err)
	}
	return rueckgabe, nil
}
