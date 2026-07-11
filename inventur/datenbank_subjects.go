package inventur

import (
	"context"
	"fmt"
)

func (repo *BookRepository) GetActiveSubjects(ctx context.Context) ([]Subject, error) {
	query := `
		SELECT DISTINCT fach
		FROM buecher
		WHERE fach IS NOT NULL AND fach != ''
		ORDER BY fach ASC
	`
	rows, err := repo.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fächer konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	var subjects []Subject
	idCounter := 1
	for rows.Next() {
		var fachName string
		if err := rows.Scan(&fachName); err != nil {
			return nil, fmt.Errorf("fehler beim lesen der fächer: %w", err)
		}
		subjects = append(subjects, Subject{
			ID:       idCounter,
			Name:     fachName,
			IsActive: true,
		})
		idCounter++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fehler beim lesen der fächer: %w", err)
	}
	return subjects, nil
}
