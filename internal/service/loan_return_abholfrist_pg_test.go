package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Kommt ein vorgemerktes Buch zurück, liegt es drei Tage bereit; fällt das Ende auf ein
// Wochenende, einen Feiertag oder in die Ferien, bis zum nächsten Schultag (entschieden am
// 28.09.2026). Am echten Postgres über processReturnVormerkungTx mit fester Uhr; alles in einer
// Transaktion, die am Ende zurückrollt. Das Nachrücken prüft repository
// (vormerkung_abholfrist_pg_test.go).
func TestRueckgabe_AbholfristMitFesterUhr(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	zone := schulzeit.Zone()
	for i, fall := range []struct {
		name        string
		jetzt, soll time.Time
	}{
		{"Dienstag: bis Freitagabend", time.Date(2026, time.September, 22, 10, 0, 0, 0, zone),
			time.Date(2026, time.September, 25, 23, 59, 59, 0, zone)},
		{"Mittwoch: Samstag wird Montag", time.Date(2026, time.September, 16, 10, 0, 0, 0, zone),
			time.Date(2026, time.September, 21, 23, 59, 59, 0, zone)},
		{"Freitag vor den Herbstferien: Montag danach", time.Date(2026, time.October, 2, 10, 0, 0, 0, zone),
			time.Date(2026, time.October, 19, 23, 59, 59, 0, zone)},
	} {
		t.Run(fall.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer db.SafeRollback(ctx, tx)
			suffix := fmt.Sprintf("%d-%d", time.Now().UnixNano(), i)

			var schuelerID, titelID, exemplarID string
			if err := tx.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				VALUES ($1, 'Abhol', 'Frist', '07A', 2031) RETURNING id`, "ABF-"+suffix).Scan(&schuelerID); err != nil {
				t.Fatalf("Schüler anlegen: %v", err)
			}
			if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp)
				VALUES ('Abholfrist-Testband', 'Prüfer', 'Buch') RETURNING id`).Scan(&titelID); err != nil {
				t.Fatalf("Titel anlegen: %v", err)
			}
			if err := tx.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
				titelID, "B-ABF-"+suffix).Scan(&exemplarID); err != nil {
				t.Fatalf("Exemplar anlegen: %v", err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')`,
				titelID, schuelerID); err != nil {
				t.Fatalf("Vormerkung anlegen: %v", err)
			}

			svc := &defaultLoanService{pool: pool, jetzt: func() time.Time { return fall.jetzt }}
			resp := &LoanResult{}
			buch := &repository.BookCopy{ID: exemplarID, TitelID: titelID, Titel: "Abholfrist-Testband"}
			if err := svc.processReturnVormerkungTx(ctx, tx, buch, resp, nil); err != nil {
				t.Fatalf("Rückgabe: %v", err)
			}
			if !resp.HasVormerkung {
				t.Fatal("die Vormerkung wurde nicht bedient")
			}
			var bis time.Time
			if err := tx.QueryRow(ctx, `SELECT bereitgestellt_bis FROM vormerkungen WHERE schueler_id = $1`,
				schuelerID).Scan(&bis); err != nil {
				t.Fatal(err)
			}
			if !bis.Equal(fall.soll) {
				t.Errorf("Abholfrist %s, erwartet %s", bis.In(zone), fall.soll)
			}
		})
	}
}
