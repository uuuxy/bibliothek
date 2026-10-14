package leser

import (
	"context"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// TestLUSDImport_DubletteInDerselbenDatei prüft, was passiert, wenn EIN LUSD-Export
// dieselbe lusd_id zweimal enthält (kommt in der Praxis vor, siehe Migration 048).
//
// Der Schutz dagegen steckt nicht in einer expliziten Prüfung, sondern in der Reihenfolge:
// Solange jeder Neuzugang sofort eingefügt wird, sieht repository.FindeAktivenSchuelerZuLUSDID die
// erste Zeile bereits innerhalb derselben Transaktion und behandelt die zweite als
// Rückkehrer-Update. Wird das Einfügen dagegen bis zum Schleifenende aufgeschoben
// (Batch/CopyFrom), findet die zweite Zeile nichts, beide landen im Batch und kollidieren am
// partiellen Unique-Index uniq_schueler_lusd_id_active — SQLSTATE 23505 reißt den GESAMTEN
// Import mit, nicht nur die doppelte Zeile.
func TestLUSDImport_DubletteInDerselbenDatei(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	s := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	if _, err := s.berechneLUSDAenderungen(ctx, []lusd.Zeile{
		{LUSDID: "L-DUP", Vorname: "Anna", Nachname: "Doppelt", Klasse: "5a", Nr: 1},
		{LUSDID: "L-DUP", Vorname: "Anna", Nachname: "Doppelt", Klasse: "5b", Nr: 2},
	}, true, true); err != nil {
		t.Fatalf("Dublette in der Importdatei darf den Import nicht abreißen lassen: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schueler WHERE lusd_id = 'L-DUP' AND deleted_at IS NULL`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("erwartet genau 1 aktive Zeile für L-DUP, waren %d", count)
	}

	// Die zweite Zeile gewinnt (letzter Stand aus dem Export).
	var klasse string
	if err := pool.QueryRow(ctx,
		`SELECT klasse FROM schueler WHERE lusd_id = 'L-DUP' AND deleted_at IS NULL`).Scan(&klasse); err != nil {
		t.Fatal(err)
	}
	if klasse != "05B" {
		t.Errorf("Klasse = %q, erwartet 05B (zweite Zeile überschreibt die erste)", klasse)
	}
}
