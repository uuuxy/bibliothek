package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestWertebereichConstraints prüft die Non-Negativitäts-/Positivitäts-CHECKs aus
// Migration 039. Sie schützen vor korrupten Zähl- und Geldwerten (u. a. dem von CodeQL
// gefundenen int32-Overflow, der negative Bestände erzeugen konnte).
func TestWertebereichConstraints(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("stock darf nicht negativ sein", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteConstraintVerletzung(t, tx, "chk_stock_nonneg",
				`INSERT INTO buecher_titel (titel, stock) VALUES ('X', -1)`)
			erwarteErfolg(t, tx, "stock = 0",
				`INSERT INTO buecher_titel (titel, stock) VALUES ('X', 0)`)
		})
	})

	t.Run("meldebestand darf nicht negativ sein", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteConstraintVerletzung(t, tx, "chk_meldebestand_nonneg",
				`INSERT INTO buecher_titel (titel, meldebestand) VALUES ('X', -1)`)
		})
	})

	t.Run("einkaufspreis darf nicht negativ sein", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_einkaufspreis_nonneg",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, einkaufspreis)
				 VALUES ($1, 'NEG-1', -0.01)`, titelID)
		})
	})

	t.Run("bestellposition braucht Menge >= 1", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			bestellID := neueBestellung(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_pos_menge_positiv",
				`INSERT INTO bestellungen_positionen (bestellung_id, titel_name, menge)
				 VALUES ($1, 'X', 0)`, bestellID)
			erwarteErfolg(t, tx, "menge = 1",
				`INSERT INTO bestellungen_positionen (bestellung_id, titel_name, menge)
				 VALUES ($1, 'X', 1)`, bestellID)
		})
	})

	t.Run("bestellposition einzelpreis nicht negativ", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			bestellID := neueBestellung(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_pos_einzelpreis_nonneg",
				`INSERT INTO bestellungen_positionen (bestellung_id, titel_name, menge, einzelpreis)
				 VALUES ($1, 'X', 1, -0.01)`, bestellID)
		})
	})

	t.Run("bestellkopf gesamtbetrag nicht negativ", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteConstraintVerletzung(t, tx, "chk_verlauf_gesamtbetrag_nonneg",
				`INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, gesamtbetrag)
				 VALUES ('V', 'v@x.de', -1.00)`)
		})
	})

	t.Run("bestellkopf anzahl_exemplare nicht negativ", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteConstraintVerletzung(t, tx, "chk_verlauf_anzahl_nonneg",
				`INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, anzahl_exemplare)
				 VALUES ('V', 'v@x.de', -1)`)
			// 0 ist zulässig: eine Bestellung darf (noch) ohne Exemplare geführt werden.
			erwarteErfolg(t, tx, "anzahl_exemplare = 0",
				`INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, anzahl_exemplare)
				 VALUES ('V', 'v@x.de', 0)`)
		})
	})

	t.Run("klassensatz-reservierung braucht anzahl >= 1", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_ksr_anzahl_positiv",
				`INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl)
				 VALUES ($1, '7a', 0)`, titelID)
		})
	})
}
