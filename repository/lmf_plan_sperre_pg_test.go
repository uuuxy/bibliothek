package repository_test

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Ein Speichern des LMF-Plans gilt nur auf dem Stand, den der Planer geladen hat
// (api/lernmittel/lmf_plan.go). Laufen zwei zugleich, muss das zweite auf das erste warten und
// dessen Stempel lesen, sonst schriebe es über den eben gespeicherten Plan. Geprüft an den zwei
// Lagen, in denen keine Zeile da ist, deren Sperre das zweite aufhielte: Es gibt noch keinen Plan
// der Art, oder das erste Speichern legt den Plan eines neuen Schuljahrs an.
func TestSperreNeuestenPlan_ZweitesSpeichernWartetAufDasErste(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	raeume := func() { pgtest.RaeumeAuf(t, pool, `DELETE FROM lmf_plaene`) }
	t.Cleanup(raeume)
	abfragen := repository.NewLMFAbfragen(pool)
	schreibe := func(tx pgx.Tx, ersterTag string) {
		t.Helper()
		plan := repository.LMFPlan{Art: repository.LMFTerminAusgabe, ErsterTag: ersterTag, Startstunde: 2, StundenJeTag: 6}
		if _, err := abfragen.SchreibePlan(ctx, tx, plan, nil, nil, nil); err != nil {
			t.Fatalf("Plan ab %s schreiben: %v", ersterTag, err)
		}
	}

	for _, fall := range []struct {
		name    string
		vorjahr bool
	}{{"ohne Plan", false}, {"neues Schuljahr", true}} {
		t.Run(fall.name, func(t *testing.T) {
			raeume()
			if fall.vorjahr {
				tx, err := pool.Begin(ctx)
				if err != nil {
					t.Fatalf("Transaktion: %v", err)
				}
				schreibe(tx, "2026-08-10")
				if err := tx.Commit(ctx); err != nil {
					t.Fatalf("Plan des Vorjahrs: %v", err)
				}
			}

			erstes, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("Transaktion: %v", err)
			}
			defer db.Verwerfe(ctx, erstes)
			geladen, err := abfragen.SperreNeuestenPlan(ctx, erstes, repository.LMFTerminAusgabe)
			if err != nil {
				t.Fatalf("erstes Speichern, Stempel lesen: %v", err)
			}
			schreibe(erstes, "2027-08-10")

			// Das zweite Speichern beginnt, solange das erste offen ist.
			type ergebnis struct {
				stempel string
				err     error
			}
			kennung := make(chan int32, 1)
			gelesen := make(chan ergebnis, 1)
			go func() {
				zweites, err := pool.Begin(ctx)
				if err != nil {
					kennung <- 0
					gelesen <- ergebnis{err: err}
					return
				}
				defer db.Verwerfe(ctx, zweites)
				var pid int32
				if err := zweites.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
					kennung <- 0
					gelesen <- ergebnis{err: err}
					return
				}
				kennung <- pid
				stempel, err := abfragen.SperreNeuestenPlan(ctx, zweites, repository.LMFTerminAusgabe)
				gelesen <- ergebnis{stempel: stempel, err: err}
			}()

			pid := <-kennung
			frist := time.Now().Add(10 * time.Second)
			for wartet := false; !wartet; {
				select {
				case e := <-gelesen:
					t.Fatalf("Das zweite Speichern wartete nicht auf das erste: Es las %q (Fehler %v), geladen war %q", e.stempel, e.err, geladen)
				default:
				}
				if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_locks WHERE pid = $1 AND NOT granted)`, pid).Scan(&wartet); err != nil {
					t.Fatalf("Sperren lesen: %v", err)
				}
				if !wartet && time.Now().After(frist) {
					t.Fatal("Das zweite Speichern kam in 10 s weder an eine Sperre noch zu einem Stempel")
				}
				time.Sleep(10 * time.Millisecond)
			}

			if err := erstes.Commit(ctx); err != nil {
				t.Fatalf("erstes Speichern: %v", err)
			}
			e := <-gelesen
			if e.err != nil {
				t.Fatalf("zweites Speichern, Stempel lesen: %v", e.err)
			}
			neu, err := abfragen.LadeNeuestenPlan(ctx, repository.LMFTerminAusgabe)
			if err != nil {
				t.Fatalf("gespeicherten Plan lesen: %v", err)
			}
			if e.stempel != neu.Plan.AktualisiertAm {
				t.Errorf("Das zweite Speichern las %q, das erste hat %q gespeichert: Es schriebe über den eben gespeicherten Plan", e.stempel, neu.Plan.AktualisiertAm)
			}
		})
	}
}
