package repository

import (
	"context"
	"time"
)

// Klassenleitung ist eine Zeile der Zuordnung von Klasse und Adresse der Klassenleitung
// (klassen_lehrer_mapping). An diese Adresse gehen die Mahnlisten der Klasse.
type Klassenleitung struct {
	Klasse, LehrerEmail string
	ErstelltAm          time.Time
}

// ListeKlassenleitungen liefert alle Zuordnungen nach Klasse geordnet. Eine Zeile, die sich
// nicht lesen lässt, wird ausgelassen.
func ListeKlassenleitungen(ctx context.Context, db DBQueryer) ([]Klassenleitung, error) {
	rows, err := db.Query(ctx,
		`SELECT klasse, lehrer_email, erstellt_am FROM klassen_lehrer_mapping ORDER BY klasse`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zeilen []Klassenleitung
	for rows.Next() {
		var k Klassenleitung
		if err := rows.Scan(&k.Klasse, &k.LehrerEmail, &k.ErstelltAm); err != nil {
			continue
		}
		zeilen = append(zeilen, k)
	}
	return zeilen, rows.Err()
}

// KlassenleitungsAdressen liefert die Zuordnung als Nachschlagetabelle, die Klasse in der Form
// von KlassenSchluessel: „5A" in der Zuordnung trifft die Klasse „5a". Jeder Fehler bricht ab.
// Ein Versand, der mit einer halben Tabelle weiterliefe, hielte jede fehlende Klasse für eine
// ohne Adresse und meldete das wie ein Ergebnis.
func KlassenleitungsAdressen(ctx context.Context, db DBQueryer) (map[string]string, error) {
	rows, err := db.Query(ctx, `SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	adressen := map[string]string{}
	for rows.Next() {
		var klasse, email string
		if err := rows.Scan(&klasse, &email); err != nil {
			return nil, err
		}
		adressen[KlassenSchluessel(klasse)] = email
	}
	return adressen, rows.Err()
}

// SetzeKlassenleitung trägt die Adresse für die Klasse ein oder ersetzt sie. neu sagt, dass es
// für die Klasse noch keine Zuordnung gab, unveraendert, dass dieselbe Adresse schon dastand:
// Daran entscheidet der Aufrufer über den Eintrag im Protokoll.
func SetzeKlassenleitung(ctx context.Context, db DBQueryer, klasse, lehrerEmail string) (neu, unveraendert bool, err error) {
	err = db.QueryRow(ctx, `
			WITH alt AS (SELECT lehrer_email FROM klassen_lehrer_mapping WHERE klasse = $1)
			INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email)
			VALUES ($1, $2)
			ON CONFLICT (klasse) DO UPDATE SET lehrer_email = EXCLUDED.lehrer_email
			RETURNING NOT EXISTS (SELECT 1 FROM alt),
			          EXISTS (SELECT 1 FROM alt WHERE lehrer_email = $2)
		`, klasse, lehrerEmail).Scan(&neu, &unveraendert)
	return neu, unveraendert, err
}

// LoescheKlassenleitung entfernt die Zuordnung der Klasse und nennt die Zahl der getroffenen
// Zeilen; null heißt, es gab keine.
func LoescheKlassenleitung(ctx context.Context, db DBQueryer, klasse string) (int64, error) {
	tag, err := db.Exec(ctx,
		`DELETE FROM klassen_lehrer_mapping WHERE klasse = $1`, klasse)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
