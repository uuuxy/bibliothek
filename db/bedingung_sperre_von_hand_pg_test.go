package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// Auch eine Sperre von Hand (is_manually_blocked) verlangt einen Grund (block_reason), nicht nur
// die Sperre des Systems (ist_gesperrt); sonst stünde eine Sperre da, die niemand erklären kann.
func TestSperreVonHandVerlangtGrund(t *testing.T) {
	pool := pgTestPool(t)

	base := `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr`

	t.Run("manuelle Sperre ohne Grund abgelehnt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteVerletzteBedingung(t, tx, "chk_schueler_block_reason",
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
			erwarteVerletzteBedingung(t, tx, "chk_schueler_block_reason",
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
