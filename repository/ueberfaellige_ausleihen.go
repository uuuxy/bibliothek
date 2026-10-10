package repository

import "context"

// ZaehleUeberfaelligeBuecher zählt die offenen Ausleihen eines Lesers, deren Frist seit mehr als
// kulanzTage Tagen abgelaufen ist. Dauerleihen und Geräte zählen nicht: Eine Dauerleihe wird
// nie überfällig, und die Sperr-Automatik der Theke fragt nach Büchern.
func ZaehleUeberfaelligeBuecher(ctx context.Context, q DBQueryer, leserID string, kulanzTage int) (int, error) {
	var n int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ausleihen
		WHERE schueler_id = $1
		  AND rueckgabe_am IS NULL
		  AND rueckgabe_frist < CURRENT_TIMESTAMP - (INTERVAL '1 day' * $2)
		  AND ist_handapparat = false
		  AND geraet_id IS NULL
	`, leserID, kulanzTage).Scan(&n)
	return n, err
}
