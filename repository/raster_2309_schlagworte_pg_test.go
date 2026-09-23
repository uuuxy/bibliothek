//go:build raster

package repository

// Nachstellung Rasterdurchgang 23.09.2026 über die Schlagworte (Migrationen 138 und 143).
// Build-Tag raster: läuft nur mit -tags raster. Rot heißt „bestätigt".
//
// V1 und V3 (Titel am Verweis, Kette — beides bei gleichzeitigen Schreibern) sind mit
// Migration 144 behoben und stehen als dauerhafte Tests in schlagworte_pflege_pg_test.go.
// V2 wartet auf eine Entscheidung (docs/OFFEN.md 4.20) und bleibt deshalb hier: rot, bis
// entschieden ist, ob Umbenennen einen Verweis von der alten Schreibweise hinterlässt.

import (
	"context"
	"slices"
	"testing"
)

// Verdacht V2 (Frage 3, zwei Türen mit verschiedener Regel): Zusammenführen lässt das alte
// Wort als Verweis stehen, Umbenennen nicht. Das Buchformular und der Bestellkorb schicken
// die ganze Menge zurück, die sie beim Öffnen gelesen haben. War die Maske beim Umbenennen
// offen, legt ihr Speichern die alte Schreibweise wieder an und nimmt dem Titel die neue.
func TestRaster_OffeneMaskeMachtUmbenennenRueckgaengig(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Emil": {"Krimi", "Berlin"}})
	var emil string
	if err := pool.QueryRow(ctx, `SELECT id FROM buecher_titel WHERE titel = 'Emil'`).Scan(&emil); err != nil {
		t.Fatal(err)
	}
	gelesen := woerterAm(t, pool, "Emil") // die Maske öffnet: [Berlin Krimi]

	if _, err := BenenneSchlagwortUm(ctx, pool, ids["Krimi"], "Kriminalroman"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetzeSchlagworte(ctx, pool, emil, gelesen); err != nil { // Maske speichert
		t.Fatal(err)
	}
	if got := woerterAm(t, pool, "Emil"); !slices.Equal(got, []string{"Berlin", "Kriminalroman"}) {
		t.Errorf("BESTÄTIGT: nach dem Speichern der offenen Maske trägt „Emil“ %v statt [Berlin Kriminalroman]", got)
	}
}

// Gegenprobe zu V2: dieselbe offene Maske, aber zusammengeführt statt umbenannt.
func TestRaster_Gegenprobe_OffeneMaskeNachZusammenfuehren(t *testing.T) {
	pool := pgTestPool(t)
	resetSchlagworte(t, pool)
	ctx := context.Background()
	ids := pflegeStand(t, pool, map[string][]string{"Emil": {"Krimi", "Berlin"}, "Kalle": {"Kriminalroman"}})
	var emil string
	if err := pool.QueryRow(ctx, `SELECT id FROM buecher_titel WHERE titel = 'Emil'`).Scan(&emil); err != nil {
		t.Fatal(err)
	}
	gelesen := woerterAm(t, pool, "Emil")
	if _, err := FuehreSchlagworteZusammen(ctx, pool, ids["Krimi"], ids["Kriminalroman"]); err != nil {
		t.Fatal(err)
	}
	if _, err := SetzeSchlagworte(ctx, pool, emil, gelesen); err != nil {
		t.Fatal(err)
	}
	if got := woerterAm(t, pool, "Emil"); !slices.Equal(got, []string{"Berlin", "Kriminalroman"}) {
		t.Errorf("trägt %v", got)
	}
}
