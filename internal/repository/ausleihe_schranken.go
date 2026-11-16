package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ZaehleOffeneBuechereiAusleihen zählt die offenen Ausleihen eines Lesers, die gegen das
// Ausleihlimit zählen: Bücher, die kein Lernmittel sind.
func ZaehleOffeneBuechereiAusleihen(ctx context.Context, q Datenbank, leserID string) (int, error) {
	var anzahl int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ausleihen a
		JOIN buecher_exemplare be ON a.exemplar_id = be.id
		JOIN buecher_titel bt ON be.titel_id = bt.id
		WHERE a.schueler_id = $1
		  AND a.rueckgabe_am IS NULL
		  AND NOT bt.ist_lernmittel
	`, leserID).Scan(&anzahl)
	return anzahl, err
}

// AbholfachVormerkung nennt, für wen ein Exemplar im Abholfach liegt.
type AbholfachVormerkung struct {
	LeserID  string
	Vorname  string
	Nachname string
}

// FindeVormerkungAmExemplar liefert, für wen das Exemplar im Abholfach liegt, solange die
// Abholfrist läuft. bereit ist false, wenn es für niemanden bereitliegt.
func FindeVormerkungAmExemplar(ctx context.Context, q Datenbank, exemplarID string) (vormerkung AbholfachVormerkung, bereit bool, err error) {
	err = q.QueryRow(ctx, `
		SELECT v.schueler_id, s.vorname, s.nachname
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.bereitgestellt_exemplar_id = $1
		  AND v.status = 'abholbereit'
		  AND v.bereitgestellt_bis > CURRENT_TIMESTAMP
	`, exemplarID).Scan(&vormerkung.LeserID, &vormerkung.Vorname, &vormerkung.Nachname)
	if errors.Is(err, pgx.ErrNoRows) {
		return vormerkung, false, nil
	}
	if err != nil {
		return vormerkung, false, err
	}
	return vormerkung, true, nil
}
