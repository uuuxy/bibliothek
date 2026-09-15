package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Migration 118: Eine Ausweisnummer gehört genau einer Person — über Schüler und Kollegium
// hinweg. Ob jemand Schüler oder Lehrkraft ist, steht in den Stammdaten, nicht auf dem Ausweis;
// die Theke sucht bei jeder Nummer unter beiden. Bis Migration 118 galt die Eindeutigkeit nur je
// Tabelle, und jeder Schreibweg prüfte nur seine eigene: Ein Schüler und eine Lehrkraft konnten
// dieselbe Nummer tragen, und der Scan lud still den Schüler.
const (
	insSchuelerAktiv = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
	                    VALUES ($1, 'Aus', 'Weis', '7a', 2030)`
	insSchuelerGeloescht = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
	                        VALUES ($1, 'Aus', 'Weis', '7a', 2030, now())`
	insLehrkraft = `INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
	                VALUES ($1, 'Leh', 'Rer', $2, 'kollegium', true)`
	constraintAusweis = "uniq_ausweis_ueber_personen"
)

func TestAusweisEindeutigUeberPersonen(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("Nummer eines Schülers ist für eine Lehrkraft vergeben", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchuelerAktiv, "AW-1")
			erwarteConstraintVerletzung(t, tx, constraintAusweis, insLehrkraft, "AW-1", "aw1@test.invalid")
		})
	})

	t.Run("Nummer einer Lehrkraft ist für einen Schüler vergeben", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Lehrkraft", insLehrkraft, "AW-2", "aw2@test.invalid")
			erwarteConstraintVerletzung(t, tx, constraintAusweis, insSchuelerAktiv, "AW-2")
		})
	})

	t.Run("gelöschter Schüler gibt die Nummer frei, Wiederherstellen stößt an", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "gelöschter Schüler", insSchuelerGeloescht, "AW-3")
			erwarteErfolg(t, tx, "Lehrkraft mit der freigegebenen Nummer", insLehrkraft, "AW-3", "aw3@test.invalid")
			erwarteConstraintVerletzung(t, tx, constraintAusweis,
				`UPDATE schueler SET deleted_at = NULL WHERE barcode_id = $1`, "AW-3")
		})
	})

	t.Run("Ändern auf die Nummer der anderen Seite", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchuelerAktiv, "AW-4S")
			erwarteErfolg(t, tx, "Lehrkraft", insLehrkraft, "AW-4L", "aw4@test.invalid")
			erwarteConstraintVerletzung(t, tx, constraintAusweis,
				`UPDATE schueler SET barcode_id = $1 WHERE barcode_id = 'AW-4S'`, "AW-4L")
			erwarteConstraintVerletzung(t, tx, constraintAusweis,
				`UPDATE benutzer SET barcode_id = $1 WHERE barcode_id = 'AW-4L'`, "AW-4S")
		})
	})

	t.Run("Gegenprobe: verschiedene Nummern und Lehrkräfte ohne Ausweis", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchuelerAktiv, "AW-5S")
			erwarteErfolg(t, tx, "Lehrkraft", insLehrkraft, "AW-5L", "aw5@test.invalid")
			erwarteErfolg(t, tx, "Lehrkraft ohne Ausweis", insLehrkraft, nil, "aw5a@test.invalid")
			erwarteErfolg(t, tx, "zweite Lehrkraft ohne Ausweis", insLehrkraft, nil, "aw5b@test.invalid")
		})
	})

	// Ein Doppel aus der Zeit vor Migration 118 darf weder den Start noch eine Änderung an
	// anderen Feldern blockieren. Geprüft wird nur, wenn sich die Nummer wirklich ändert.
	t.Run("Altdoppel blockiert unbeteiligte Änderungen nicht", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			// Nur die Lehrkraft ohne Trigger: Der Schüler braucht den Trigger, der seine Klasse
			// ins Vokabular einträgt.
			erwarteErfolg(t, tx, "Schüler", insSchuelerAktiv, "AW-6")
			erwarteErfolg(t, tx, "Trigger aus", `SET LOCAL session_replication_role = replica`)
			erwarteErfolg(t, tx, "Lehrkraft mit derselben Nummer", insLehrkraft, "AW-6", "aw6@test.invalid")
			erwarteErfolg(t, tx, "Trigger an", `SET LOCAL session_replication_role = origin`)
			erwarteErfolg(t, tx, "Lehrkraft ändern, Nummer gleich",
				`UPDATE benutzer SET vorname = 'Neu', barcode_id = barcode_id WHERE barcode_id = $1`, "AW-6")
			erwarteErfolg(t, tx, "Schüler ändern, Nummer gleich",
				`UPDATE schueler SET vorname = 'Neu', barcode_id = barcode_id WHERE barcode_id = $1`, "AW-6")
		})
	})
}

// TestAusweisEindeutigUeberPersonen_Gleichzeitig: Zwei Arbeitsplätze vergeben dieselbe Nummer
// gleichzeitig, einer an einen Schüler, einer an eine Lehrkraft. Ohne Sperre sieht keine der
// beiden Prüfungen die noch nicht festgeschriebene Zeile der anderen, und beide kommen durch.
func TestAusweisEindeutigUeberPersonen_Gleichzeitig(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM schueler WHERE barcode_id = 'AW-GLEICH'`,
			`DELETE FROM benutzer WHERE barcode_id = 'AW-GLEICH'`,
		} {
			if _, err := pool.Exec(ctx, q); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	})

	tx1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("erste Transaktion: %v", err)
	}
	defer SafeRollback(ctx, tx1)
	if _, err := tx1.Exec(ctx, insSchuelerAktiv, "AW-GLEICH"); err != nil {
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
		if _, err := tx2.Exec(ctx, insLehrkraft, "AW-GLEICH", "awgleich@test.invalid"); err != nil {
			ergebnis <- err
			return
		}
		ergebnis <- tx2.Commit(ctx)
	}()

	// Die zweite Vergabe soll warten, bis die erste feststeht. Kommt sie vorher zurück, hat sie
	// die Zeile der ersten nicht gesehen.
	select {
	case err := <-ergebnis:
		t.Fatalf("die zweite Vergabe wartete nicht auf die erste (Fehler: %v)", err)
	case <-time.After(300 * time.Millisecond):
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
