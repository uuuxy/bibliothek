package repository

import (
	"context"
	"errors"
	"sync"
	"testing"

	"bibliothek/db"
)

// Der Abschluss einer Inventur zählt nicht nur, er hält fest, welche Exemplare fehlen: Mit
// einer bloßen Zahl kann niemand ins Regal gehen und nachsehen, ob ein Buch fehlt oder falsch
// einsortiert war, und nach der Aussonderung fallen die Exemplare aus der Bedingung des
// Bereichs.
//
// Geprüft am echten Postgres: Die Buchung ist eine CTE aus UPDATE … RETURNING mit
// anschließendem INSERT. Ob dabei die nicht erfassten Zeilen ankommen, ist Verhalten der
// Datenbank.
func TestInventurFehlbestandNenntDieBuecher(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	bearbeiter := legeBearbeiterAn(t, pool)

	// Ein Titel mit Signatur, damit der Bericht auch die Regalangabe trägt; danach sortiert man
	// beim Nachsuchen. Quelle ist buecher_titel.signatur, der Text vom Buchrücken.
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

	repo := NewInventurAbfragen(pool)
	if err := repo.VerbucheScan(ctx, sessionID, gescannt); err != nil {
		t.Fatalf("Scan erfassen: %v", err)
	}

	verloren, err := repo.SchliesseLaufAb(ctx, sessionID, InventurBereich{}, bearbeiter)
	if err != nil {
		t.Fatalf("Abschluss: %v", err)
	}
	if verloren != 2 {
		t.Fatalf("%d Verluste gemeldet, erwartet 2 (das gescannte Exemplar darf nicht dabei sein)", verloren)
	}

	liste, err := repo.LadeVerluste(ctx, sessionID)
	if err != nil {
		t.Fatalf("Fehlbestand laden: %v", err)
	}
	if len(liste) != 2 {
		t.Fatalf("%d Zeilen im Fehlbestand, erwartet 2", len(liste))
	}

	// Die Abschrift muss lesbar sein: Eine Liste aus Barcodes allein hilft im Regal nicht.
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

	// Und sie überlebt das Löschen des Exemplars: Dann fragt jemand nach.
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE id = $1`, fehlt1); err != nil {
		t.Fatalf("Exemplar loeschen: %v", err)
	}
	nachher, err := repo.LadeVerluste(ctx, sessionID)
	if err != nil {
		t.Fatalf("Fehlbestand nach Loeschung: %v", err)
	}
	if len(nachher) != 2 {
		t.Fatalf("%d Zeilen nach dem Loeschen, erwartet weiterhin 2 — die Abschrift muss bleiben", len(nachher))
	}
}

// Scannen mehrere Geräte, hinterlässt ein Scan, der zugleich mit dem Abschluss eintrifft, ein
// vorliegendes Buch nicht als Verlust. Der Scan hält die Zeile des Laufs mit FOR SHARE, der
// Abschluss nimmt sie mit FOR UPDATE: Entweder schreibt der Scan zuerst fest, und der Abschluss
// sieht die Erfassung, oder der Abschluss war zuerst, und der Scan wird mit
// ErrInventurAbgeschlossen abgewiesen. „Erfolgreich gescannt und als Verlust ausgesondert"
// entsteht nie.
//
// Der Test erzwingt das enge Fenster: Der Scan beginnt seine Transaktion und hält die Sperre,
// während der Abschluss FOR UPDATE verlangt.
func TestInventurScanUndAbschlussAbgestimmt(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	for runde := 0; runde < 8; runde++ {
		resetInventurDaten(t, pool)
		bearbeiter := legeBearbeiterAn(t, pool)
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

		repo := NewInventurAbfragen(pool)
		var wg sync.WaitGroup
		start := make(chan struct{})
		var scanErr, finishErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			scanErr = repo.VerbucheScan(ctx, sessionID, exID)
		}()
		go func() {
			defer wg.Done()
			<-start
			tx, err := pool.Begin(ctx)
			if err != nil {
				finishErr = err
				return
			}
			defer db.Verwerfe(ctx, tx)
			txRepo := NewInventurAbfragen(tx)
			s, err := txRepo.SperreLaufFuerAbschluss(ctx, sessionID)
			if err != nil {
				finishErr = err
				return
			}
			if _, err := txRepo.SchliesseLaufAb(ctx, s.ID, s.Bereich(), bearbeiter); err != nil {
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

		// Der verbotene Zustand: Der Scan meldete Erfolg (erfasst), und das Buch ist Verlust.
		scanErfolg := scanErr == nil
		if scanErfolg && ausgesondert {
			t.Fatalf("Runde %d: Buch als Verlust ausgesondert, obwohl der Scan Erfolg meldete (erfasst=%v)", runde, erfasst)
		}
		// Und die Gegenrichtung: Wurde der Scan abgewiesen, liegt es am abgeschlossenen Lauf
		// und nicht an einem anderen Fehler.
		if scanErr != nil && !errors.Is(scanErr, ErrInventurAbgeschlossen) {
			t.Fatalf("Runde %d: unerwarteter Scan-Fehler: %v", runde, scanErr)
		}
		// Konsistenz: erfasst genau dann, wenn der Scan Erfolg hatte.
		if scanErfolg != erfasst {
			t.Fatalf("Runde %d: Scan-Erfolg=%v, aber erfasst=%v", runde, scanErfolg, erfasst)
		}
	}
}
