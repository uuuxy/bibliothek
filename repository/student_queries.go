package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// GetByBarcode liest einen Schüler anhand seiner Barcode-ID aus.
func (r *pgStudentRepository) GetByBarcode(ctx context.Context, barcode string) (*Student, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, '')
		FROM schueler
		WHERE barcode_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	s, err := scanStudent(r.db.QueryRow(ctx, query, barcode))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// GetByID liest einen Schüler anhand seiner UUID aus.
func (r *pgStudentRepository) GetByID(ctx context.Context, id string) (*Student, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, '')
		FROM schueler
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	s, err := scanStudent(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// suchTokens zerlegt eine Eingabe in einzelne Suchbegriffe und entschärft die
// LIKE-Metazeichen. Ohne das Escaping wäre ein getipptes "%" eine Wildcard, die
// die halbe Schülerschaft zurückgibt, und "_" ein Platzhalter für ein Zeichen.
func suchTokens(queryText string) []string {
	felder := strings.Fields(queryText)
	tokens := make([]string, 0, len(felder))
	for _, f := range felder {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(f)
		tokens = append(tokens, escaped)
	}
	return tokens
}

// SearchStudentsFuzzy durchsucht die Schülerschaft nach Namen oder Barcodes und
// liefert zusätzlich die Gesamtzahl der Treffer (nicht nur die des Limits).
//
// Zwei Eigenschaften, die der frühere Ganzstring-Vergleich nicht hatte:
//
//  1. Jedes Token muss irgendeine Spalte treffen (UND-Verknüpfung), nicht die
//     komplette Eingabe eine einzelne. Damit ist die Reihenfolge von Vor- und
//     Nachname bedeutungslos — "Lena Hoffmann" und "Hoffmann Lena" finden dieselbe
//     Person, und mehrteilige Namen ("Anna Maria", "García Rodríguez") ebenso.
//  2. Verglichen wird über die Normalform suchnorm() statt über den Rohtext. Sie
//     faltet Diakritika weg ("Garcia" findet García, "Ozturk" findet Öztürk) UND
//     zieht die deutsche Ersatzschreibung auf denselben Nenner ("Mueller" findet
//     Müller, "Oeztuerk" findet Öztürk, "Strasse" findet Straße) — in beide
//     Richtungen, denn mal steht der Umlaut in der Datenbank und wird ohne getippt,
//     mal ist es umgekehrt.
//
// Sortiert wird nach Trefferqualität: ein Token am Wortanfang des Nachnamens wiegt
// schwerer als eines im Vornamen, beides zusammen schlägt alles. Sonst stünde bei
// "max hoffmann" der gesuchte Schüler irgendwo alphabetisch in der Liste.
func (r *pgStudentRepository) SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, int, error) {
	tokens := suchTokens(queryText)
	if len(tokens) == 0 {
		return nil, 0, nil
	}

	// $2 ist das erste Token separat: Als direkte LIKE-Bedingung kann der Planer
	// dafür die Trigramm-Indizes aus Migration 054 ziehen (BitmapOr über alle drei).
	// Die vollständige UND-Prüfung über alle Tokens ($1) filtert danach exakt nach.
	//
	// Der Ankerausdruck muss BUCHSTÄBLICH dem Index entsprechen: lower(barcode_id),
	// nicht lower(coalesce(barcode_id, '')). Mit dem coalesce ist dieser eine
	// OR-Zweig nicht indexierbar — und ein einziger nicht-indexierbarer Zweig kippt
	// die gesamte Abfrage in den Seq Scan (gemessen: 30 ms statt 0,1 ms bei 20.000
	// Schülern). Ein NULL-Barcode ergibt hier NULL statt false, was in der
	// OR-Verknüpfung dasselbe Ergebnis liefert.
	query := `
		WITH tokens AS (
			SELECT suchnorm(t) AS norm, lower(t) AS roh FROM unnest($1::text[]) AS t
		)
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, ''),
		       count(*) OVER () AS gesamt
		FROM schueler s
		WHERE s.deleted_at IS NULL
		  AND (
			   suchnorm(s.vorname)   LIKE '%' || suchnorm($2::text) || '%'
			OR suchnorm(s.nachname)  LIKE '%' || suchnorm($2::text) || '%'
			OR lower(s.barcode_id)   LIKE '%' || lower($2::text) || '%'
		  )
		  AND (
			SELECT bool_and(
				   suchnorm(coalesce(s.vorname, ''))    LIKE '%' || tokens.norm || '%'
				OR suchnorm(coalesce(s.nachname, ''))   LIKE '%' || tokens.norm || '%'
				OR lower(coalesce(s.barcode_id, ''))    LIKE '%' || tokens.roh || '%')
			FROM tokens
		  )
		ORDER BY
			  (SELECT count(*) FROM tokens WHERE suchnorm(coalesce(s.nachname, '')) LIKE tokens.norm || '%') * 2
			+ (SELECT count(*) FROM tokens WHERE suchnorm(coalesce(s.vorname, ''))  LIKE tokens.norm || '%') DESC,
			s.nachname ASC, s.vorname ASC
		LIMIT $3
	`
	rows, err := r.db.Query(ctx, query, tokens, tokens[0], limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []Student
	gesamt := 0
	for rows.Next() {
		var zeilenGesamt int
		s, err := scanStudentMitZusatz(rows, &zeilenGesamt)
		if err != nil {
			return nil, 0, err
		}
		gesamt = zeilenGesamt
		results = append(results, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return results, gesamt, nil
}
