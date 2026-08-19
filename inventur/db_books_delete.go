package inventur

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"bibliothek/db"
)

// DeleteBooks deletes multiple book records.
//
// Die drei abhängigen DELETEs (Schadensfälle, Ausleihhistorie, Titel — der Titel-Delete
// räumt die Exemplare per ON DELETE CASCADE mit) laufen in EINER Transaktion. Ohne sie
// hinterließ ein Crash zwischen Schritt 2 und 3 einen bösartigen Halbzustand: Gebühren-
// und Ausleihhistorie der Titel gelöscht, Buch und Exemplare aber erhalten — eine offene
// Gebühr verschwände unwiederbringlich, während das Buch weiter im Bestand steht.
func (repo *BookRepository) DeleteBooks(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	if err := repo.pruefeKeineAktivenAusleihen(ctx, ids); err != nil {
		return err
	}

	localCovers, err := repo.sammleLokaleCoverPfade(ctx, ids)
	if err != nil {
		return err
	}

	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("löschen konnte nicht begonnen werden: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Zugehörige Datensätze ALLER Exemplare dieser Titel entfernen, sonst greift der
	// ON DELETE RESTRICT der FKs. Reihenfolge erzwingen die RESTRICT-FKs, Atomarität die Tx.
	if _, err := tx.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[]))", ids); err != nil {
		return fmt.Errorf("failed to delete damage records for titles: %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[])) AND rueckgabe_am IS NOT NULL", ids); err != nil {
		return fmt.Errorf("failed to delete past loans for titles: %w", err)
	}

	result, err := tx.Exec(ctx, `DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return fmt.Errorf("bücher konnten nicht gelöscht werden: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("löschen konnte nicht abgeschlossen werden: %w", err)
	}

	// Erst nach dem Commit: die lokalen Cover-Dateien sind unwiederbringlich, das darf
	// NICHT geschehen, solange das DB-Löschen noch scheitern (zurückrollen) könnte.
	loescheLokaleCoverDateien(localCovers)
	return nil
}

// pruefeKeineAktivenAusleihen bricht ab, wenn zu einem der Titel noch ein Exemplar
// aktuell verliehen ist.
func (repo *BookRepository) pruefeKeineAktivenAusleihen(ctx context.Context, ids []string) error {
	var hasActiveLoans bool
	err := repo.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			WHERE e.titel_id = ANY($1::uuid[]) AND a.rueckgabe_am IS NULL
		)`, ids).Scan(&hasActiveLoans)
	if err != nil {
		return fmt.Errorf("fehler bei der prüfung auf aktive ausleihen: %w", err)
	}
	if hasActiveLoans {
		return fmt.Errorf("löschen abgebrochen: Mindestens ein Exemplar dieser Titel ist aktuell verliehen")
	}
	return nil
}

// sammleLokaleCoverPfade liefert die lokal gespeicherten Cover-Pfade (/uploads/...)
// der angegebenen Titel, damit sie nach dem Löschen entfernt werden können.
func (repo *BookRepository) sammleLokaleCoverPfade(ctx context.Context, ids []string) ([]string, error) {
	coverRows, err := repo.db.Query(ctx, "SELECT cover_url FROM buecher_titel WHERE id = ANY($1::uuid[]) AND cover_url LIKE '/uploads/%'", ids)
	if err != nil {
		return nil, fmt.Errorf("cover-dateien konnten nicht ermittelt werden: %w", err)
	}
	defer coverRows.Close()

	localCovers := make([]string, 0)
	for coverRows.Next() {
		var coverURL string
		if scanErr := coverRows.Scan(&coverURL); scanErr != nil {
			return nil, fmt.Errorf("cover-pfade konnten nicht gelesen werden: %w", scanErr)
		}
		localCovers = append(localCovers, coverURL)
	}
	if rowsErr := coverRows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("cover-pfade konnten nicht iteriert werden: %w", rowsErr)
	}
	return localCovers, nil
}

// loescheLokaleCoverDateien entfernt die lokalen Cover-Dateien (best-effort; Fehler
// werden ignoriert, da der DB-Datensatz bereits gelöscht ist).
func loescheLokaleCoverDateien(localCovers []string) {
	for _, coverURL := range localCovers {
		if !strings.HasPrefix(coverURL, "/uploads/") {
			continue
		}
		name := filepath.Base(coverURL)
		if name == "" || name == "." || name == "/" {
			continue
		}
		// loescheUploadDatei arbeitet über os.Root (uploads_pfad.go) — die Einhegung auf
		// das Upload-Verzeichnis macht das Betriebssystem, nicht mehr das filepath.Base
		// oben. Deshalb steht hier kein #nosec mehr: Es gibt kein os.Remove, das gosec
		// beanstanden könnte.
		_ = loescheUploadDatei(name) //nolint:errcheck // Best-Effort-Aufräumen nach dem DB-Löschen
	}
}
