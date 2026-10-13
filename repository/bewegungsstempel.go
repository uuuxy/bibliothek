package repository

import (
	"context"
	"fmt"
	"time"
)

// sqlStempelVor ist der neue Wert von letzte_bewegung_am für einen Schreiber mit übergebenem
// Zeitpunkt (nil = jetzt): der spätere von altem Stempel und Zeitpunkt. GREATEST übergeht NULL,
// ein Exemplar ohne Stempel bekommt also den Zeitpunkt.
func sqlStempelVor(param string) string {
	return `GREATEST(letzte_bewegung_am, COALESCE(` + param + `::timestamptz, CURRENT_TIMESTAMP))`
}

// sqlStempelJetzt ist dasselbe für Schreiber ohne Zeitpunkt. CURRENT_TIMESTAMP ist der Beginn
// der Transaktion — eine Transaktion, die vor der letzten Bewegung begann, setzte den Stempel
// sonst zurück (TestBewegungsstempel_LangeTransaktionSetztIhnNichtZurueck).
const sqlStempelJetzt = `GREATEST(letzte_bewegung_am, CURRENT_TIMESTAMP)`

// stempleBewegungZum setzt buecher_exemplare.letzte_bewegung_am (Migration 116) auf den
// Zeitpunkt; nil heißt jetzt (Online-Scan), das Nachbuchen gibt den Scan-Zeitpunkt mit.
// Der Wächter des Nachbuchens weist Scans ab, die älter sind als die letzte Bewegung.
// Aufgerufen NACH der Zeile in ausleihen — Sperrreihenfolge Schüler → Ausleihe → Exemplar
// (docs/invarianten.md, Abschnitt 1); das Nachbuchen hält dieselbe Reihenfolge.
//
// Der Stempel läuft nie rückwärts (sqlStempelVor): Ein früherer Zeitpunkt ließe jeden Scan
// zwischen ihm und der echten letzten Bewegung wieder durch den Wächter.
func stempleBewegungZum(ctx context.Context, q Datenbank, exemplarID string, zeitpunkt *time.Time) error {
	tag, err := q.Exec(ctx, `UPDATE buecher_exemplare SET letzte_bewegung_am = `+sqlStempelVor(`$2`)+` WHERE id = $1`, exemplarID, zeitpunkt)
	if err != nil {
		return fmt.Errorf("bewegung stempeln: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}
	return nil
}
