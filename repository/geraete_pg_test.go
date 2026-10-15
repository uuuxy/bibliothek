package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Die Verwaltung der Geräte am echten Postgres: Anlegen mit dem benannten Fehler bei
// doppeltem Barcode, die Liste mit dem, der das Gerät gerade hat (Schüler oder Lehrkraft), und
// der Schalter für defekt.
func TestGeraeteVerwaltung_AnlegenListeStatus(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewGeraeteAbfragen(pool)

	id, err := repo.LegeAn(ctx, "iPad 9. Gen", nil, "G-E2E-IPAD-1", "Ladekabel, Stift", "")
	if err != nil {
		t.Fatalf("Anlegen: %v", err)
	}

	// (1) Ein doppelter Barcode ergibt den benannten Fehler und nicht den rohen 23505.
	if _, err := repo.LegeAn(ctx, "Zweitgerät", nil, "G-E2E-IPAD-1", "", ""); !errors.Is(err, ErrGeraetBarcodeVergeben) {
		t.Fatalf("Duplikat: erwartet ErrGeraetBarcodeVergeben, bekam %v", err)
	}

	// (2) Frisch angelegt: im Schrank (kein Ausleiher).
	g := geraetAusListe(t, repo, id)
	if g.AusgeliehenAn != nil || !g.IstAusleihbar || g.Zubehoer != "Ladekabel, Stift" {
		t.Fatalf("frisches Gerät: %+v", g)
	}

	// (3) Ausleihe an einen Schüler → die Liste nennt ihn mit Klasse.
	schueler := legeSchuelerAn(t, pool, "S-GER-1", "Nio", "6c")
	bearbeiter := legeEigenenBearbeiterAn(t, pool, "GER-B")
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (geraet_id, schueler_id, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, now() + interval '14 days', $3)`, id, schueler, bearbeiter); err != nil {
		t.Fatalf("Ausleihe seeden: %v", err)
	}
	g = geraetAusListe(t, repo, id)
	if g.AusgeliehenAn == nil || *g.AusgeliehenAn != "Nio Test (06C)" {
		t.Fatalf("Ausleiher fehlt/falsch: %+v", g.AusgeliehenAn)
	}

	// (4) Defekt-Schalter + Stammdaten-Pflege.
	notiz := "Display-Kratzer"
	defekt := false
	modell, zubehoer := "iPad 9. Gen (Leihgerät)", "Ladekabel"
	if err := repo.Aendere(ctx, id, GeraetAenderung{
		Modellname: &modell, Zubehoer: &zubehoer, ZustandNotiz: &notiz, IstAusleihbar: &defekt}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	g = geraetAusListe(t, repo, id)
	if g.IstAusleihbar || g.Modellname != "iPad 9. Gen (Leihgerät)" || g.Zubehoer != "Ladekabel" {
		t.Fatalf("Update nicht angekommen: %+v", g)
	}

	// (5) Unbekannte ID → pgx.ErrNoRows für das 404-Mapping.
	ausleihbar := true
	if err := repo.Aendere(ctx, "00000000-0000-0000-0000-000000000000", GeraetAenderung{IstAusleihbar: &ausleihbar}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("unbekannte ID: erwartet ErrNoRows, bekam %v", err)
	}
}

// geraetAusListe holt ein Gerät über die Verwaltungs-Liste — genau den Weg, den
// die Oberfläche nimmt.
func geraetAusListe(t *testing.T, repo GeraeteAbfragen, id string) GeraetMitStatus {
	t.Helper()
	liste, err := repo.LadeAlle(context.Background())
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	for _, g := range liste {
		if g.ID == id {
			return g
		}
	}
	t.Fatalf("Gerät %s nicht in der Liste", id)
	return GeraetMitStatus{}
}
