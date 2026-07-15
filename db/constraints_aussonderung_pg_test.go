package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestAussonderungGrundConstraint prüft Migration 043. Der erste Unterfall ist der
// Grund, warum es diese Testebene überhaupt gibt: Er war in der ersten Fassung des
// Constraints NICHT abgedeckt und fiel erst gegen echtes Postgres auf.
func TestAussonderungGrundConstraint(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("ausgesondert ohne Grund ist unmoeglich (NULL-Falle)", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_aussonderung_grund",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausgesondert, aussonderung_grund)
				 VALUES ($1, 'AUS-1', true, NULL)`, titelID)
		})
	})

	t.Run("ausgesondert mit unbekanntem Grund ist unmoeglich", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_aussonderung_grund",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausgesondert, aussonderung_grund)
				 VALUES ($1, 'AUS-2', true, 'KAPUTT')`, titelID)
		})
	})

	t.Run("im Umlauf darf kein Grund gesetzt sein", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_aussonderung_grund",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausgesondert, aussonderung_grund)
				 VALUES ($1, 'AUS-3', false, 'VERLUST')`, titelID)
		})
	})

	t.Run("alle vier Gruende sind gueltig", func(t *testing.T) {
		for _, grund := range []string{"VERLUST", "BESCHAEDIGUNG", "AUSSORTIERT", "BESTANDSKORREKTUR"} {
			inTx(t, pool, func(tx pgx.Tx) {
				titelID := neuerTitel(t, tx)
				erwarteErfolg(t, tx, "Grund "+grund,
					`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausgesondert, aussonderung_grund)
					 VALUES ($1, 'OK-'||$2, true, $2)`, titelID, grund)
			})
		}
	})

	t.Run("Aussondern per UPDATE erfordert ebenfalls einen Grund", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			exemplarID := neuesExemplar(t, tx, titelID)
			// Der Pfad, den ein vergessener Schreibpfad im Code nehmen würde.
			erwarteConstraintVerletzung(t, tx, "chk_aussonderung_grund",
				`UPDATE buecher_exemplare SET ist_ausgesondert = true WHERE id = $1`, exemplarID)
			erwarteErfolg(t, tx, "Aussondern mit Grund",
				`UPDATE buecher_exemplare SET ist_ausgesondert = true, aussonderung_grund = 'VERLUST'
				 WHERE id = $1`, exemplarID)
		})
	})
}
