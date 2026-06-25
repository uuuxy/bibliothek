package inventur

import (
	"context"
	"fmt"
)

func (repo *BookRepository) GetActiveSubjects(ctx context.Context) ([]Subject, error) {
	query := "SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC"
	rows, err := repo.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fächer konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	var subjects []Subject
	for rows.Next() {
		var s Subject
		if err := rows.Scan(&s.ID, &s.Name, &s.IsActive); err != nil {
			return nil, fmt.Errorf("fehler beim lesen der fächer: %w", err)
		}
		subjects = append(subjects, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fehler beim lesen der fächer: %w", err)
	}
	return subjects, nil
}
