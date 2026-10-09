package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"

	"bibliothek/pkg/coverablage"
)

// titelCoverLeser ist, was LokaleCoverNurDieserTitel braucht: eine Transaktion oder der Pool.
type titelCoverLeser interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// LokaleCoverNurDieserTitel liefert die lokal gespeicherten Cover (/uploads/…) der Titel,
// die kein weiterer Titel trägt: die Dateien, die nach dem Löschen dieser Titel niemandem
// mehr gehören. Beide Lösch-Türen eines Titels fragen hier, vor dem Löschen und in derselben
// Transaktion; entfernt werden die Dateien erst nach dem Commit (LoescheCoverDateien).
func LokaleCoverNurDieserTitel(ctx context.Context, q titelCoverLeser, ids []string) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT t.cover_url FROM buecher_titel t
		WHERE t.id = ANY($1::uuid[]) AND t.cover_url LIKE '/uploads/%'
		  AND NOT EXISTS (
		      SELECT 1 FROM buecher_titel anderer
		      WHERE anderer.cover_url = t.cover_url AND anderer.id <> ALL($1::uuid[]))`, ids)
	if err != nil {
		return nil, fmt.Errorf("cover-dateien konnten nicht ermittelt werden: %w", err)
	}
	cover, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("cover-pfade konnten nicht gelesen werden: %w", err)
	}
	return cover, nil
}

// LoescheCoverDateien entfernt die Dateien gelöschter Titel. Erst nach dem Commit aufrufen:
// Eine Datei kommt nicht zurück, wenn die Transaktion doch scheitert. Ein Fehler hält nichts
// auf, der Titel ist gelöscht.
func LoescheCoverDateien(coverURLs []string) {
	for _, coverURL := range coverURLs {
		if err := coverablage.Loesche(coverURL); err != nil {
			log.Printf("Cover-Datei %q nicht entfernt: %v", coverURL, err)
		}
	}
}

// CoverAbgleichAuswahl wählt die Titel, die der Cover-Abgleich anfasst: noch nie versuchte,
// gescheiterte und solche mit einer Adresse außerhalb der eigenen Ablage; deren Bild holt der
// Abgleich als lokale Datei. Einen Titel mit lokalem Cover wählt sie nie, was auch immer
// cover_status sagt: Ein von Hand hochgeladenes Cover bleibt stehen.
const CoverAbgleichAuswahl = `
		SELECT id, isbn FROM buecher_titel
		WHERE isbn IS NOT NULL AND isbn != ''
		  AND ` + coverNichtLokal + `
		  AND (
		        cover_status IN ('PENDING', 'FAILED')
		     OR COALESCE(cover_url, '') <> ''
		      )`

// coverNichtLokal heißt: Der Titel trägt kein lokal liegendes Cover. Die Bedingung gilt beim
// Auswählen und noch einmal bei jedem Schreiben. Dazwischen liegt die Laufzeit des Abgleichs;
// ein in dieser Zeit von Hand hochgeladenes Cover bleibt mit seinem Stand stehen.
const coverNichtLokal = `COALESCE(cover_url, '') NOT LIKE '/uploads/%'`

// TitelMitISBN ist ein Titel, wie der Cover-Abgleich ihn braucht.
type TitelMitISBN struct {
	ID   string
	ISBN string
}

// TitelFuerCoverAbgleich liest die Titel aus CoverAbgleichAuswahl. Die Zeilen sind gelesen und
// geschlossen, bevor der Abgleich je Titel schreibt.
func TitelFuerCoverAbgleich(ctx context.Context, db DBQueryer) ([]TitelMitISBN, error) {
	rows, err := db.Query(ctx, CoverAbgleichAuswahl)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var titel []TitelMitISBN
	for rows.Next() {
		var t TitelMitISBN
		if err := rows.Scan(&t.ID, &t.ISBN); err == nil {
			titel = append(titel, t)
		}
	}
	return titel, rows.Err()
}

// SetzeGefundenesCover trägt die Adresse des gefundenen Covers ein, solange der Titel kein
// lokales trägt.
func SetzeGefundenesCover(ctx context.Context, db DBQueryer, titelID, coverURL string) error {
	_, err := db.Exec(ctx, `UPDATE buecher_titel SET cover_url = $1, cover_status = 'FOUND' WHERE id = $2 AND `+coverNichtLokal, coverURL, titelID)
	return err
}

// SetzeCoverStatus merkt das Ergebnis des Abgleichs am Titel, solange er kein lokales Cover
// trägt.
func SetzeCoverStatus(ctx context.Context, db DBQueryer, titelID, status string) error {
	_, err := db.Exec(ctx, `UPDATE buecher_titel SET cover_status = $1 WHERE id = $2 AND `+coverNichtLokal, status, titelID)
	return err
}
