package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Seit 01.09.2026 über internal/pgtest statt eines eigenen Harness: Der frühere
// ladeSchemaPool hielt den Advisory-Lock 0x42DB0001 SELBST (eigene Verbindung,
// eigener Schema-Reset je Test). Sobald ein zweiter Test im selben Binary den Lock
// über pgtest genommen hätte — der ihn bewusst bis Prozessende hält —, hätte dieser
// Test für immer auf sein eigenes Binary gewartet; genau dieser Selbst-Deadlock hat
// am 01.09. über order_search die ganze Suite in 10-Minuten-Timeouts gezogen
// (de42b820). Der eigene DROP SCHEMA mitten im Binary hätte zudem jedem
// nachfolgenden Test die Tabellen weggezogen. pgtest teilt das Schema mit allen
// Tests des Binaries — deshalb räumt jeder Test seine Zeilen selbst ab und prüft
// nur ID-gescoped.

// TestListBooksSchlank_KeineSchwerenFelder belegt die Payload-Verschlankung von
// GET /api/books: Die Katalogliste liefert die schwerste Spalte (erweiterte_eigenschaften)
// nicht mit — die Listenansicht und ihre Suche im Browser brauchen sie nicht. Der
// Einzel-Read (ListBooksByIDs, für Detail und Bearbeiten) liefert sie vollständig.
func TestListBooksSchlank_KeineSchwerenFelder(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

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
	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'SCHLANK-1')`, id); err != nil {
		t.Fatalf("Exemplar: %v", err)
	}

	// Listenansicht: Titel da, aber das schwere Feld leer.
	liste, err := repo.ListBooks(ctx, "", nil, "", false)
	if err != nil {
		t.Fatalf("ListBooks: %v", err)
	}
	var gefunden *Book
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
	if gefunden.Title != "Schlank-Test" || gefunden.Author != "Autor" {
		t.Errorf("Stammdaten der Listenansicht unvollständig: %+v", gefunden)
	}

	// Detail-Read: das schwere Feld vollständig.
	detail, err := repo.ListBooksByIDs(ctx, []string{id})
	if err != nil {
		t.Fatalf("ListBooksByIDs: %v", err)
	}
	if len(detail) != 1 {
		t.Fatalf("Detail-Read lieferte %d statt 1", len(detail))
	}
	if detail[0].ErweiterteEigenschaften["regal"] != "A1" {
		t.Errorf("Detail-Read muss erweiterte_eigenschaften liefern, war %v", detail[0].ErweiterteEigenschaften)
	}
}
