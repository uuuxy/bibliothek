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

// KlassenleitungVersetzung ist eine Zeile der Zuordnung mit ihrem Namen nach dem
// Schuljahreswechsel. Entfaellt heißt: Die Klasse gibt es danach nicht mehr (Abschlussklasse)
// oder sie wird neu gebildet (nach der 6 und nach der 10).
type KlassenleitungVersetzung struct {
	Alt, Neu  string
	Entfaellt bool
}

// LeseKlassenleitungVersetzungen liest die Zuordnungen absteigend nach Stufe und liest sie ganz,
// bevor die Transaktion wieder schreibt.
func LeseKlassenleitungVersetzungen(ctx context.Context, db DBQueryer) ([]KlassenleitungVersetzung, error) {
	rows, err := db.Query(ctx, `
		SELECT klasse,
		       lpad((substring(klasse from '^\d+')::int + 1)::text,
		            greatest(length(substring(klasse from '^\d+')), length((substring(klasse from '^\d+')::int + 1)::text)), '0')
		         || substring(klasse from '^\d+(.*)$') AS neue_klasse,
		       (`+AbschlussklasseSQL("klasse")+`
		        OR substring(klasse from '^\d+')::int IN (6, 10)) AS entfaellt
		FROM klassen_lehrer_mapping
		WHERE klasse ~ '^\d+'
		ORDER BY substring(klasse from '^\d+')::int DESC, klasse DESC`)
	if err != nil {
		return nil, err
	}
	var zeilen []KlassenleitungVersetzung
	for rows.Next() {
		var z KlassenleitungVersetzung
		if err := rows.Scan(&z.Alt, &z.Neu, &z.Entfaellt); err != nil {
			rows.Close()
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return zeilen, nil
}

// KlassenleitungVorhanden sagt, ob es für die Klasse eine Zuordnung gibt.
func KlassenleitungVorhanden(ctx context.Context, db DBQueryer, klasse string) (bool, error) {
	var vorhanden bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM klassen_lehrer_mapping WHERE klasse = $1)`, klasse).Scan(&vorhanden)
	return vorhanden, err
}

// BenenneKlassenleitungUm hängt die Zuordnung einer Klasse an deren neuen Namen.
func BenenneKlassenleitungUm(ctx context.Context, db DBQueryer, neu, alt string) error {
	_, err := db.Exec(ctx, `UPDATE klassen_lehrer_mapping SET klasse = $1 WHERE klasse = $2`, neu, alt)
	return err
}
