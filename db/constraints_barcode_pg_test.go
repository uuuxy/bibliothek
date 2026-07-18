package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestBarcodePartialUnique sichert die Invariante aus Migration 049 ab: barcode_id ist NUR
// unter aktiven Schülern eindeutig. Ein soft-gelöschter Ausweis-Barcode darf bei
// Wiederanmeldung/Recycling neu vergeben werden — sonst wäre der Barcode dauerhaft
// "verbrannt" und ein Neuzugang mit recyceltem Ausweis crashte.
func TestBarcodePartialUnique(t *testing.T) {
	pool := pgTestPool(t)

	const insAktiv = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
	                  VALUES ($1, $2, 'Test', '7a', 2030)`
	const insGeloescht = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
	                      VALUES ($1, $2, 'Test', '7a', 2030, now())`

	t.Run("zwei aktive mit gleichem Barcode werden abgelehnt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "erster aktiver Schüler", insAktiv, "BC-100", "Anna")
			erwarteConstraintVerletzung(t, tx, "uniq_schueler_barcode_active",
				insAktiv, "BC-100", "Bela")
		})
	})

	t.Run("soft-geloeschter Barcode blockiert die Neuvergabe nicht", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "gelöschter Vorgänger", insGeloescht, "BC-200", "Cara")
			erwarteErfolg(t, tx, "aktiver Neuzugang mit recyceltem Ausweis", insAktiv, "BC-200", "Dana")
		})
	})
}
