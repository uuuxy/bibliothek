package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Migration 119: benutzer.personenart ist leer, „lehrkraft" oder „liv" (Lehrkraft im
// Vorbereitungsdienst). Die Personenart sagt, wer jemand ist; die Rolle, was er darf.
func TestBenutzerPersonenartWertebereich(t *testing.T) {
	pool := pgTestPool(t)
	const ins = `INSERT INTO benutzer (vorname, nachname, email, rolle, personenart)
	             VALUES ('Per', 'Sonenart', $1, 'kollegium', $2)`

	inTx(t, pool, func(tx pgx.Tx) {
		erwarteErfolg(t, tx, "Lehrkraft", ins, "pa1@test.invalid", "lehrkraft")
		erwarteErfolg(t, tx, "LiV", ins, "pa2@test.invalid", "liv")
		erwarteErfolg(t, tx, "ohne Angabe", ins, "pa3@test.invalid", nil)
		erwarteConstraintVerletzung(t, tx, "chk_benutzer_personenart", ins, "pa4@test.invalid", "schueler")
	})
}

// Die Migration trägt bei jedem vorhandenen Kollegiumskonto „lehrkraft" ein — die anderen Rollen
// (Admin, Mitarbeiter, Helfer) bleiben leer, sie sind nicht zwingend Lehrkräfte.
func TestBenutzerPersonenartMigrationTraegtKollegiumNach(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	sql, err := os.ReadFile("../migrations/119_benutzer_personenart.sql")
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}

	inTx(t, pool, func(tx pgx.Tx) {
		erwarteErfolg(t, tx, "Kollegium", `INSERT INTO benutzer (vorname, nachname, email, rolle)
			VALUES ('Kol', 'Legium', 'pam1@test.invalid', 'kollegium')`)
		erwarteErfolg(t, tx, "Mitarbeiter", `INSERT INTO benutzer (vorname, nachname, email, rolle)
			VALUES ('Mit', 'Arbeit', 'pam2@test.invalid', 'mitarbeiter')`)
		erwarteErfolg(t, tx, "Kollegium schon als LiV",
			`INSERT INTO benutzer (vorname, nachname, email, rolle, personenart)
			VALUES ('Li', 'V', 'pam3@test.invalid', 'kollegium', 'liv')`)
		erwarteErfolg(t, tx, "Kollegium ohne Angabe", `UPDATE benutzer SET personenart = NULL
			WHERE email IN ('pam1@test.invalid', 'pam2@test.invalid')`)

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("Migration ausführen: %v", err)
		}
		for email, erwartet := range map[string]string{
			"pam1@test.invalid": "lehrkraft",
			"pam2@test.invalid": "",
			"pam3@test.invalid": "liv",
		} {
			var art *string
			if err := tx.QueryRow(ctx, `SELECT personenart FROM benutzer WHERE email = $1`, email).Scan(&art); err != nil {
				t.Fatalf("%s lesen: %v", email, err)
			}
			ist := ""
			if art != nil {
				ist = *art
			}
			if ist != erwartet {
				t.Errorf("%s: Personenart %q, erwartet %q", email, ist, erwartet)
			}
		}
	})
}
