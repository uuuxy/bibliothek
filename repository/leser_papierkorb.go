package repository

import (
	"context"
	"time"
)

// PapierkorbZeile ist ein Leser im Papierkorb. AnonymizedAt ist gesetzt, wenn die Zeile nach
// der Frist anonymisiert wurde: Dann ist sie kein Leser mehr, sondern ein Pseudonym.
type PapierkorbZeile struct {
	ID            string
	Barcode       string
	Vorname       string
	Nachname      string
	Klasse        string
	AbgaengerJahr *int
	Gesperrt      bool
	DeletedAt     time.Time
	AnonymizedAt  *time.Time
}

// ListePapierkorb liefert die Leser im Papierkorb, die jüngsten Löschungen zuerst und
// höchstens limit Zeilen.
func ListePapierkorb(ctx context.Context, db DBQueryer, limit int) ([]PapierkorbZeile, error) {
	rows, err := db.Query(ctx, `
			SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''),
			       coalesce(klasse, ''), abgaenger_jahr, coalesce(ist_gesperrt, false), deleted_at,
			       anonymized_at
			FROM leser
			WHERE deleted_at IS NOT NULL
			ORDER BY deleted_at DESC
			LIMIT $1
		`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := []PapierkorbZeile{}
	for rows.Next() {
		var z PapierkorbZeile
		if err := rows.Scan(&z.ID, &z.Barcode, &z.Vorname, &z.Nachname, &z.Klasse, &z.AbgaengerJahr, &z.Gesperrt, &z.DeletedAt, &z.AnonymizedAt); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
}

// PapierkorbLeserAnonymisiert sagt, ob ein Leser im Papierkorb schon anonymisiert ist;
// pgx.ErrNoRows, wenn er nicht im Papierkorb liegt.
func PapierkorbLeserAnonymisiert(ctx context.Context, db DBQueryer, id string) (bool, error) {
	var anonymisiert bool
	err := db.QueryRow(ctx,
		`SELECT anonymized_at IS NOT NULL FROM leser WHERE id = $1 AND deleted_at IS NOT NULL`, id,
	).Scan(&anonymisiert)
	return anonymisiert, err
}

// StelleLeserWiederHer holt einen Leser aus dem Papierkorb und hebt die Sperre auf, die das
// Löschen gesetzt hat; eine Sperre aus anderem Grund bleibt. Liefert die Zahl der Zeilen: null
// heißt, er lag nicht im Papierkorb. Ein Zwilling, der inzwischen seinen Platz besetzt, kommt
// als Verletzung eines der Teilindizes der aktiven Leser zurück.
func StelleLeserWiederHer(ctx context.Context, db DBQueryer, id string) (int64, error) {
	tag, err := db.Exec(ctx, `
			UPDATE leser SET
				deleted_at = NULL,
				ist_gesperrt = CASE WHEN block_reason = 'Systematisch gelöscht' THEN false ELSE ist_gesperrt END,
				block_reason = CASE WHEN block_reason = 'Systematisch gelöscht' THEN NULL ELSE block_reason END,
				aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1 AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
