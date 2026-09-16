package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Eine Ausweisnummer gehört genau einer Person — Schüler wie Kollegium. Ob jemand Schüler
// oder Lehrkraft ist, steht in den Stammdaten, nicht auf dem Ausweis; die Theke sucht bei
// jeder Nummer unter allen Lesern. Bis Migration 118 galt die Eindeutigkeit nur je Tabelle,
// und jeder Schreibweg prüfte nur seine eigene: Ein Schüler und eine Lehrkraft konnten
// dieselbe Nummer tragen, und der Scan lud still den Schüler.
//
// Seit Migration 125 stehen alle Leser in EINER Tabelle. Der Trigger von 118 ist damit
// entfallen — die Zusage ist dieselbe geblieben, sie hängt jetzt am partiellen Index
// uniq_schueler_barcode_active. Dieser Test prüft weiter die ZUSAGE, nicht das Mittel:
// Käme eine zweite Tabelle mit Ausweisnummern zurück, müsste er neu geschrieben werden.
const (
	insLeserSchueler = `INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
	                    VALUES ($1, 'Aus', 'Weis', '7a', 2030)`
	insLeserGeloescht = `INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
	                     VALUES ($1, 'Aus', 'Weis', '7a', 2030, now())`
	insLeserLehrkraft = `INSERT INTO leser (barcode_id, vorname, nachname, art)
	                     VALUES ($1, 'Leh', 'Rer', 'lehrkraft')`
	constraintAusweis = "uniq_schueler_barcode_active"
)

func TestAusweisEindeutigUeberPersonen(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("Nummer eines Schülers ist für eine Lehrkraft vergeben", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insLeserSchueler, "AW-1")
			erwarteConstraintVerletzung(t, tx, constraintAusweis, insLeserLehrkraft, "AW-1")
		})
	})

	t.Run("Nummer einer Lehrkraft ist für einen Schüler vergeben", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Lehrkraft", insLeserLehrkraft, "AW-2")
			erwarteConstraintVerletzung(t, tx, constraintAusweis, insLeserSchueler, "AW-2")
		})
	})

	t.Run("gelöschter Leser gibt die Nummer frei, Wiederherstellen stößt an", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "gelöschter Schüler", insLeserGeloescht, "AW-3")
			erwarteErfolg(t, tx, "Lehrkraft mit der freigegebenen Nummer", insLeserLehrkraft, "AW-3")
			erwarteConstraintVerletzung(t, tx, constraintAusweis,
				`UPDATE leser SET deleted_at = NULL WHERE barcode_id = $1 AND art = 'schueler'`, "AW-3")
		})
	})

	t.Run("Ändern auf die Nummer der anderen Person", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insLeserSchueler, "AW-4S")
			erwarteErfolg(t, tx, "Lehrkraft", insLeserLehrkraft, "AW-4L")
			erwarteConstraintVerletzung(t, tx, constraintAusweis,
				`UPDATE leser SET barcode_id = $1 WHERE barcode_id = 'AW-4S'`, "AW-4L")
			erwarteConstraintVerletzung(t, tx, constraintAusweis,
				`UPDATE leser SET barcode_id = $1 WHERE barcode_id = 'AW-4L'`, "AW-4S")
		})
	})

	// Ohne Ausweis dürfen beliebig viele Leser nebeneinander stehen: Ein Kollege bekommt
	// seine Nummer erst, wenn ein Ausweis gedruckt wird. Der partielle Index lässt NULL
	// mehrfach zu — stünde dort ein leerer String, wäre schon der zweite Kollege abgewiesen.
	t.Run("Gegenprobe: verschiedene Nummern und Leser ohne Ausweis", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insLeserSchueler, "AW-5S")
			erwarteErfolg(t, tx, "Lehrkraft", insLeserLehrkraft, "AW-5L")
			erwarteErfolg(t, tx, "Lehrkraft ohne Ausweis", insLeserLehrkraft, nil)
			erwarteErfolg(t, tx, "zweite Lehrkraft ohne Ausweis", insLeserLehrkraft, nil)
		})
	})

	// Eine Änderung an anderen Feldern darf nicht an der Nummer scheitern.
	t.Run("Änderung ohne Nummernwechsel geht durch", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insLeserSchueler, "AW-6")
			erwarteErfolg(t, tx, "Schüler ändern, Nummer gleich",
				`UPDATE leser SET vorname = 'Neu', barcode_id = barcode_id WHERE barcode_id = $1`, "AW-6")
		})
	})
}

// TestAusweisEindeutigUeberPersonen_Gleichzeitig: Zwei Arbeitsplätze vergeben dieselbe Nummer
// gleichzeitig, einer an einen Schüler, einer an eine Lehrkraft. Der zweite muss warten, bis
// der erste feststeht, und danach abgewiesen werden — ohne diese Serialisierung sieht keine
// der beiden Prüfungen die noch nicht festgeschriebene Zeile der anderen.
//
// Gewartet wird jetzt auf eine ZEILENSPERRE des eindeutigen Index (wait_event 'tuple'), nicht
// mehr auf die Advisory-Sperre des Triggers aus Migration 118: Postgres hält den zweiten
// INSERT selbst an, bis der erste committet oder zurückrollt.
func TestAusweisEindeutigUeberPersonen_Gleichzeitig(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE barcode_id = 'AW-GLEICH'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	tx1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("erste Transaktion: %v", err)
	}
	defer SafeRollback(ctx, tx1)
	if _, err := tx1.Exec(ctx, insLeserSchueler, "AW-GLEICH"); err != nil {
		t.Fatalf("Schüler in der ersten Transaktion: %v", err)
	}

	ergebnis := make(chan error, 1)
	go func() {
		tx2, err := pool.Begin(ctx)
		if err != nil {
			ergebnis <- err
			return
		}
		defer SafeRollback(ctx, tx2)
		if _, err := tx2.Exec(ctx, insLeserLehrkraft, "AW-GLEICH"); err != nil {
			ergebnis <- err
			return
		}
		ergebnis <- tx2.Commit(ctx)
	}()

	// Die zweite Vergabe soll warten, bis die erste feststeht. Kommt sie vorher zurück, hat sie
	// die Zeile der ersten nicht gesehen. Gewartet wird, bis Postgres sie als wartend führt —
	// eine feste Wartezeit bewiese auf einem langsamen Rechner nichts: Käme die zweite Vergabe
	// erst nach dem Festschreiben der ersten an, scheiterte sie auch ohne Sperre.
	frist := time.Now().Add(10 * time.Second)
	for wartend := 0; wartend == 0; {
		select {
		case err := <-ergebnis:
			t.Fatalf("die zweite Vergabe wartete nicht auf die erste (Fehler: %v)", err)
		default:
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'`).
			Scan(&wartend); err != nil {
			t.Fatalf("wartende Vergabe suchen: %v", err)
		}
		if wartend == 0 && time.Now().After(frist) {
			t.Fatal("die zweite Vergabe erschien nie als wartend auf die Sperre")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("erste Transaktion festschreiben: %v", err)
	}

	select {
	case err := <-ergebnis:
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.ConstraintName != constraintAusweis {
			t.Fatalf("die zweite Vergabe muss an %q scheitern, Ergebnis: %v", constraintAusweis, err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("die zweite Vergabe kam nach dem Festschreiben der ersten nicht zurück")
	}
}
