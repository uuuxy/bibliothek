package api

import (
	"context"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// mitTx führt fn mit einer echten, am Ende zurückgerollten Transaktion aus.
func mitTx(t *testing.T, pool *pgxpool.Pool, fn func(tx pgx.Tx)) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer db.SafeRollback(ctx, tx)
	fn(tx)
}

// TestSammleNachbestellungen_BereitsBestellteZaehlen sichert den Budget-Fix ab: Ein
// Titel mit Meldebestand 5, 0 verfügbaren, aber 5 bereits bestellten Exemplaren darf
// NICHT erneut zur Bestellung vorgeschlagen werden. Vorher zählte die Query nur
// ist_ausleihbar=true und bestellte Woche für Woche dieselben Titel nach.
func TestSammleNachbestellungen_BereitsBestellteZaehlen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	// Titel A: Meldebestand 5, 0 verfügbar, 5 bereits bestellt -> KEIN Bedarf mehr.
	titelA := titelMitMeldebestand(t, pool, "Voll bestellt", 5)
	for i := 0; i < 5; i++ {
		exemplar(t, pool, titelA, "B-A"+string(rune('0'+i)), false, "bestellt")
	}

	// Titel B: Meldebestand 5, 2 verfügbar, 1 bestellt -> Restbedarf 2.
	titelB := titelMitMeldebestand(t, pool, "Teilbedarf", 5)
	exemplar(t, pool, titelB, "B-B1", true, "")
	exemplar(t, pool, titelB, "B-B2", true, "")
	exemplar(t, pool, titelB, "B-B3", false, "Im Zulauf - Klett")

	mitTx(t, pool, func(tx pgx.Tx) {
		items, err := sammleNachbestellungen(context.Background(), tx)
		if err != nil {
			t.Fatalf("sammleNachbestellungen: %v", err)
		}

		byTitel := map[string]reorderItem{}
		for _, it := range items {
			byTitel[it.Titel] = it
		}

		if _, drin := byTitel["Voll bestellt"]; drin {
			t.Error("vollständig bestellter Titel wurde erneut zur Bestellung vorgeschlagen (Budget-Falle)")
		}
		b, ok := byTitel["Teilbedarf"]
		if !ok {
			t.Fatal("Titel mit echtem Restbedarf fehlt in der Bestellliste")
		}
		if b.OrderQty != 2 {
			t.Errorf("Restbedarf: erwartet 2 (5 − 2 verfügbar − 1 bestellt), war %d", b.OrderQty)
		}
	})
}

// TestNaechsteBarcodeNummer_Numerisch sichert den Fix gegen die lexikografische
// Sortierung: Bei 'B-99999' und 'B-100000' im Bestand muss die nächste Nummer 100001
// sein, nicht 100000 (was am UNIQUE-Constraint scheitern und das Bestellsystem
// einfrieren würde).
func TestNaechsteBarcodeNummer_Numerisch(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	titel := titelMitMeldebestand(t, pool, "Barcode-Test", 0)
	exemplar(t, pool, titel, "B-99999", true, "")
	exemplar(t, pool, titel, "B-100000", true, "") // lexikografisch KLEINER als B-99999

	mitTx(t, pool, func(tx pgx.Tx) {
		got := naechsteBarcodeNummer(context.Background(), tx)
		if got != 100001 {
			t.Errorf("nächste Barcode-Nummer: erwartet 100001, war %d "+
				"(lexikografische Sortierung hätte 100000 geliefert -> UNIQUE-Crash)", got)
		}
	})
}

// TestNaechsteBarcodeNummer_LeererBestand: Fallback 10001, wenn keine B-Barcodes da sind.
func TestNaechsteBarcodeNummer_LeererBestand(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	mitTx(t, pool, func(tx pgx.Tx) {
		if got := naechsteBarcodeNummer(context.Background(), tx); got != 10001 {
			t.Errorf("Fallback erwartet 10001, war %d", got)
		}
	})
}
