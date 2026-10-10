package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// wartetAufSperre ruft sperre in einer zweiten Transaktion, während halter die Zeile hält, und
// verlangt, dass der Aufruf wartet und erst nach dem Ende von halter zurückkommt.
func wartetAufSperre(t *testing.T, pool *pgxpool.Pool, halter pgx.Tx, sperre func(tx pgx.Tx) error) {
	t.Helper()
	ctx := context.Background()
	fertig := make(chan error, 1)
	go func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			fertig <- err
			return
		}
		defer db.SafeRollback(ctx, tx)
		fertig <- sperre(tx)
	}()
	ueberschneidung(t, pool, halter, fertig)
	if len(fertig) > 0 {
		t.Fatalf("die zweite Transaktion hat nicht gewartet (Fehler: %v)", <-fertig)
	}
	if err := halter.Commit(ctx); err != nil {
		t.Fatalf("erste Transaktion abschließen: %v", err)
	}
	if err := <-fertig; err != nil {
		t.Errorf("zweite Transaktion nach dem Warten: %v", err)
	}
}

// Wer die Zeile eines Lesers sperrt, lässt eine zweite Buchung für denselben Leser warten, bis
// die erste Transaktion zu Ende ist.
func TestSperreLeserzeile_ZweiteTransaktionWartet(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	leser := seedSchueler(t, pool, "ZSP-1", "Sperrzeile", "7a")

	erste := beginne(t, pool)
	defer db.SafeRollback(ctx, erste)
	if err := SperreLeserzeile(ctx, erste, leser); err != nil {
		t.Fatalf("erste Sperre: %v", err)
	}
	wartetAufSperre(t, pool, erste, func(tx pgx.Tx) error { return SperreLeserzeile(ctx, tx, leser) })
}

// Die Sperre am Exemplar liefert seinen Stand, jede Spalte in ihrem Feld, und lässt eine zweite
// Transaktion warten.
func TestSperreExemplarzeile_StandUndSperre(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	titel := titelIDVonExemplar(t, pool, seedSignaturMitExemplaren(t, pool, "Sperrstand", 1)[0])
	bewegt := time.Date(2026, 4, 2, 9, 30, 0, 0, time.UTC)
	var imRegal, abgeschrieben, gesperrt string
	for nummer, ziel := range map[string]*string{"ZSP-REGAL": &imRegal, "ZSP-ABGESCHRIEBEN": &abgeschrieben, "ZSP-GESPERRT": &gesperrt} {
		sql := `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`
		switch nummer {
		case "ZSP-ABGESCHRIEBEN":
			sql = `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund, letzte_bewegung_am)
				VALUES ($1, $2, true, true, 'VERLUST', '2026-04-02 09:30:00+00') RETURNING id`
		case "ZSP-GESPERRT":
			sql = `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, false) RETURNING id`
		}
		if err := pool.QueryRow(ctx, sql, titel, nummer).Scan(ziel); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", nummer, err)
		}
	}

	tx := beginne(t, pool)
	defer db.SafeRollback(ctx, tx)
	faelle := []struct {
		name, id                 string
		ausleihbar, ausgesondert bool
		bewegung                 *time.Time
	}{
		{"im Regal", imRegal, true, false, nil},
		{"abgeschrieben", abgeschrieben, true, true, &bewegt},
		{"gesperrt", gesperrt, false, false, nil},
	}
	for _, f := range faelle {
		ist, err := SperreExemplarzeile(ctx, tx, f.id)
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		gleicheBewegung := (ist.LetzteBewegung == nil) == (f.bewegung == nil) &&
			(f.bewegung == nil || ist.LetzteBewegung.Equal(*f.bewegung))
		if ist.Ausleihbar != f.ausleihbar || ist.Ausgesondert != f.ausgesondert || !gleicheBewegung {
			t.Errorf("%s: %+v, erwartet ausleihbar %v, ausgesondert %v, Bewegung %v", f.name, ist, f.ausleihbar, f.ausgesondert, f.bewegung)
		}
	}
	wartetAufSperre(t, pool, tx, func(zweite pgx.Tx) error {
		_, err := SperreExemplarzeile(ctx, zweite, imRegal)
		return err
	})
}
