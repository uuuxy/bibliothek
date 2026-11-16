package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// barcode_id ist nur unter aktiven Lesern eindeutig (uniq_schueler_barcode_active): Die Nummer
// eines gelöschten Ausweises darf bei einer Wiederanmeldung oder mit einem wiederverwendeten
// Ausweis neu vergeben werden, sonst wäre sie dauerhaft verbraucht und der Neuzugang scheiterte.
func TestBarcodeEindeutigUnterAktiven(t *testing.T) {
	pool := pgTestPool(t)

	const insAktiv = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
	                  VALUES ($1, $2, 'Test', '7a', 2030)`
	const insGeloescht = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
	                      VALUES ($1, $2, 'Test', '7a', 2030, now())`

	t.Run("zwei aktive mit gleichem Barcode werden abgelehnt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "erster aktiver Schüler", insAktiv, "BC-100", "Anna")
			erwarteVerletzteBedingung(t, tx, "uniq_schueler_barcode_active",
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
