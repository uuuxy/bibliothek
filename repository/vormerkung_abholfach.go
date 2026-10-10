package repository

import (
	"context"
	"time"
)

// AbholbereitesBuch ist ein Buch, das für einen Leser im Abholfach liegt.
type AbholbereitesBuch struct {
	Titel             string
	BereitgestelltBis *time.Time
}

// AbholbereiteBuecher liefert die Bücher, die für den Leser im Abholfach liegen, höchstens fünf
// und das mit der kürzesten Abholfrist zuerst: Der Hinweis an der Theke nennt sie einzeln.
func AbholbereiteBuecher(ctx context.Context, q DBQueryer, leserID string) ([]AbholbereitesBuch, error) {
	rows, err := q.Query(ctx, `
		SELECT t.titel, v.bereitgestellt_bis
		FROM vormerkungen v
		JOIN buecher_titel t ON t.id = v.titel_id
		WHERE v.schueler_id = $1 AND v.status = 'abholbereit'
		ORDER BY v.bereitgestellt_bis ASC NULLS LAST
		LIMIT 5`, leserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buecher []AbholbereitesBuch
	for rows.Next() {
		var b AbholbereitesBuch
		if err := rows.Scan(&b.Titel, &b.BereitgestelltBis); err != nil {
			return nil, err
		}
		buecher = append(buecher, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buecher, nil
}
