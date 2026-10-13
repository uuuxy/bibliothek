package repository

import (
	"context"
	"time"
)

// AbholbereiterTitel ist ein Titel, der für einen Leser im Abholfach liegt.
type AbholbereiterTitel struct {
	Titel             string
	BereitgestelltBis *time.Time
}

// LadeAbholbereiteTitel liefert die Titel, die für den Leser im Abholfach liegen, höchstens
// fünf und den mit der kürzesten Abholfrist zuerst: Der Hinweis an der Theke nennt sie einzeln.
func LadeAbholbereiteTitel(ctx context.Context, q Datenbank, leserID string) ([]AbholbereiterTitel, error) {
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

	var titel []AbholbereiterTitel
	for rows.Next() {
		var t AbholbereiterTitel
		if err := rows.Scan(&t.Titel, &t.BereitgestelltBis); err != nil {
			return nil, err
		}
		titel = append(titel, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return titel, nil
}
