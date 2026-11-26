package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/db"
)

// Akte und Benutzerverwaltung schreiben Leserzeile und Konto derselben Person in einer
// Transaktion: AendereLeser erst leser, dann benutzer. Sperren Ändern und Löschen eines Kontos
// andersherum, verklemmen sich zwei Plätze, die die Person zugleich bearbeiten, und Postgres
// bricht einen von beiden mit 40P01 ab.
//
// Die Probe hält die Leserzeile wie AendereLeser nach seiner ersten Anweisung, lässt daneben
// das Konto ändern oder löschen und schreibt dann das Konto, wie AendereLeser weitermacht.
// Beide müssen durchgehen.
func TestKonto_SperrtDieLeserzeileVorDemKonto(t *testing.T) {
	wege := []struct {
		name string
		weg  func(ctx context.Context, repo *KontoAbfragen, konto, bearbeiter string) error
	}{
		{"ändern", func(ctx context.Context, repo *KontoAbfragen, konto, _ string) error {
			neu := "Verheiratet"
			_, err := repo.Aendere(ctx, KontoAenderung{ID: konto, Nachname: &neu})
			return err
		}},
		{"löschen", func(ctx context.Context, repo *KontoAbfragen, konto, bearbeiter string) error {
			_, err := repo.LoescheKonto(ctx, konto, bearbeiter)
			return err
		}},
	}
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	bearbeiter := legeBearbeiterAn(t, pool)
	for i, w := range wege {
		t.Run(w.name, func(t *testing.T) {
			ctx := context.Background()
			konto, leser := legeZugangsanfrageAn(t, pool, fmt.Sprintf("sperrfolge.%d@schule.example", i))

			akte, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Verwerfe(ctx, akte)
			if _, err := akte.Exec(ctx, `UPDATE leser SET vorname = vorname WHERE id = $1`, leser); err != nil {
				t.Fatal(err)
			}

			ergebnis := make(chan error, 1)
			go func() { ergebnis <- w.weg(ctx, NewKontoAbfragen(pool), konto, bearbeiter) }()
			ueberschneidung(t, pool, akte, ergebnis)
			if len(ergebnis) > 0 {
				t.Fatalf("Konto %s wartet nicht auf die Leserzeile: Die Probe stellt die Lage nicht her (Fehler: %v)", w.name, <-ergebnis)
			}

			if _, err := akte.Exec(ctx, `SET LOCAL lock_timeout = '5s'`); err != nil {
				t.Fatal(err)
			}
			if _, err := akte.Exec(ctx, `UPDATE benutzer SET vorname = vorname WHERE id = $1`, konto); err != nil {
				t.Errorf("die Akte schreibt das Konto nicht: %v", err)
			}
			if err := akte.Commit(ctx); err != nil {
				t.Errorf("Commit der Akte: %v", err)
			}
			select {
			case err := <-ergebnis:
				if err != nil {
					t.Errorf("Konto %s neben der Akte: %v", w.name, err)
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("Konto %s hängt", w.name)
			}
		})
	}
}
