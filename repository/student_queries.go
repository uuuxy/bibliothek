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

// Die drei folgenden SQL-Bausteine sind die EINE Schülersuche der Anwendung. Sie stehen
// hier als Konstanten, weil zwei Ansichten dieselbe Suche brauchen, aber verschiedene
// Spalten zurückgeben: die Omnibox an der Theke (SearchStudentsFuzzy, Ausleihe) und die
// Schülerdatei mit ihren Ausleihzahlen (ListStudentsWithStats). Vorher hatte die
// Schülerdatei gar keine Serversuche und filterte im Browser über die ersten 500 Zeilen —
// wer dahinter lag, war schlicht nicht auffindbar.
//
// Belegung der Platzhalter in beiden Abfragen gleich: $1 = Tokens (text[]), $2 = erstes
// Token. Der Tabellen-Alias muss s sein.

// SchuelerSuchCTE zerlegt die Tokens in Normalform (diakritikfrei) und Rohform.
const SchuelerSuchCTE = `
	WITH tokens AS (
		SELECT suchnorm(t) AS norm, lower(t) AS roh FROM unnest($1::text[]) AS t
	)`

// SchuelerSuchBedingung ist das eigentliche Prädikat: Jedes Token muss irgendeine Spalte
// treffen (UND-Verknüpfung), verglichen über die Normalform.
//
// mitKlasse schaltet die Klasse als zusätzliches Suchfeld frei. Die Schülerdatei braucht
// das (ihr Filter konnte es immer schon, "10a" listet die Klasse), die Omnibox an der
// Theke nicht — dort sucht man einen Menschen, keine Klasse, und 30 Namen im Vorschlag
// wären dort im Weg. Weil jedes Token einzeln geprüft wird, bedeutet "10a Müller"
// übrigens genau das Richtige: Klasse 10a UND Name Müller.
//
// Die vorangestellte Ankerbedingung auf $2 ist Absicht und kein Duplikat: Als direkte
// LIKE-Bedingung kann der Planer dafür die Trigramm-Indizes aus Migration 054 ziehen
// (BitmapOr über alle drei). Sie muss BUCHSTÄBLICH dem Index entsprechen — lower(barcode_id),
// nicht lower(coalesce(barcode_id, ”)). Mit dem coalesce ist dieser eine OR-Zweig nicht
// indexierbar, und ein einziger nicht-indexierbarer Zweig kippt die gesamte Abfrage in den
// Seq Scan (gemessen: 30 ms statt 0,1 ms bei 20.000 Schülern). Ein NULL-Barcode ergibt hier
// NULL statt false, was in der OR-Verknüpfung dasselbe Ergebnis liefert. Die Klasse steht
// bewusst NICHT im Anker: Sie hat keinen Trigramm-Index, sie erweitert nur die Nachprüfung.
func SchuelerSuchBedingung(mitKlasse bool) string {
	klasse := ""
	if mitKlasse {
		klasse = "\n\t\t\tOR suchnorm(coalesce(s.klasse, ''))     LIKE '%' || tokens.norm || '%'"
	}
	return `(
		   suchnorm(s.vorname)   LIKE '%' || suchnorm($2::text) || '%'
		OR suchnorm(s.nachname)  LIKE '%' || suchnorm($2::text) || '%'
		OR lower(s.barcode_id)   LIKE '%' || lower($2::text) || '%'` + klasseImAnker(mitKlasse) + `
	  )
	  AND (
		SELECT bool_and(
			   suchnorm(coalesce(s.vorname, ''))    LIKE '%' || tokens.norm || '%'
			OR suchnorm(coalesce(s.nachname, ''))   LIKE '%' || tokens.norm || '%'
			OR lower(coalesce(s.barcode_id, ''))    LIKE '%' || tokens.roh || '%'` + klasse + `)
		FROM tokens
	  )`
}

// klasseImAnker erweitert die Ankerbedingung um die Klasse. Ohne das könnte die Suche
// nach "10a" gar nichts finden: Der Anker filtert VOR der Nachprüfung, und ein
// Klassenname steht in keinem der drei Namensfelder.
func klasseImAnker(mitKlasse bool) string {
	if !mitKlasse {
		return ""
	}
	return "\n\t\tOR suchnorm(s.klasse)    LIKE '%' || suchnorm($2::text) || '%'"
}

// SchuelerSuchRang sortiert nach Trefferqualität: ein Token am Wortanfang des Nachnamens
// wiegt schwerer als eines im Vornamen, beides zusammen schlägt alles. Sonst stünde bei
// "max hoffmann" der gesuchte Schüler irgendwo alphabetisch in der Liste.
const SchuelerSuchRang = `
		  (SELECT count(*) FROM tokens WHERE suchnorm(coalesce(s.nachname, '')) LIKE tokens.norm || '%') * 2
		+ (SELECT count(*) FROM tokens WHERE suchnorm(coalesce(s.vorname, ''))  LIKE tokens.norm || '%') DESC`

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
// Die SQL-Bausteine stehen oben als Konstanten — dieselbe Suche bedient die Schülerdatei.
func (r *pgStudentRepository) SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, int, error) {
	tokens := suchTokens(queryText)
	if len(tokens) == 0 {
		return nil, 0, nil
	}

	query := SchuelerSuchCTE + `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, ''),
		       count(*) OVER () AS gesamt
		FROM schueler s
		WHERE s.deleted_at IS NULL
		  AND ` + SchuelerSuchBedingung(false) + `
		ORDER BY ` + SchuelerSuchRang + `,
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
