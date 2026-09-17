package repository

import (
	"context"
	"testing"
)

// TestListLeserMitStats_ZeigtKollegium ist der Rot-Test der Leserdatei: Bis zum
// 16.09.2026 gab es nur die Schülerdatei, und sie las die Sicht `schueler`. Ein Kollege
// stand in keiner Liste — man sah nicht, welche Bücher er hat.
//
// Zwei Eigenschaften prüft dieser Test, und keine ist selbstverständlich:
//
//  1. Die Spalten klasse, abgaenger_jahr und barcode_id sind seit Migration 123 nullbar.
//     Ohne coalesce zerbricht der Scan an der ersten Lehrkraft — ein 500 an der Stelle,
//     wo vorher eine Liste stand (Bugklasse „nullbare Spalte → 500").
//  2. Kollegium steht VORN. Die ungefilterte Liste ist bei 500 Zeilen gekappt; stünde
//     das Kollegium hinter den Schülern, wäre es bei 875 Schülern nie zu sehen — und
//     zwar lautlos.
func TestListLeserMitStats_ZeigtKollegium(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchSchueler(t, pool, "LLIST-1", "Lena", "Hoffmann")
	seedSuchLeser(t, pool, "LLIST-2", "Katrin", "Wendlandt", "lehrkraft")
	seedSuchLeser(t, pool, "", "Hendrik", "Aalbers", "liv")

	repo := NewStudentRepository(pool)

	// Die Schülerdatei-Sicht bleibt, was sie war.
	nurSchueler, err := repo.ListStudentsWithStats(ctx, nil, "")
	if err != nil {
		t.Fatalf("Schülerliste: %v", err)
	}
	if len(nurSchueler) != 1 || nurSchueler[0].Nachname != "Hoffmann" {
		t.Errorf("ohne Art-Angabe darf die Liste nur Schüler zeigen, bekommen: %+v", nurSchueler)
	}

	alle, err := repo.ListLeserMitStats(ctx, nil, "")
	if err != nil {
		t.Fatalf("Leserliste: %v", err)
	}
	if len(alle) != 3 {
		t.Fatalf("die Leserdatei zeigt alle drei, bekommen: %d (%+v)", len(alle), alle)
	}
	if alle[0].Nachname != "Aalbers" || alle[1].Nachname != "Wendlandt" {
		t.Errorf("Kollegium gehört an den Anfang (die Liste ist bei 500 gekappt), bekommen: %s, %s, %s",
			alle[0].Nachname, alle[1].Nachname, alle[2].Nachname)
	}
	if alle[0].Art != "liv" || alle[1].Art != "lehrkraft" || alle[2].Art != "schueler" {
		t.Errorf("Art je Zeile: %q, %q, %q", alle[0].Art, alle[1].Art, alle[2].Art)
	}
	if alle[0].BarcodeID != "" || alle[0].Klasse != "" || alle[0].AbgaengerJahr != 0 {
		t.Errorf("ein Kollege ohne Ausweis und ohne Klasse muss leere Felder liefern: %+v", alle[0])
	}

	// Die Suche läuft über alle — genau das ist „eine Suche über alle Leser".
	treffer, err := repo.ListLeserMitStats(ctx, nil, "Wendlandt")
	if err != nil {
		t.Fatalf("Suche in der Leserdatei: %v", err)
	}
	if len(treffer) != 1 || treffer[0].Nachname != "Wendlandt" {
		t.Errorf("Suche nach einem Kollegen: %+v", treffer)
	}
}
