package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// LoescheErfuellteVormerkung löscht die Vormerkung des Lesers für den Titel, den er gerade
// ausgeliehen hat, und liefert das Exemplar, das für ihn bereitlag. Das Ergebnis ist nil, wenn
// er keine Vormerkung hatte oder noch nichts bereitlag.
func LoescheErfuellteVormerkung(ctx context.Context, q DBQueryer, titelID, leserID string) (*string, error) {
	var bereitgestellt *string
	err := q.QueryRow(ctx,
		`DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2
		 RETURNING bereitgestellt_exemplar_id`,
		titelID, leserID).Scan(&bereitgestellt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return bereitgestellt, err
}

// ExemplarNummer liefert die Nummer auf dem Etikett eines Exemplars.
func ExemplarNummer(ctx context.Context, q DBQueryer, exemplarID string) (string, error) {
	var nummer string
	err := q.QueryRow(ctx,
		`SELECT barcode_id FROM buecher_exemplare WHERE id = $1`, exemplarID).Scan(&nummer)
	return nummer, err
}

// WartendeVormerkung ist die Vormerkung, die als nächste bedient wird, mit dem Namen des
// Wartenden für die Meldung an der Theke.
type WartendeVormerkung struct {
	ID       string
	Vorname  string
	Nachname string
	Klasse   string
}

// SperreNaechsteWartendeVormerkung liefert die älteste wartende Vormerkung des Titels von einem
// Schüler, der abholen darf, und sperrt ihre Zeile bis zum Ende der Transaktion. ohneLeser
// nimmt einen Leser aus: Wer das Buch gerade zurückgibt, stellt es sich nicht selbst wieder
// bereit. Gesperrt wird nur die Vormerkung, nicht die Zeile des Schülers; die sperrt die
// Ausleihe zuerst, und in umgekehrter Reihenfolge warteten beide aufeinander. Eine Vormerkung,
// die eine andere Rückgabe desselben Titels gerade bedient, wird übergangen. da ist false,
// wenn niemand wartet.
func SperreNaechsteWartendeVormerkung(ctx context.Context, tx pgx.Tx, titelID string, ohneLeser *string) (wartende WartendeVormerkung, da bool, err error) {
	err = tx.QueryRow(ctx, `
		SELECT v.id, s.vorname, s.nachname, COALESCE(s.klasse, '')
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.titel_id = $1 AND v.status = 'wartend'
		  AND `+sqlWartenderDarfAbholen+`
		  AND ($2::uuid IS NULL OR v.schueler_id <> $2::uuid)
		ORDER BY v.erstellt_am ASC LIMIT 1
		FOR UPDATE OF v SKIP LOCKED
	`, titelID, ohneLeser).Scan(&wartende.ID, &wartende.Vorname, &wartende.Nachname, &wartende.Klasse)
	if errors.Is(err, pgx.ErrNoRows) {
		return wartende, false, nil
	}
	if err != nil {
		return wartende, false, err
	}
	return wartende, true, nil
}

// StelleVormerkungBereit teilt der Vormerkung das Exemplar zu: Es liegt bis zum Ende der
// Abholfrist für den Wartenden im Abholfach. Gibt es die Vormerkung nicht, ist das
// ErrVormerkungNichtGefunden.
func StelleVormerkungBereit(ctx context.Context, q DBQueryer, vormerkungID, exemplarID string, abholfrist time.Time) error {
	tag, err := q.Exec(ctx, "UPDATE vormerkungen SET status = 'abholbereit', bereitgestellt_exemplar_id = $1, bereitgestellt_bis = $3 WHERE id = $2", exemplarID, vormerkungID, abholfrist)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrVormerkungNichtGefunden
	}
	return nil
}
