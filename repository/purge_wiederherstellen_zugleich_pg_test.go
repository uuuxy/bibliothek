package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Endgültig gelöscht wird nur, was im Papierkorb liegt, auch wenn jemand den Leser im selben
// Augenblick wiederherstellt. Die Frage „im Papierkorb?" sperrt dafür die Zeile bis zum Löschen;
// ohne die Sperre läse sie den Stand vor dem Wiederherstellen, und das Löschen träfe danach den
// wiederhergestellten, aktiven Leser.
func TestLoescheLeserEndgueltig_WiederherstellenZugleichLaesstDenLeserStehen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art, deleted_at)
		VALUES ('S-PURGE-ZUGLEICH', 'Zu', 'Gleich', '9a', 2029, 'schueler', NOW()) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM leser WHERE id = $1`, id); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})

	// Das Wiederherstellen hält die Zeile; festgeschrieben ist noch nichts.
	wieder, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Verwerfe(context.Background(), wieder)
	if _, err := wieder.Exec(ctx, `UPDATE leser SET deleted_at = NULL WHERE id = $1`, id); err != nil {
		t.Fatalf("wiederherstellen: %v", err)
	}

	ergebnis := make(chan error, 1)
	go func() { ergebnis <- repository.NewLeserAbfragen(pool).LoescheLeserEndgueltig(ctx, id, "") }()

	// Warten, bis das Löschen an der Zeile hängt. Gezählt werden Sperren von Zeilen, nicht die
	// Sperre, an der die Testpakete einander abwarten.
	frist := time.Now().Add(10 * time.Second)
	for wartend := 0; wartend == 0; {
		select {
		case err := <-ergebnis:
			t.Fatalf("das Löschen wartete nicht auf das Wiederherstellen: %v", err)
		default:
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'
			  AND wait_event IN ('transactionid', 'tuple')`).Scan(&wartend); err != nil {
			t.Fatalf("wartendes Löschen suchen: %v", err)
		}
		if wartend == 0 && time.Now().After(frist) {
			t.Fatal("das Löschen erschien nie als wartend auf die Zeile")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := wieder.Commit(ctx); err != nil {
		t.Fatalf("Wiederherstellen festschreiben: %v", err)
	}

	select {
	case err := <-ergebnis:
		var blockiert *repository.LoeschenBlockiertFehler
		if !errors.As(err, &blockiert) || !blockiert.NichtImPapierkorb {
			t.Errorf("LoescheLeserEndgueltig nach dem Wiederherstellen = %v, erwartet die Blockade „nicht im Papierkorb“", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("das Löschen kam nach dem Wiederherstellen nicht zurück")
	}
	var aktiv bool
	if err := pool.QueryRow(ctx, `SELECT deleted_at IS NULL FROM leser WHERE id = $1`, id).Scan(&aktiv); err != nil {
		t.Fatalf("der wiederhergestellte Leser ist nicht mehr da: %v", err)
	}
	if !aktiv {
		t.Error("der Leser liegt nach dem Wiederherstellen wieder im Papierkorb")
	}
}
