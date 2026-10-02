package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"

	"bibliothek/pkg/coverdatei"
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
		if err := coverdatei.Loesche(coverURL); err != nil {
			log.Printf("Cover-Datei %q nicht entfernt: %v", coverURL, err)
		}
	}
}
