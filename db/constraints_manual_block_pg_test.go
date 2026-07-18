package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestManualBlockRequiresReason sichert Migration 051 ab: Auch eine manuelle Sperre
// (is_manually_blocked) verlangt einen block_reason — nicht nur die Systemsperre
// (ist_gesperrt). Sonst entsteht die "Zombie-Sperre" ohne Kontext.
func TestManualBlockRequiresReason(t *testing.T) {
	pool := pgTestPool(t)

	base := `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr`

	t.Run("manuelle Sperre ohne Grund abgelehnt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteConstraintVerletzung(t, tx, "chk_schueler_block_reason",
				base+`, is_manually_blocked) VALUES ('MB-1', 'A', 'B', '7a', 2030, true)`)
		})
	})

	t.Run("manuelle Sperre mit Grund erlaubt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "manuell + Grund",
				base+`, is_manually_blocked, block_reason) VALUES ('MB-2', 'A', 'B', '7a', 2030, true, 'Vandalismus')`)
		})
	})

	t.Run("Systemsperre ohne Grund weiterhin abgelehnt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteConstraintVerletzung(t, tx, "chk_schueler_block_reason",
				base+`, ist_gesperrt) VALUES ('MB-3', 'A', 'B', '7a', 2030, true)`)
		})
	})

	t.Run("ungesperrt ohne Grund erlaubt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "kein Block",
				base+`) VALUES ('MB-4', 'A', 'B', '7a', 2030)`)
		})
	})
}
