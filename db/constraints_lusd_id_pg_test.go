package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestLusdIdPartialUnique sichert die Invariante aus Migration 035 ab: lusd_id ist NUR
// unter aktiven Schülern eindeutig (partieller Index uniq_schueler_lusd_id_active,
// WHERE deleted_at IS NULL AND lusd_id IS NOT NULL). Ohne diese Beschränkung blockierte
// eine soft-gelöschte lusd_id die Wiederanmeldung desselben Schülers beim nächsten
// LUSD-Import — er blieb dauerhaft unsichtbar. Die Migration selbst wird nur einmal
// angewandt; getestet wird der von schema.sql ausgelieferte Endzustand (den auch der
// Migrationslauf herstellen muss).
func TestLusdIdPartialUnique(t *testing.T) {
	pool := pgTestPool(t)

	const insAktiv = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
	                  VALUES ($1, $2, 'Test', '7a', 2030, $3)`
	const insGeloescht = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, deleted_at)
	                      VALUES ($1, $2, 'Test', '7a', 2030, $3, now())`

	t.Run("zwei aktive mit gleicher lusd_id werden abgelehnt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "erster aktiver Schüler", insAktiv, "S-A", "Anna", "L-100")
			erwarteConstraintVerletzung(t, tx, "uniq_schueler_lusd_id_active",
				insAktiv, "S-B", "Bela", "L-100")
		})
	})

	t.Run("soft-geloeschte lusd_id blockiert die Neuanlage nicht", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "gelöschter Vorgänger", insGeloescht, "S-DEL", "Cara", "L-200")
			erwarteErfolg(t, tx, "aktiver Rückkehrer mit gleicher lusd_id", insAktiv, "S-NEW", "Cara", "L-200")
		})
	})

	t.Run("mehrere soft-geloeschte mit gleicher lusd_id sind erlaubt (Historie)", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "erste gelöschte Zeile", insGeloescht, "S-D1", "Dana", "L-300")
			erwarteErfolg(t, tx, "zweite gelöschte Zeile", insGeloescht, "S-D2", "Dana", "L-300")
		})
	})

	t.Run("NULL lusd_id ist vom Unique-Index ausgenommen", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "erste NULL-lusd_id",
				`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				 VALUES ('S-N1', 'Emil', 'Test', '7a', 2030)`)
			erwarteErfolg(t, tx, "zweite NULL-lusd_id",
				`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				 VALUES ('S-N2', 'Fritz', 'Test', '7a', 2030)`)
		})
	})
}
