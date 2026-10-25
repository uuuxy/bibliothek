package repository_test

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Ein besetztes Kürzel darf die Transaktion eines Imports nicht abbrechen: Die Importe
// registrieren ihre Fächer in der eigenen Transaktion, und ein Fehler der Datenbank ließe dort
// keinen zweiten Versuch mit Endung mehr zu (25P02). Bestands-Import und Littera-XML verlören
// die ganze Datei, die Littera-Übernahme jeden Titel des Fachs.
func TestStelleFaecherSicher_KuerzelBesetztInDerTransaktion(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const vorhanden, neu = "Kollisionsprobe vorhanden", "Kollisions probe"

	var ids []string
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM systematik_kategorien WHERE id = ANY($1::uuid[])`, ids); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})
	var id string
	// Das Kürzel ist genau der Kandidat des neuen Fachs: dessen Bezeichnung ohne Leerzeichen.
	if err := pool.QueryRow(ctx, `INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
		VALUES ('Kollisionsprobe', $1) RETURNING id`, vorhanden).Scan(&id); err != nil {
		t.Fatalf("vorhandenes Fach anlegen: %v", err)
	}
	ids = append(ids, id)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Verwerfe(ctx, tx)

	got, err := repository.StelleFaecherSicher(ctx, tx, []string{neu})
	if err != nil {
		t.Fatalf("repository.StelleFaecherSicher(tx, %q) = Fehler %v, erwartet das angelegte Fach", neu, err)
	}
	if got[neu] != neu {
		t.Errorf("repository.StelleFaecherSicher(tx, %q)[%q] = %q, erwartet %q", neu, neu, got[neu], neu)
	}
	// Die Transaktion bleibt brauchbar: Der Import schreibt danach weiter und schließt ab.
	var kuerzel string
	if err := tx.QueryRow(ctx, `SELECT id, kuerzel FROM systematik_kategorien WHERE bezeichnung = $1`, neu).
		Scan(&id, &kuerzel); err != nil {
		t.Fatalf("neues Fach in der Transaktion lesen: %v", err)
	}
	ids = append(ids, id)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit nach dem Anlegen: %v", err)
	}
	if !strings.HasPrefix(kuerzel, "Kollisionsprobe~") {
		t.Errorf("Kürzel des neuen Fachs = %q, erwartet „Kollisionsprobe~“ mit Endung", kuerzel)
	}
}
