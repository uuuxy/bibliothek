package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"bibliothek/internal/pgtest"
)

// Migration 181 löscht die Anschriften im Kollegium und setzt chk_leser_anschrift_nur_schueler.
// Geprüft wird die echte Migrationsdatei, zweimal hintereinander, an einer Tabelle im Stand davor
// (ohne den Check) und in einer Transaktion, die am Ende zurückgerollt wird. Die Anschrift eines
// Schülers bleibt stehen.
func TestAnschriftNurSchueler_Migration181(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()
	if _, err := tx.Exec(ctx, `ALTER TABLE leser DROP CONSTRAINT chk_leser_anschrift_nur_schueler`); err != nil {
		t.Fatalf("Stand vor der Migration herstellen: %v", err)
	}

	lege := func(sql string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, sql).Scan(&id); err != nil {
			t.Fatalf("Leser anlegen: %v", err)
		}
		return id
	}
	lehrkraft := lege(`INSERT INTO leser (barcode_id, vorname, nachname, art, strasse, hausnummer, plz, ort, eltern_email)
		VALUES ('M181-LK', 'Lena', 'Lehrkraft', 'lehrkraft', 'Schulweg', '1', '61348', 'Bad Homburg', 'eltern@example.org')
		RETURNING id::text`)
	liv := lege(`INSERT INTO leser (barcode_id, vorname, nachname, art, plz)
		VALUES ('M181-LIV', 'Lars', 'Liv', 'liv', '61348') RETURNING id::text`)
	schueler := lege(`INSERT INTO leser (barcode_id, vorname, nachname, art, klasse, abgaenger_jahr,
		                                 strasse, hausnummer, plz, ort, eltern_email)
		VALUES ('M181-SCH', 'Sina', 'Schueler', 'schueler', '07A', 2030,
		        'Schulweg', '2', '61348', 'Bad Homburg', 'eltern@example.org') RETURNING id::text`)

	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "181_anschrift_nur_schueler.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	anschrift := func(id string) string {
		t.Helper()
		var s string
		if err := tx.QueryRow(ctx,
			`SELECT concat_ws('|', strasse, hausnummer, plz, ort, eltern_email) FROM leser WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatalf("Leser lesen: %v", err)
		}
		return s
	}
	for name, id := range map[string]string{"Lehrkraft": lehrkraft, "LiV": liv} {
		if s := anschrift(id); s != "" {
			t.Errorf("%s: Anschrift %q steht noch, erwartet leer", name, s)
		}
	}
	if s, want := anschrift(schueler), "Schulweg|2|61348|Bad Homburg|eltern@example.org"; s != want {
		t.Errorf("Schüler: Anschrift %q, erwartet %q unverändert", s, want)
	}

	// Danach hält die Datenbank die Regel selbst, auch gegen ein Reparaturskript.
	_, err = tx.Exec(ctx, `UPDATE leser SET ort = 'Oberursel' WHERE id = $1`, liv)
	var pgFehler *pgconn.PgError
	if !errors.As(err, &pgFehler) || pgFehler.ConstraintName != "chk_leser_anschrift_nur_schueler" {
		t.Errorf("Ort an einer LiV = %v, erwartet Verstoß gegen chk_leser_anschrift_nur_schueler", err)
	}
}
