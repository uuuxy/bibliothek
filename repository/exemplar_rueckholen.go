package repository

import (
	"context"
	"fmt"
	"time"
)

// HoleExemplarZurueck bringt ein ausgesondertes oder gesperrtes Exemplar zurück in den
// Umlauf und beendet die Forderung, die seine Abwesenheit abgerechnet hat — in der
// Transaktion des AUFRUFERS.
//
// Bis zum 15.09.2026 lag dieser Rumpf im Omnibox-Service mit eigener Transaktion. Stufe 2
// des Offline-Baus braucht ihn als Baustein (OFFEN.md 2.2, Commit 8): Das Nachbuchen holt
// ein verloren gemeldetes Buch zurück und bucht im selben Schritt die Ausleihe — beides
// muss zusammen gelingen oder zusammen scheitern. Getrennt wäre eines von beiden möglich:
// das Buch zurück im Regal und die Forderung weiter offen (das Kind bliebe gesperrt), oder
// die Forderung storniert, ohne dass das Buch wieder ausleihbar ist. Der Online-Scan
// (omnibox_service.holeExemplarZurueck) und der Fund im Fehlbestandsbericht
// (MarkiereVerlustAlsGefunden) legen ihre Transaktion selbst darum.
//
// 0 Zeilen heißt: Das Exemplar ist zwischen Lookup und Update verschwunden — dann darf
// niemand „reaktiviert" melden (Phantom-Erfolg-Sweep 31.08.2026).
// bewegtAm ist der Zeitpunkt der Rückkehr (nil = jetzt); das Nachbuchen gibt den Scan-Zeitpunkt mit.
func HoleExemplarZurueck(ctx context.Context, q DBQueryer, exemplarID, bearbeiterID string, bewegtAm *time.Time) (RueckkehrBefund, error) {
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
	return VerbucheRueckkehr(ctx, q, exemplarID, bearbeiterID)
}
