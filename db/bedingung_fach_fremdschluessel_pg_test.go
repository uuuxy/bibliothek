package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// buecher_titel.subject verweist auf systematik_kategorien(bezeichnung): Umbenennen zieht die
// Titel mit (CASCADE), Löschen ist gesperrt, solange Titel daran hängen, und eine zweite
// Sachgruppe, die sich nur in Groß- und Kleinschreibung unterscheidet, wird abgewiesen. Geprüft
// wird der Stand aus schema.sql, den auch die Migrationen herstellen müssen.
func TestFachVerweistAufSystematik(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	t.Run("unbekanntes fach wird abgewiesen", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteVerletzteBedingung(t, tx, "fk_titel_subject_systematik",
				`INSERT INTO buecher_titel (titel, subject) VALUES ('FK-Test', 'NieRegistriert078')`)
		})
	})

	t.Run("ohne fach (NULL) bleibt ein titel erlaubt", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Titel ohne Fach",
				`INSERT INTO buecher_titel (titel) VALUES ('FK-Test-Ohne-Fach')`)
		})
	})

	t.Run("umbenennen zieht titel mit", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Sachgruppe",
				`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('Deu078', 'Deutsch078')`)
			erwarteErfolg(t, tx, "Titel mit Fach",
				`INSERT INTO buecher_titel (titel, subject) VALUES ('FK-Test-Cascade', 'Deutsch078')`)
			erwarteErfolg(t, tx, "Umbenennung",
				`UPDATE systematik_kategorien SET bezeichnung = 'Germanistik078' WHERE bezeichnung = 'Deutsch078'`)

			var subject string
			if err := tx.QueryRow(ctx,
				`SELECT subject FROM buecher_titel WHERE titel = 'FK-Test-Cascade'`).Scan(&subject); err != nil {
				t.Fatalf("Titel nach Umbenennung lesen: %v", err)
			}
			if subject != "Germanistik078" {
				t.Fatalf("CASCADE hat den Titel nicht mitgezogen: subject = %q", subject)
			}
		})
	})

	t.Run("loeschen ist gesperrt solange titel dranhaengen", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Sachgruppe",
				`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('Mat078', 'Mathematik078')`)
			erwarteErfolg(t, tx, "Titel mit Fach",
				`INSERT INTO buecher_titel (titel, subject) VALUES ('FK-Test-Restrict', 'Mathematik078')`)
			erwarteVerletzteBedingung(t, tx, "fk_titel_subject_systematik",
				`DELETE FROM systematik_kategorien WHERE bezeichnung = 'Mathematik078'`)
		})
	})

	t.Run("bezeichnung ist case-insensitiv eindeutig", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "erste Schreibweise",
				`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('Bio078', 'Biologie078')`)
			erwarteVerletzteBedingung(t, tx, "uniq_systematik_bezeichnung_ci",
				`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('bio078x', 'biologie078')`)
		})
	})
}
