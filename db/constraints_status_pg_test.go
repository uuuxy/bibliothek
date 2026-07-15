package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestStatusConstraints prüft die geschlossenen Zustandsvokabulare aus den
// Migrationen 040 und 041. Ein Tippfehler in einem Statuswert würde sonst still
// die Programmlogik verbiegen (z. B. Cover-Retry-Auswahl).
func TestStatusConstraints(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("vormerkung-status nur wartend/abholbereit", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_vormerkung_status",
				`INSERT INTO vormerkungen (titel_id, status) VALUES ($1, 'erledigt')`, titelID)
			erwarteErfolg(t, tx, "status = abholbereit",
				`INSERT INTO vormerkungen (titel_id, status) VALUES ($1, 'abholbereit')`, titelID)
		})
	})

	t.Run("inventur-status nur NULL/ausstehend/erfasst", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_inventur_status",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, inventur_status)
				 VALUES ($1, 'INV-1', 'unbekannt')`, titelID)
			erwarteErfolg(t, tx, "inventur_status = NULL (nicht in Inventur)",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, inventur_status)
				 VALUES ($1, 'INV-2', NULL)`, titelID)
		})
	})

	t.Run("grade_level nur 0-13", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteConstraintVerletzung(t, tx, "chk_grade_level_bereich",
				`INSERT INTO buecher_titel (titel, grade_level) VALUES ('X', 14)`)
			erwarteConstraintVerletzung(t, tx, "chk_grade_level_bereich",
				`INSERT INTO buecher_titel (titel, grade_level) VALUES ('X', -1)`)
			// 13 muss durchgehen: kooperative Gesamtschule inkl. Oberstufe.
			erwarteErfolg(t, tx, "grade_level = 13 (Oberstufe)",
				`INSERT INTO buecher_titel (titel, grade_level) VALUES ('X', 13)`)
		})
	})

	t.Run("cover_status nur bekannte Werte", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			// Kleinschreibung ist ein echter Fehler: die Retry-Auswahl im CoverService
			// vergleicht exakt gegen die GROSS-Werte.
			erwarteConstraintVerletzung(t, tx, "chk_cover_status",
				`INSERT INTO buecher_titel (titel, cover_status) VALUES ('X', 'pending')`)
			erwarteErfolg(t, tx, "cover_status = NOT_FOUND",
				`INSERT INTO buecher_titel (titel, cover_status) VALUES ('X', 'NOT_FOUND')`)
		})
	})
}
