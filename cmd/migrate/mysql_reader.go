package main

import (
	"database/sql"
	"fmt"
)

// mysqlMedium ist eine Zeile der Tabelle `medien` der alten Datenbank in MySQL.
type mysqlMedium struct {
	ID               int
	Titel            string
	Untertitel       sql.NullString
	Autor            sql.NullString
	ISBN             sql.NullString
	Verlag           sql.NullString
	Erscheinungsjahr sql.NullInt64
	Medientyp        sql.NullString
	Standort         sql.NullString // Standort als Freitext → JSONB
	Regal            sql.NullString // Regal oder Reihe         → JSONB
	Notizen          sql.NullString // Notizen als Freitext     → JSONB
	Anzahl           int            // Zahl der Stücke
	ErstelltAm       sql.NullTime
}

// ladeMySQLTitel liest alle Zeilen der Tabelle medien aus der Quelle. Weicht das alte Schema
// ab, sind Spaltenliste und Tabellenname hier anzupassen.
func ladeMySQLTitel(quelle *sql.DB) ([]mysqlMedium, error) {
	const abfrage = `
		SELECT
			id,
			titel,
			IFNULL(untertitel, ''),
			IFNULL(autor, ''),
			IFNULL(isbn, ''),
			IFNULL(verlag, ''),
			IFNULL(erscheinungsjahr, 0),
			IFNULL(medientyp, 'Buch'),
			IFNULL(standort, ''),
			IFNULL(regal, ''),
			IFNULL(notizen, ''),
			IFNULL(anzahl, 1),
			erstellt_am
		FROM medien
		ORDER BY id
	`
	zeilen, err := quelle.Query(abfrage)
	if err != nil {
		return nil, fmt.Errorf("mysql query: %w", err)
	}
	defer func() { _ = zeilen.Close() }() //nolint:errcheck

	var medien []mysqlMedium
	for zeilen.Next() {
		m, err := leseMySQLMedium(zeilen)
		if err != nil {
			return nil, err
		}
		medien = append(medien, m)
	}
	return medien, zeilen.Err()
}

// leseMySQLMedium liest eine Zeile in ein mysqlMedium und macht aus leeren Freitextfeldern
// typisierte NULL-Werte (sql.Null*). Eine Anzahl ≤ 0 wird zu 1.
func leseMySQLMedium(zeilen *sql.Rows) (mysqlMedium, error) {
	var m mysqlMedium
	var (
		untertitel       string
		autor            string
		isbn             string
		verlag           string
		erscheinungsjahr int64
		medientyp        string
		standort         string
		regal            string
		notizen          string
	)
	if err := zeilen.Scan(
		&m.ID, &m.Titel,
		&untertitel, &autor, &isbn, &verlag,
		&erscheinungsjahr, &medientyp,
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
