package repository

import (
	"context"
	"fmt"
	"time"
)

// HoleExemplarZurueck bringt ein ausgesondertes oder gesperrtes Exemplar zurück in den Umlauf
// und beendet die Forderung, die seine Abwesenheit abgerechnet hat, in der Transaktion des
// Rufers.
//
// Die Transaktion gehört dem Rufer, weil das Nachbuchen ein verloren gemeldetes Buch zurückholt
// und im selben Schritt die Ausleihe bucht: Beides muss zusammen gelingen oder zusammen
// scheitern. Getrennt wäre eines von beiden möglich: das Buch zurück im Regal und die Forderung
// weiter offen (das Kind bliebe gesperrt), oder die Forderung storniert, ohne dass das Buch
// wieder ausleihbar ist. Der Online-Scan (OmniboxDienst.holeExemplarZurueck) und der Fund im
// Fehlbestandsbericht (MarkiereVerlustAlsGefunden) legen ihre Transaktion selbst darum.
//
// 0 Zeilen heißt: Das Exemplar ist zwischen Nachschlagen und Ändern verschwunden; dann darf
// niemand „reaktiviert" melden. bewegtAm ist der Zeitpunkt der Rückkehr (nil heißt jetzt); das
// Nachbuchen gibt den Zeitpunkt des Scans mit.
func HoleExemplarZurueck(ctx context.Context, q Datenbank, exemplarID, bearbeiterID string, bewegtAm *time.Time) (RueckkehrBefund, error) {
	tag, err := q.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausleihbar = true, ist_ausgesondert = false, aussonderung_grund = NULL,
		    zustand_notiz = '', bestellstatus = NULL, aktualisiert_am = CURRENT_TIMESTAMP,
		    letzte_bewegung_am = `+sqlStempelVor(`$2`)+`
		WHERE id = $1`, exemplarID, bewegtAm)
	if err != nil {
		return RueckkehrBefund{}, fmt.Errorf("exemplar zurückholen: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return RueckkehrBefund{}, ErrExemplarNichtGefunden
	}
	return verbucheRueckkehr(ctx, q, exemplarID, bearbeiterID)
}
