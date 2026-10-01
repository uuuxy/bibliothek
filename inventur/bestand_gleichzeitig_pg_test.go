package inventur

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Zwei Plätze speichern denselben Titel im selben Augenblick. Der Vergleich mit dem gesehenen
// Stand (setzeBestand) trägt dann nur, weil UpdateBook davor die Zeile des Titels schreibt:
// Der zweite Vorgang wartet auf den ersten und zählt erst danach. Zählte er vorher, sähe er
// den Stand von vor dem ersten und sonderte ein zweites Mal aus.
func TestBestand_ZweiterSpeichervorgangZaehltNachDemErsten(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	marke := fmt.Sprintf("GLEICH%d", time.Now().UnixNano())

	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ($1, $2, 'Buch') RETURNING id`,
		"Zwei Plätze "+marke, marke).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id = $1`,
			`DELETE FROM buecher_titel WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, titelID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})
	exemplar := func(q repository.DBQueryer, nr int) {
		t.Helper()
		if _, err := q.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, true)`,
			titelID, fmt.Sprintf("B-GL-%d-%s", nr, marke)); err != nil {
			t.Fatalf("Exemplar %d anlegen: %v", nr, err)
		}
	}
	for nr := 1; nr <= 3; nr++ {
		exemplar(pool, nr)
	}
	// {im Bestand, ausgesondert}
	zaehle := func() [2]int {
		t.Helper()
		var z [2]int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FILTER (WHERE NOT ist_ausgesondert), count(*) FILTER (WHERE ist_ausgesondert)
			FROM buecher_exemplare WHERE titel_id = $1`, titelID).Scan(&z[0], &z[1]); err != nil {
			t.Fatalf("zählen: %v", err)
		}
		return z
	}

	buecher, err := repo.ListBooksByIDs(ctx, []string{titelID})
	if err != nil || len(buecher) != 1 || buecher[0].Stock != 3 {
		t.Fatalf("Einzel-Read: %v, %d Titel — erwartet einen mit Bestand 3", err, len(buecher))
	}
	buch := buecher[0]

	// Der erste Vorgang hält die Zeile des Titels, wie UpdateBook es tut, und hat ein viertes
	// Exemplar angelegt; festgeschrieben ist noch nichts.
	erster, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(context.Background(), erster)
	if _, err := erster.Exec(ctx, `UPDATE buecher_titel SET aktualisiert_am = NOW() WHERE id = $1`, titelID); err != nil {
		t.Fatalf("erster Vorgang, Titel: %v", err)
	}
	exemplar(erster, 4)

	// Der zweite verringert von 3 auf 1 und hat wie der erste den Bestand 3 gesehen.
	ergebnis := make(chan error, 1)
	go func() {
		gesehen := 3
		ergebnis <- repo.UpdateBook(ctx, titelID, buch, &Bestandsangabe{Soll: 1, Gesehen: &gesehen})
	}()

	select {
	case err := <-ergebnis:
		t.Fatalf("der zweite Speichervorgang hat nicht auf den ersten gewartet (Antwort: %v); im Bestand %v", err, zaehle())
	case <-time.After(500 * time.Millisecond):
	}
	if err := erster.Commit(ctx); err != nil {
		t.Fatalf("erster Vorgang, Festschreiben: %v", err)
	}

	select {
	case err := <-ergebnis:
		var veraltet *BestandVeraltet
		if !errors.As(err, &veraltet) {
			t.Fatalf("der zweite Speichervorgang wurde nicht abgelehnt (Antwort: %v); im Bestand und ausgesondert: %v", err, zaehle())
		}
		if veraltet.Aktuell != 4 {
			t.Errorf("die Ablehnung nennt den Stand %d, erwartet 4", veraltet.Aktuell)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("der zweite Speichervorgang kommt nach dem Festschreiben des ersten nicht zurück")
	}
	if ist := zaehle(); ist != [2]int{4, 0} {
		t.Errorf("im Bestand %d, ausgesondert %d — erwartet 4 und 0", ist[0], ist[1])
	}
}
