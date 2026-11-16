package repository

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/pkg/betrag"
)

// AenderePositionspreis setzt den Einzelpreis einer Bestellposition neu und zieht mit, was ihn
// beim Bestellen übernommen hat: den Einkaufspreis der Exemplare dieses Titels aus der Bestellung
// und die Summe der Bestellung, alles in einer Transaktion. Es liefert den Titel, den alten Preis
// und die Zahl der geänderten Exemplare; pgx.ErrNoRows, wenn die Position nicht zu dieser
// Bestellung gehört.
//
// Ein Titel steht in einer Bestellung einmal, weil der Warenkorb gleiche Titel zusammenfasst;
// titel_id trifft deshalb genau die Exemplare dieser Position. Ohne Titel (ON DELETE SET NULL)
// gibt es keine Exemplare mehr, und nur Position und Summe ändern sich.
func AenderePositionspreis(ctx context.Context, pool db.Pool, bestellungID, positionID string, preis betrag.Cent) (titel string, alt betrag.Cent, exemplare int64, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", 0, 0, err
	}
	defer db.Verwerfe(ctx, tx)

	var titelID *string
	if err := tx.QueryRow(ctx, `
		SELECT titel_name, einzelpreis, titel_id::text FROM bestellungen_positionen
		WHERE id = $1 AND bestellung_id = $2
		FOR UPDATE`, positionID, bestellungID).Scan(&titel, &alt, &titelID); err != nil {
		return "", 0, 0, err
	}
	tag, err := tx.Exec(ctx, `UPDATE bestellungen_positionen SET einzelpreis = $2 WHERE id = $1`, positionID, preis)
	if err != nil {
		return "", 0, 0, err
	}
	if tag.RowsAffected() != 1 {
		return "", 0, 0, fmt.Errorf("position %s: %d Zeilen geändert, erwartet 1", positionID, tag.RowsAffected())
	}
	// Null Exemplare ist kein Fehler: Eine Bestellung ohne Barcodes oder eine alte Bestellung
	// hat keine.
	if titelID != nil {
		tag, err := tx.Exec(ctx, `
			UPDATE buecher_exemplare SET einkaufspreis = $3
			WHERE bestellung_id = $1 AND titel_id = $2`, bestellungID, *titelID, preis)
		if err != nil {
			return "", 0, 0, err
		}
		exemplare = tag.RowsAffected()
	}
	tag, err = tx.Exec(ctx, `
		UPDATE bestellungen_verlauf SET gesamtbetrag = (
			SELECT coalesce(sum(menge * einzelpreis), 0) FROM bestellungen_positionen WHERE bestellung_id = $1)
		WHERE id = $1`, bestellungID)
	if err != nil {
		return "", 0, 0, err
	}
	if tag.RowsAffected() != 1 {
		return "", 0, 0, fmt.Errorf("bestellung %s: Summe nicht geändert (%d Zeilen)", bestellungID, tag.RowsAffected())
	}
	return titel, alt, exemplare, tx.Commit(ctx)
}
