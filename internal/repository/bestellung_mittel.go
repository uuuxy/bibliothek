package repository

import (
	"context"

	"bibliothek/internal/db"
)

// KorrigiereBestellungMittel setzt den Topf einer bestehenden Bestellung neu und liefert den
// bisherigen Wert (nil bei einer alten Bestellung ohne Zuordnung). pgx.ErrNoRows, wenn es die
// Bestellung nicht gibt.
//
// Alter und neuer Wert kommen aus einer Anweisung: Die Unterabfrage sperrt die Zeile und
// liefert den bisherigen Topf mit, ohne ein Fenster, in dem ein zweiter Bearbeiter
// dazwischenschreibt. Geändert wird nur der Topf: Kundennummer, Name und Adresse auf dem Beleg
// sind die Abschrift dessen, was der Händler bekommen hat
// (internal/api/bestellung/bestellung_mittel.go).
func KorrigiereBestellungMittel(ctx context.Context, pool db.Pool, bestellungID, mittel string) (*string, error) {
	var vorher *string
	err := pool.QueryRow(ctx, `
		UPDATE bestellungen_verlauf b
		   SET mittel = $2
		  FROM (SELECT id, mittel FROM bestellungen_verlauf WHERE id = $1 FOR UPDATE) alt
		 WHERE b.id = alt.id
		RETURNING alt.mittel`, bestellungID, mittel).Scan(&vorher)
	return vorher, err
}
