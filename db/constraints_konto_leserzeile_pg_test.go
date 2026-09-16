package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Migration 125: Jedes Konto hat eine Leserzeile — dort stehen sein Ausweis und seine
// Ausleihen. Ein Konto ohne sie fände die Theke nicht, und genau das war der Fehler, den
// der Umbau abschafft (ein Admin ohne Personenart war für die Theke keine Person).
//
// Die Zusage hängt an einem Trigger und nicht an den Schreibwegen: Konten entstehen an fünf
// Stellen (Benutzerverwaltung, Selbstanmeldung, Littera-Übernahme, Seed, Testaufbauten).
// Jede davon einzeln zu erinnern hieße, dass die sechste es vergisst — deshalb prüft dieser
// Test mit rohem SQL, also am selben Weg, den ein neuer Schreibpfad nähme.
func TestKontoBekommtImmerEineLeserzeile(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	lies := func(t *testing.T, tx pgx.Tx, email string) (leserID *string, art *string) {
		t.Helper()
		if err := tx.QueryRow(ctx, `
			SELECT b.leser_id::text, l.art
			FROM benutzer b LEFT JOIN leser l ON l.id = b.leser_id
			WHERE b.email = $1`, email).Scan(&leserID, &art); err != nil {
			t.Fatalf("%s lesen: %v", email, err)
		}
		return leserID, art
	}

	inTx(t, pool, func(tx pgx.Tx) {
		for _, rolle := range []string{"kollegium", "mitarbeiter", "helfer", "leitung", "admin"} {
			email := "lz-" + rolle + "@test.invalid"
			erwarteErfolg(t, tx, "Konto "+rolle,
				`INSERT INTO benutzer (vorname, nachname, email, rolle) VALUES ('Le', 'Ser', $1, $2::benutzer_rolle)`,
				email, rolle)

			leserID, art := lies(t, tx, email)
			if leserID == nil {
				t.Errorf("%s: Konto ohne Leserzeile — die Theke fände diese Person nicht", rolle)
				continue
			}
			// Nicht 'schueler': Diese Zeile darf der LUSD-Abgleich nie als Abgänger
			// markieren und der Löschjob nie anfassen.
			if art == nil || *art != "lehrkraft" {
				t.Errorf("%s: Art %v, erwartet lehrkraft", rolle, art)
			}
		}
	})
}

// Gegenprobe: Wer seine Leserzeile SELBST mitbringt, behält sie. Ohne diese Ausnahme legte
// der Trigger bei jedem Konto eine zweite Person an — etwa wenn die Leserdatei später eine
// vorhandene Lehrkraft mit einem Konto verbindet.
func TestKontoMitEigenerLeserzeileBehaeltSie(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	inTx(t, pool, func(tx pgx.Tx) {
		var leserID string
		if err := tx.QueryRow(ctx, `INSERT INTO leser (vorname, nachname, art)
			VALUES ('Vor', 'Handen', 'liv') RETURNING id::text`).Scan(&leserID); err != nil {
			t.Fatalf("Leserzeile anlegen: %v", err)
		}
		erwarteErfolg(t, tx, "Konto mit vorhandener Leserzeile",
			`INSERT INTO benutzer (vorname, nachname, email, rolle, leser_id)
			 VALUES ('Vor', 'Handen', 'lz-eigen@test.invalid', 'kollegium', $1::uuid)`, leserID)

		var gesetzt string
		if err := tx.QueryRow(ctx,
			`SELECT leser_id::text FROM benutzer WHERE email = 'lz-eigen@test.invalid'`).Scan(&gesetzt); err != nil {
			t.Fatalf("Konto lesen: %v", err)
		}
		if gesetzt != leserID {
			t.Errorf("der Trigger hat eine zweite Person angelegt: %s statt %s", gesetzt, leserID)
		}

		var anzahl int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM leser WHERE nachname = 'Handen'`).Scan(&anzahl); err != nil {
			t.Fatalf("zählen: %v", err)
		}
		if anzahl != 1 {
			t.Errorf("%d Leserzeilen mit diesem Namen, erwartet 1", anzahl)
		}
	})
}
