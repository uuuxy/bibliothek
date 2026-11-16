package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Die Datenbank kommt aus internal/pgtest. pgtest hält die Sperre der Datenbank bis zum Ende des
// Prozesses; ein Test mit eigener Sperre wartete ewig auf sie, und ein eigenes DROP SCHEMA mitten
// im Testprogramm zöge den folgenden Tests die Tabellen weg. pgtest teilt das Schema mit allen
// Tests des Programms; deshalb räumt jeder Test seine Zeilen selbst ab und prüft nur an seinen
// eigenen Kennungen.

// TestLadeTitelListe_SchlankOhneSchwereFelder belegt die Payload-Verschlankung von
// GET /api/books: Die Katalogliste liefert die schwerste Spalte (erweiterte_eigenschaften)
// nicht mit — die Listenansicht und ihre Suche im Browser brauchen sie nicht. Der
// Einzel-Read (LadeTitelNachIDs, für Detail und Bearbeiten) liefert sie vollständig.
func TestLadeTitelListe_SchlankOhneSchwereFelder(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	abfragen := NewTitelAbfragen(pool)

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, erweiterte_eigenschaften)
		VALUES ('Schlank-Test', 'Autor', '{"regal":"A1","notiz":"vertraulich"}'::jsonb)
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	// Exemplar hängt per ON DELETE CASCADE am Titel — eine Zeile räumt beides ab.
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})
	testdaten.LegeExemplarAn(t, pool, id, "SCHLANK-1", true, "")

	// Listenansicht: Titel da, aber das schwere Feld leer.
	liste, err := abfragen.LadeTitelListe(ctx, "", "", false)
	if err != nil {
		t.Fatalf("LadeTitelListe: %v", err)
	}
	var gefunden *KatalogTitel
	for i := range liste {
		if liste[i].ID == id {
			gefunden = &liste[i]
		}
	}
	if gefunden == nil {
		t.Fatal("Titel fehlt in der Katalogliste")
	}
	if len(gefunden.ErweiterteEigenschaften) != 0 {
		t.Errorf("Listenansicht schleppt erweiterte_eigenschaften mit: %v", gefunden.ErweiterteEigenschaften)
	}
	// Stammdaten müssen aber vollständig da sein (Suche/Karte hängen daran).
	if gefunden.Titel != "Schlank-Test" || gefunden.Autor != "Autor" {
		t.Errorf("Stammdaten der Listenansicht unvollständig: %+v", gefunden)
	}

	// Detail-Read: das schwere Feld vollständig.
	detail, err := abfragen.LadeTitelNachIDs(ctx, []string{id})
	if err != nil {
		t.Fatalf("LadeTitelNachIDs: %v", err)
	}
	if len(detail) != 1 {
		t.Fatalf("Detail-Read lieferte %d statt 1", len(detail))
	}
	if detail[0].ErweiterteEigenschaften["regal"] != "A1" {
		t.Errorf("Detail-Read muss erweiterte_eigenschaften liefern, war %v", detail[0].ErweiterteEigenschaften)
	}
}
