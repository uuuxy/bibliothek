package inventur

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeleteBooks deletes multiple book records.
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

	// Clean up related records for ALL copies of these titles to prevent ON DELETE RESTRICT errors
	if _, err := repo.db.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[]))", ids); err != nil {
		return fmt.Errorf("failed to delete damage records for titles: %w", err)
	}
	if _, err := repo.db.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[])) AND rueckgabe_am IS NOT NULL", ids); err != nil {
		return fmt.Errorf("failed to delete past loans for titles: %w", err)
	}

	result, err := repo.db.Exec(ctx, `DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return fmt.Errorf("bücher konnten nicht gelöscht werden: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}

	loescheLokaleCoverDateien(localCovers)
	return nil
}

// pruefeKeineAktivenAusleihen bricht ab, wenn zu einem der Titel noch ein Exemplar
// aktuell verliehen ist.
func (repo *BookRepository) pruefeKeineAktivenAusleihen(ctx context.Context, ids []string) error {
	var activeLoans int
	err := repo.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		WHERE e.titel_id = ANY($1::uuid[]) AND a.rueckgabe_am IS NULL`, ids).Scan(&activeLoans)
	if err != nil {
		return fmt.Errorf("fehler bei der prüfung auf aktive ausleihen: %w", err)
	}
	if activeLoans > 0 {
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
		// #nosec G304 - name is sanitized using filepath.Base
		_ = os.Remove(filepath.Join("uploads", name))
	}
}
