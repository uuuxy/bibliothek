package repository

import (
	"context"
	"errors"
	"sync"
	"testing"

	"bibliothek/db"
)

// TestInventurFehlbestandNenntDieBuecher belegt, dass der Abschluss einer Inventur nicht
// nur ZÄHLT, sondern festhält, WELCHE Exemplare fehlen.
//
// Vorher gab FinishInventurSession allein eine Zahl zurück: „47 Verluste". Damit kann
// niemand ins Regal gehen und nachsehen, ob ein Buch wirklich fehlt oder nur falsch
// einsortiert war. Und rekonstruieren liess es sich danach auch nicht mehr — die
// Exemplare fallen durch die Aussonderung aus der Scope-Bedingung.
//
// Geprüft wird an echtem Postgres: Die Buchung ist ein CTE aus UPDATE ... RETURNING mit
// anschliessendem INSERT. Ob dabei wirklich die nicht erfassten Zeilen ankommen, ist
// Datenbankverhalten und mit pgxmock nicht nachzubilden.
func TestInventurFehlbestandNenntDieBuecher(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	// Ein Titel mit Signatur, damit der Bericht auch die Regalangabe trägt — danach
	// sortiert man beim Nachsuchen. Quelle ist buecher_titel.signatur, der Text vom
	// Buchrücken: Die Abschrift lief früher über einen Join auf die nie gepflegte
	// Tabelle `signatures` und war deshalb immer leer (Migration 060).
	var titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, signatur) VALUES ('Deutschbuch 7', 'Biermann', 'LMF Deu 7')
		RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	exemplar := func(barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			VALUES ($1, $2, true) RETURNING id
		`, titelID, barcode).Scan(&id); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", barcode, err)
		}
		return id
	}
	gescannt := exemplar("INV-DA")
	fehlt1 := exemplar("INV-WEG-1")
	_ = exemplar("INV-WEG-2")

	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventur_sessions (scope_type) VALUES ('global') RETURNING id
	`).Scan(&sessionID); err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}

	repo := NewInventoryRepository(pool)
	if err := repo.RecordInventurScan(ctx, sessionID, gescannt); err != nil {
		t.Fatalf("Scan erfassen: %v", err)
	}

	verloren, err := repo.FinishInventurSession(ctx, sessionID, InventurScope{})
	if err != nil {
		t.Fatalf("Abschluss: %v", err)
	}
	if verloren != 2 {
		t.Fatalf("%d Verluste gemeldet, erwartet 2 (das gescannte Exemplar darf nicht dabei sein)", verloren)
	}

	liste, err := repo.LadeInventurVerluste(ctx, sessionID)
	if err != nil {
		t.Fatalf("Fehlbestand laden: %v", err)
	}
	if len(liste) != 2 {
		t.Fatalf("%d Zeilen im Fehlbestand, erwartet 2", len(liste))
	}

	// Die Abschrift muss lesbar sein — eine Liste aus Barcodes allein hilft im Regal nicht.
	gefunden := map[string]InventurVerlust{}
	for _, v := range liste {
		gefunden[v.BarcodeID] = v
	}
	if _, ok := gefunden["INV-DA"]; ok {
		t.Error("das gescannte Exemplar steht faelschlich im Fehlbestand")
	}
	eintrag, ok := gefunden["INV-WEG-1"]
	if !ok {
		t.Fatal("INV-WEG-1 fehlt im Fehlbestand")
	}
	if eintrag.Titel != "Deutschbuch 7" || eintrag.Autor != "Biermann" || eintrag.Signatur != "LMF Deu 7" {
		t.Errorf("Abschrift unvollstaendig: %+v", eintrag)
	}

	// Und sie ueberlebt das Loeschen des Exemplars — genau dann fragt jemand nach.
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE id = $1`, fehlt1); err != nil {
		t.Fatalf("Exemplar loeschen: %v", err)
	}
	nachher, err := repo.LadeInventurVerluste(ctx, sessionID)
	if err != nil {
		t.Fatalf("Fehlbestand nach Loeschung: %v", err)
	}
	if len(nachher) != 2 {
		t.Fatalf("%d Zeilen nach dem Loeschen, erwartet weiterhin 2 — die Abschrift muss bleiben", len(nachher))
	}
}

// TestInventurScanFinishKoordination belegt den Nebenläufigkeits-Fix (19.08.2026):
// Im Multi-Scanner-Betrieb darf ein Scan, der zeitgleich zum Abschluss eintrifft, ein
// physisch vorliegendes Buch NICHT als Verlust hinterlassen. Der Scan hält die Session
// FOR SHARE, der Abschluss FOR UPDATE — entweder committet der Scan zuerst (dann sieht
// der Verlust-Snapshot die Erfassung) oder der Abschluss war zuerst (dann wird der Scan
// mit ErrInventurAbgeschlossen abgewiesen). Der verbotene Endzustand „erfolgreich
// gescannt UND als Verlust ausgesondert" darf NIE entstehen.
//
// Der Test erzwingt das enge Fenster: Der Scan startet seine Transaktion und hält die
// FOR-SHARE-Sperre, WÄHREND der Abschluss versucht, FOR UPDATE zu nehmen.
func TestInventurScanFinishKoordination(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	for runde := 0; runde < 8; runde++ {
		resetInventurDaten(t, pool)
		var titelID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel) VALUES ('KoordTitel') RETURNING id`).Scan(&titelID); err != nil {
			t.Fatalf("Titel: %v", err)
		}
		var exID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1,'INV-KOORD',true) RETURNING id`,
			titelID).Scan(&exID); err != nil {
			t.Fatalf("Exemplar: %v", err)
		}
		var sessionID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO inventur_sessions (scope_type) VALUES ('global') RETURNING id`).Scan(&sessionID); err != nil {
			t.Fatalf("Session: %v", err)
		}

		repo := NewInventoryRepository(pool)
		var wg sync.WaitGroup
		start := make(chan struct{})
		var scanErr, finishErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			scanErr = repo.RecordInventurScan(ctx, sessionID, exID)
		}()
		go func() {
			defer wg.Done()
			<-start
			tx, err := pool.Begin(ctx)
			if err != nil {
				finishErr = err
				return
			}
			defer db.SafeRollback(ctx, tx)
			txRepo := NewInventoryRepository(tx)
			s, err := txRepo.SperreInventurSessionFuerAbschluss(ctx, sessionID)
			if err != nil {
				finishErr = err
				return
			}
			if _, err := txRepo.FinishInventurSession(ctx, s.ID, s.Scope()); err != nil {
				finishErr = err
				return
			}
			finishErr = tx.Commit(ctx)
		}()
		close(start)
		wg.Wait()
		if finishErr != nil {
			t.Fatalf("Runde %d: Abschluss-Fehler: %v", runde, finishErr)
		}

		// Zustand prüfen: erfasst? ausgesondert (=VERLUST)?
		var erfasst, ausgesondert bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM inventur_erfassungen WHERE session_id=$1 AND exemplar_id=$2)`,
			sessionID, exID).Scan(&erfasst); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx,
			`SELECT ist_ausgesondert FROM buecher_exemplare WHERE id=$1`, exID).Scan(&ausgesondert); err != nil {
			t.Fatal(err)
		}

		// Der VERBOTENE Zustand: Scan meldete Erfolg (erfasst) UND das Buch ist Verlust.
		scanErfolg := scanErr == nil
		if scanErfolg && ausgesondert {
			t.Fatalf("Runde %d: Buch als Verlust ausgesondert, obwohl der Scan Erfolg meldete (erfasst=%v)", runde, erfasst)
		}
		// Und die Gegenrichtung: Wurde der Scan abgewiesen, muss es an der abgeschlossenen
		// Session liegen — nicht an einem anderen Fehler.
		if scanErr != nil && !errors.Is(scanErr, ErrInventurAbgeschlossen) {
			t.Fatalf("Runde %d: unerwarteter Scan-Fehler: %v", runde, scanErr)
		}
		// Konsistenz: erfasst genau dann, wenn der Scan Erfolg hatte.
		if scanErfolg != erfasst {
			t.Fatalf("Runde %d: Scan-Erfolg=%v, aber erfasst=%v", runde, scanErfolg, erfasst)
		}
	}
}
