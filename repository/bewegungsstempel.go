package repository

import (
	"context"
	"fmt"
	"time"
)

// sqlStempelVor ist der neue Wert von letzte_bewegung_am für einen Schreiber mit übergebenem
// Zeitpunkt (nil = jetzt): der spätere von altem Stempel und Zeitpunkt. GREATEST übergeht NULL,
// ein Exemplar ohne Stempel bekommt also den Zeitpunkt.
func sqlStempelVor(platzhalter string) string {
	return `GREATEST(letzte_bewegung_am, COALESCE(` + platzhalter + `::timestamptz, CURRENT_TIMESTAMP))`
}

// sqlStempelJetzt ist dasselbe für Schreiber ohne Zeitpunkt. CURRENT_TIMESTAMP ist der Beginn
// der Transaktion; GREATEST verhindert, dass eine Transaktion, die vor der letzten Bewegung
// begann, den Stempel zurücksetzt (TestBewegungsstempel_LangeTransaktionSetztIhnNichtZurueck).
const sqlStempelJetzt = `GREATEST(letzte_bewegung_am, CURRENT_TIMESTAMP)`

// stempleBewegungZum setzt buecher_exemplare.letzte_bewegung_am auf den Zeitpunkt; nil heißt
// jetzt (Scan an der Theke), das Nachbuchen gibt den Zeitpunkt des Scans mit. Der Wächter des
// Nachbuchens weist Scans ab, die älter sind als die letzte Bewegung. Gerufen wird es nach der
// Zeile in ausleihen: Die Reihenfolge der Sperren ist Leser, Ausleihe, Exemplar
// (docs/invarianten.md, Abschnitt 1), und das Nachbuchen hält dieselbe.
//
// Der Stempel läuft nie rückwärts (sqlStempelVor): Ein früherer Zeitpunkt ließe jeden Scan
// zwischen ihm und der letzten Bewegung wieder durch den Wächter.
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
