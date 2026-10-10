package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ZaehleOffeneBuechereiAusleihen zählt die offenen Ausleihen eines Lesers, die gegen das
// Ausleihlimit zählen: Bücher, die kein Lernmittel sind.
func ZaehleOffeneBuechereiAusleihen(ctx context.Context, q DBQueryer, leserID string) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ausleihen a
		JOIN buecher_exemplare be ON a.exemplar_id = be.id
		JOIN buecher_titel bt ON be.titel_id = bt.id
		WHERE a.schueler_id = $1
		  AND a.rueckgabe_am IS NULL
		  AND NOT bt.ist_lernmittel
	`, leserID).Scan(&count)
	return count, err
}

// AbholfachReservierung nennt, für wen ein Exemplar im Abholfach liegt.
type AbholfachReservierung struct {
	LeserID  string
	Vorname  string
	Nachname string
}

// ReservierungAmExemplar liefert, für wen das Exemplar im Abholfach liegt, solange die
// Abholfrist läuft. bereit ist false, wenn es für niemanden bereitliegt.
func ReservierungAmExemplar(ctx context.Context, q DBQueryer, exemplarID string) (reservierung AbholfachReservierung, bereit bool, err error) {
	err = q.QueryRow(ctx, `
		SELECT v.schueler_id, s.vorname, s.nachname
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.bereitgestellt_exemplar_id = $1
		  AND v.status = 'abholbereit'
		  AND v.bereitgestellt_bis > CURRENT_TIMESTAMP
	`, exemplarID).Scan(&reservierung.LeserID, &reservierung.Vorname, &reservierung.Nachname)
	if errors.Is(err, pgx.ErrNoRows) {
		return reservierung, false, nil
	}
	if err != nil {
		return reservierung, false, err
	}
	return reservierung, true, nil
}
