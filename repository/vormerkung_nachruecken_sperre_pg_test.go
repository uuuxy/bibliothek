package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"
)

// Wer nachrückt, entscheidet die Reihenfolge der Vormerkungen, nicht, wessen Leserzeile gerade
// frei ist. Die Ausleihe sperrt die Zeile des Lesers für die Dauer ihrer Buchung; steht der
// Älteste in der Warteschlange in diesem Augenblick an der Theke, bekommt trotzdem er das
// freigewordene Buch und nicht der Nächste hinter ihm.
func TestNachruecken_GesperrteLeserzeileUeberspringtDenAeltestenNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	exemplar := seedSignaturMitExemplaren(t, pool, "Nachruecksperre", 1)[0]
	titel := titelIDVonExemplar(t, pool, exemplar)
	aelteste := seedSchueler(t, pool, "NRS-1", "Aelteste", "7a")
	naechste := seedSchueler(t, pool, "NRS-2", "Naechste", "7a")
	for i, leser := range []string{aelteste, naechste} {
		if _, err := pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, erstellt_am)
			VALUES ($1, $2, 'wartend', now() - make_interval(days => $3))`, titel, leser, 10-i); err != nil {
			t.Fatalf("Vormerkung anlegen: %v", err)
		}
	}

	theke := beginne(t, pool)
	defer db.SafeRollback(ctx, theke)
	if err := SperreLeserzeile(ctx, theke, aelteste); err != nil {
		t.Fatalf("Leserzeile sperren: %v", err)
	}

	nachruecken := beginne(t, pool)
	defer db.SafeRollback(ctx, nachruecken)
	bedient, err := bedieneNaechstenWartenden(ctx, nachruecken, exemplar, titel, time.Now())
	if err != nil || !bedient {
		t.Fatalf("Nachrücken: bedient=%v, Fehler %v", bedient, err)
	}
	if err := nachruecken.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var bereitFuer string
	if err := pool.QueryRow(ctx, `SELECT schueler_id::text FROM vormerkungen WHERE titel_id = $1 AND status = 'abholbereit'`, titel).
		Scan(&bereitFuer); err != nil {
		t.Fatalf("bereitgestellte Vormerkung lesen: %v", err)
	}
	if bereitFuer != aelteste {
		t.Errorf("das Buch liegt für %s bereit, erwartet für die älteste Vormerkung (%s)", bereitFuer, aelteste)
	}
}
