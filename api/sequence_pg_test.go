package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
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

// TestGetNextSequence_UeberlangeNummerBlockiertNicht: Der Ausweis-Barcode darf beim
// Anlegen eines Schülers frei eingegeben werden. Ein verrutschter Scan legt damit eine
// Nummer an, die keine bigint mehr ist — und ließ danach JEDE automatische Vergabe mit
// "value out of range for type bigint" scheitern. Ein einziger krummer Datensatz legte
// also die Ausweisvergabe für alle lahm. Zu lange Nummern werden übergangen.
func TestGetNextSequence_UeberlangeNummerBlockiertNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	schueler(t, pool, "S-10005")
	schueler(t, pool, "S-999999999999999999999") // 21 Ziffern, sprengt bigint

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)

	seqRepo := repository.NewSequenceRepository(tx)
	got, err := seqRepo.GetNextSequence(ctx, "schueler", "barcode_id", "S-")
	if err != nil {
		t.Fatalf("GetNextSequence: %v", err)
	}
	if got != 10006 {
		t.Errorf("nächste Ausweisnummer: erwartet 10006, war %d", got)
	}
}

// TestGetNextSequence_MultibytePrefix: Wenn das Präfix Multibyte-Zeichen enthält
// (wie "Schüler-", wo 'ü' 2 Bytes belegt), darf len(prefix)+1 nicht als Offset
// für die substr-Funktion in PostgreSQL verwendet werden, da diese 1-basiert über
// Characters (Runes) iteriert und nicht über Bytes. Dieser Test sichert ab, dass
// utf8.RuneCountInString genutzt wird.
func TestGetNextSequence_MultibytePrefix(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// "Schüler-" hat 9 Bytes, aber 8 Zeichen.
	schueler(t, pool, "Schüler-123")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)

	seqRepo := repository.NewSequenceRepository(tx)
	got, err := seqRepo.GetNextSequence(ctx, "schueler", "barcode_id", "Schüler-")
	if err != nil {
		t.Fatalf("GetNextSequence: %v", err)
	}
	if got != 124 {
		t.Errorf("nächste Ausweisnummer für Multibyte-Präfix: erwartet 124, war %d", got)
	}
}

// schueler legt einen aktiven Schüler mit gegebenem Ausweis-Barcode an.
func schueler(t *testing.T, pool *pgxpool.Pool, barcode string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ($1, 'Test', 'Schueler', '5a', 2030)`, barcode)
	if err != nil {
		t.Fatalf("Schüler %q anlegen: %v", barcode, err)
	}
}
