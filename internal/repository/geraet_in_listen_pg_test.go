package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
)

// Ein verliehenes Gerät zählte in der Leserliste und sperrte das Löschen, fehlte aber in
// Leserakte, Kontoauszug, Mahnliste, Mahnbrief und Abgängerliste: Ihre Abfragen gingen über das
// Exemplar. Ein überfälliges iPad muss überall stehen, wo ein Buch steht.
func TestGeraeteAusleihe_StehtInAllenListenDesLesers(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	marke := fmt.Sprintf("GER%d", time.Now().UnixNano())

	var leserID, geraetID, ausleiheID string
	if err := pool.QueryRow(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art, klasse, abgaenger_jahr)
		VALUES ($1, 'Ida', 'Pad', 'schueler', '10R1', 2026) RETURNING id`, "SCH-"+marke).Scan(&leserID); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ($1, $2) RETURNING id`,
		"iPad "+marke, "G-"+marke).Scan(&geraetID); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO ausleihen (geraet_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, NOW() - interval '30 days', NOW() - interval '10 days') RETURNING id`, geraetID, leserID).Scan(&ausleiheID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, schritt := range []struct{ sql, id string }{
			{`DELETE FROM ausleihen WHERE id = $1`, ausleiheID},
			{`DELETE FROM geraete WHERE id = $1`, geraetID},
			{`DELETE FROM leser WHERE id = $1`, leserID},
		} {
			if _, err := pool.Exec(auf, schritt.sql, schritt.id); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})
	modell, barcode := "iPad "+marke, "G-"+marke

	t.Run("Leserakte", func(t *testing.T) {
		ausleihen, err := NewLeserAbfragen(pool).LadeLaufendeAusleihen(ctx, leserID)
		if err != nil || len(ausleihen) != 1 || ausleihen[0].Titel != modell || ausleihen[0].BarcodeID != barcode || !ausleihen[0].IstGeraet {
			t.Errorf("LadeLaufendeAusleihen = %+v, %v; erwartet das Gerät", ausleihen, err)
		}
	})
	t.Run("Kontoauszug", func(t *testing.T) {
		buecher, err := LadeKontoauszugBuecher(ctx, pool, uuid.MustParse(leserID))
		if err != nil || len(buecher) != 1 || buecher[0].Titel != modell || buecher[0].Barcode != barcode {
			t.Errorf("LadeKontoauszugBuecher = %+v, %v; erwartet das Gerät", buecher, err)
		}
	})
	t.Run("Mahnliste", func(t *testing.T) {
		klassen, err := NewMahnwesenAbfragen(pool).LadeUeberfaelligeNachKlasse(ctx, "10R1")
		if err != nil {
			t.Fatal(err)
		}
		gefunden := false
		for _, k := range klassen {
			for _, s := range k.Schueler {
				for _, m := range s.Medien {
					gefunden = gefunden || (m.AusleiheID == ausleiheID && m.Titel == modell && m.Barcode == barcode)
				}
			}
		}
		if !gefunden {
			t.Errorf("LadeUeberfaelligeNachKlasse: das Gerät fehlt in %+v", klassen)
		}
	})
	t.Run("Mahnbrief", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Verwerfe(ctx, tx)
		briefe, err := NewMahnwesenAbfragen(pool).LadeMahnbriefe(ctx, tx, []string{ausleiheID})
		if err != nil || len(briefe) != 1 || len(briefe[0].Buecher) != 1 || briefe[0].Buecher[0].Titel != modell {
			t.Errorf("LadeMahnbriefe = %+v, %v; erwartet einen Brief mit dem Gerät", briefe, err)
		}
	})
	t.Run("Abgängerliste", func(t *testing.T) {
		zeilen, err := LadeAbgaengerAusleihen(ctx, pool, "10R1", []string{leserID})
		if err != nil || len(zeilen) != 1 || zeilen[0].Titel != modell {
			t.Errorf("LadeAbgaengerAusleihen = %+v, %v; erwartet das Gerät", zeilen, err)
		}
	})
}
