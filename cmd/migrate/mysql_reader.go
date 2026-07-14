package main

import (
	"database/sql"
	"fmt"
)

// mysqlMedium represents one row from the old MySQL `medien` table.
type mysqlMedium struct {
	ID               int
	Titel            string
	Untertitel       sql.NullString
	Autor            sql.NullString
	ISBN             sql.NullString
	Verlag           sql.NullString
	Erscheinungsjahr sql.NullInt64
	Beschreibung     sql.NullString
	Medientyp        sql.NullString
	Standort         sql.NullString // free-text shelf location → JSONB
	Regal            sql.NullString // rack/row label           → JSONB
	Notizen          sql.NullString // free-text notes          → JSONB
	Anzahl           int            // physical copy count
	ErstelltAm       sql.NullTime
}

func readMySQLTitles(db *sql.DB) ([]mysqlMedium, error) {
	// Adjust the column list / table name here if your old schema differs.
	const q = `
		SELECT
			id,
			titel,
			IFNULL(untertitel, ''),
			IFNULL(autor, ''),
			IFNULL(isbn, ''),
			IFNULL(verlag, ''),
			IFNULL(erscheinungsjahr, 0),
			IFNULL(beschreibung, ''),
			IFNULL(medientyp, 'Buch'),
			IFNULL(standort, ''),
			IFNULL(regal, ''),
			IFNULL(notizen, ''),
			IFNULL(anzahl, 1),
			erstellt_am
		FROM medien
		ORDER BY id
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("mysql query: %w", err)
	}
	defer func() { _ = rows.Close() }() //nolint:errcheck

	var results []mysqlMedium
	for rows.Next() {
		m, err := scanMySQLMedium(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// scanMySQLMedium liest eine Zeile in ein mysqlMedium und wandelt leere Freitextfelder
// in typisierte NULL-Werte (sql.Null*) um. Ein Anzahl ≤ 0 wird auf 1 normalisiert.
func scanMySQLMedium(rows *sql.Rows) (mysqlMedium, error) {
	var m mysqlMedium
	var (
		untertitel       string
		autor            string
		isbn             string
		verlag           string
		erscheinungsjahr int64
		beschreibung     string
		medientyp        string
		standort         string
		regal            string
		notizen          string
	)
	if err := rows.Scan(
		&m.ID, &m.Titel,
		&untertitel, &autor, &isbn, &verlag,
		&erscheinungsjahr, &beschreibung, &medientyp,
		&standort, &regal, &notizen,
		&m.Anzahl, &m.ErstelltAm,
	); err != nil {
		return m, fmt.Errorf("mysql scan row id=%d: %w", m.ID, err)
	}
	if untertitel != "" {
		m.Untertitel = sql.NullString{String: untertitel, Valid: true}
	}
	if autor != "" {
		m.Autor = sql.NullString{String: autor, Valid: true}
	}
	if isbn != "" {
		m.ISBN = sql.NullString{String: isbn, Valid: true}
	}
	if verlag != "" {
		m.Verlag = sql.NullString{String: verlag, Valid: true}
	}
	if erscheinungsjahr > 0 {
		m.Erscheinungsjahr = sql.NullInt64{Int64: erscheinungsjahr, Valid: true}
	}
	if beschreibung != "" {
		m.Beschreibung = sql.NullString{String: beschreibung, Valid: true}
	}
	if medientyp != "" {
		m.Medientyp = sql.NullString{String: medientyp, Valid: true}
	}
	if standort != "" {
		m.Standort = sql.NullString{String: standort, Valid: true}
	}
	if regal != "" {
		m.Regal = sql.NullString{String: regal, Valid: true}
	}
	if notizen != "" {
		m.Notizen = sql.NullString{String: notizen, Valid: true}
	}
	if m.Anzahl <= 0 {
		m.Anzahl = 1
	}
	return m, nil
}
