//go:build raster

package repository

// Nachstellung Rasterdurchgang 23.09.2026 abends — die Formwechsel seit dem Durchgang vom
// 22.09.2026 (Migrationen 134 bis 142 und die Türen dazwischen). Build-Tag raster: läuft nur
// mit -tags raster. Rot heißt „bestätigt".

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Verdacht K5 (Fragen 6 und 12, Migration 137): trg_leser_stempel_rueckgabe sperrt bei jeder
// Rückgabe die Leserzeile des Ausleihers — NACH der Ausleihe. Die Fremdrückgabe an der Theke
// (HandleUnifiedCheckout → handleForeignReturn) sperrt vorher das Kind der offenen Sitzung
// (zaehleAktiveSchuelerAusleihen). Geben zwei Kinder an zwei Theken zugleich je das Buch des
// anderen ab, sperrt jede Transaktion ihr eigenes Kind und wartet im Trigger auf das andere.
// Die Schritte hier sind die des Codes: dieselbe Sperre auf den Schüler,
// GetActiveLoanByCopyIDTx, ReturnLoanZumTx. Die Gegenprobe läuft dieselben Schritte mit
// abgeschalteten Triggern (session_replication_role = replica) und kommt durch. Der Test
// bleibt rot bis zur Entscheidung in OFFEN.md 5.22.
func TestRaster_FremdrueckgabeUeberKreuzVerklemmtSich(t *testing.T) {
	pool := pgTestPool(t)
	for _, fall := range []struct {
		name        string
		ohneTrigger bool
	}{
		{"Gegenprobe ohne Trigger", true},
		{"mit dem Trigger aus Migration 137", false},
	} {
		t.Run(fall.name, func(t *testing.T) {
			err1, err2 := kreuzRueckgabe(t, pool, fall.ohneTrigger)
			for i, err := range []error{err1, err2} {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "40P01" {
					t.Errorf("Theke %d: Verklemmung (40P01) — %v", i+1, err)
				} else if err != nil {
					t.Errorf("Theke %d: %v", i+1, err)
				}
			}
		})
	}
}

func kreuzRueckgabe(t *testing.T, pool *pgxpool.Pool, ohneTrigger bool) (err1, err2 error) {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	eins := func(sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return id
	}
	staff := eins(`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Kreuz', 'Theke', $1, 'mitarbeiter', true) RETURNING id`, "kreuz-"+suffix+"@schule.invalid")
	lina := eins(`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Lina', 'Kreuz', '07B', 2031) RETURNING id`, "S-KL-"+suffix)
	mia := eins(`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Mia', 'Kreuz', '07B', 2031) RETURNING id`, "S-KM-"+suffix)
	titel := eins(`INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ('Kreuz-Testband', 'Prüfer', 'Buch') RETURNING id`)
	exA := eins(`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, true) RETURNING id`, titel, "B-KA-"+suffix)
	exB := eins(`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, true) RETURNING id`, titel, "B-KB-"+suffix)
	loanA := eins(`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist) VALUES ($1, $2, now() + interval '14 days') RETURNING id`, exA, mia)
	loanB := eins(`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist) VALUES ($1, $2, now() + interval '14 days') RETURNING id`, exB, lina)
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE id IN ($1, $2)`,
		} {
			if _, err := pool.Exec(ctx, sql, loanA, loanB); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE id IN ($1, $2)`, exA, exB); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id = $1`, titel); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id IN ($1, $2)`, lina, mia); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE id = $1`, staff); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	loanRepo := NewLoanRepository(pool)
	t1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = t1.Rollback(ctx) }()
	t2, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = t2.Rollback(ctx) }()
	if ohneTrigger {
		for _, tx := range []interface {
			Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
		}{t1, t2} {
			if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = replica`); err != nil {
				t.Fatalf("Trigger abschalten: %v", err)
			}
		}
	}
	var pid1 int
	if err := t1.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid1); err != nil {
		t.Fatal(err)
	}

	// Theke 1: Linas Sitzung ist offen, sie gibt Mias Buch A ab.
	if _, err := t1.Exec(ctx, `SELECT id FROM schueler WHERE id = $1 FOR UPDATE`, lina); err != nil {
		t.Fatal(err)
	}
	if l, err := loanRepo.GetActiveLoanByCopyIDTx(ctx, t1, exA); err != nil || l == nil {
		t.Fatalf("Theke 1: Ausleihe A: %v %v", l, err)
	}
	// Theke 2: Mias Sitzung ist offen, sie gibt Linas Buch B ab.
	if _, err := t2.Exec(ctx, `SELECT id FROM schueler WHERE id = $1 FOR UPDATE`, mia); err != nil {
		t.Fatal(err)
	}
	if l, err := loanRepo.GetActiveLoanByCopyIDTx(ctx, t2, exB); err != nil || l == nil {
		t.Fatalf("Theke 2: Ausleihe B: %v %v", l, err)
	}

	fertig1 := make(chan error, 1)
	go func() { fertig1 <- ReturnLoanZumTx(ctx, t1, loanA, staff, true, nil) }()
	// Warten, bis Theke 1 auf eine Sperre wartet — oder fertig ist (ohne Trigger wartet sie nie).
	gewartet := false
	for i := 0; i < 50 && !gewartet; i++ {
		select {
		case err1 = <-fertig1:
			gewartet = true
			fertig1 <- err1
		default:
			var wartet bool
			if err := pool.QueryRow(ctx, `SELECT cardinality(pg_blocking_pids($1)) > 0`, pid1).Scan(&wartet); err != nil {
				t.Fatal(err)
			}
			if wartet {
				gewartet = true
			} else {
				time.Sleep(20 * time.Millisecond)
			}
		}
	}
	err2 = ReturnLoanZumTx(ctx, t2, loanB, staff, true, nil)
	if err2 == nil {
		err2 = t2.Commit(ctx)
	} else {
		_ = t2.Rollback(ctx)
	}
	err1 = <-fertig1
	if err1 == nil {
		err1 = t1.Commit(ctx)
	}
	return err1, err2
}
