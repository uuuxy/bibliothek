package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// TestGetNextSequence_NumerischNichtLexikografisch sichert den Fix gegen den
// lexikografischen Kollaps (#1) ab: Liegen 'B-99999' und 'B-100000' im Bestand, muss die
// nächste Nummer 100001 sein — nicht 100000. Lexikografisch gilt 'B-99999' > 'B-100000'
// (die '9' schlägt die '1'); die alte ORDER-BY-DESC-Query hätte 99999 als Maximum
// geliefert und das System endlos 'B-100000' neu anlegen lassen (UNIQUE-Crash, Einfrieren
// der Barcode-Vergabe). Der zentrale Generator speist ALLE Schüler- und Exemplar-Barcodes.
func TestGetNextSequence_NumerischNichtLexikografisch(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	titel := titelMitMeldebestand(t, pool, "Seq-Test", 0)
	exemplar(t, pool, titel, "B-99999", true, "")
	exemplar(t, pool, titel, "B-100000", true, "") // lexikografisch KLEINER als B-99999

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)

	seqRepo := repository.NewSequenceRepository(tx)
	got, err := seqRepo.GetNextSequence(ctx, "buecher_exemplare", "barcode_id", "B-")
	if err != nil {
		t.Fatalf("GetNextSequence: %v", err)
	}
	if got != 100001 {
		t.Errorf("nächste Nummer: erwartet 100001, war %d "+
			"(lexikografische Sortierung hätte 100000 geliefert -> UNIQUE-Crash)", got)
	}
}

// TestGetNextSequence_LeererBestandFallback: Ohne passende Barcodes startet die Sequenz
// beim Fallback 10001 — auch bei komplett leerer Tabelle. Der Advisory-Lock muss auch dann
// sauber genommen werden (die Lock-Zeile ist der treibende LEFT-JOIN-Partner).
func TestGetNextSequence_LeererBestandFallback(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)

	seqRepo := repository.NewSequenceRepository(tx)
	got, err := seqRepo.GetNextSequence(ctx, "buecher_exemplare", "barcode_id", "B-")
	if err != nil {
		t.Fatalf("GetNextSequence: %v", err)
	}
	if got != 10001 {
		t.Errorf("Fallback: erwartet 10001, war %d", got)
	}
}
