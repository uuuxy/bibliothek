package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// Die Werte der Zustandsspalten sind eine geschlossene Liste. Ein Tippfehler in einem Wert
// verböge sonst still die Programmlogik, etwa die Auswahl der Cover für einen neuen Versuch.
func TestStatusNurAusDerListe(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("vormerkung-status nur wartend/abholbereit", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := legeTitelAn(t, tx)
			erwarteVerletzteBedingung(t, tx, "chk_vormerkung_status",
				`INSERT INTO vormerkungen (titel_id, status) VALUES ($1, 'erledigt')`, titelID)
			erwarteErfolg(t, tx, "status = abholbereit",
				`INSERT INTO vormerkungen (titel_id, status) VALUES ($1, 'abholbereit')`, titelID)
		})
	})

	// Den Fortschritt einer Inventur hält ihr Lauf (repository/inventurlauf_pg_test.go), den
	// Jahrgang chk_jahrgang_spanne (repository/jahrgang_unbekannt_pg_test.go).

	t.Run("cover_status nur bekannte Werte", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			// Klein geschrieben ist ein Fehler: Die Auswahl im CoverDienst vergleicht genau mit
			// den groß geschriebenen Werten.
			erwarteVerletzteBedingung(t, tx, "chk_cover_status",
				`INSERT INTO buecher_titel (titel, cover_status) VALUES ('X', 'pending')`)
			erwarteErfolg(t, tx, "cover_status = NOT_FOUND",
				`INSERT INTO buecher_titel (titel, cover_status) VALUES ('X', 'NOT_FOUND')`)
		})
	})
}
