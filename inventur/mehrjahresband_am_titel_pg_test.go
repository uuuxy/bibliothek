package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Mehrjahresband am Werk (Antwort der Schule vom 22.09.2026, docs/OFFEN.md 9.6): Was die
// Maske über Anlegen und Ändern schreibt, muss an der Tür ankommen, die die Fristregel
// liest — repository.GetCopyByBarcode füllt BookCopy.ZielJahrgang, und daraus rechnet
// internal/service/loan_rules.go die Jahre über den Stichtag hinaus. Der Einzel-Read
// (ListBooksByIDs) muss den Wert ebenfalls liefern, sonst zeigt die Maske nach dem
// Öffnen eine 0 und schreibt sie beim Speichern zurück (Upsert-Blanking-Bugklasse).
// Rot gesehen am Rückbau des Schreibpfads (22.09.2026).
func TestMehrjahresband_StehtAmWerkUndErreichtDieFristregel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	const isbn = "978-9-99-318001-1"
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
		t.Fatal(err)
	}
	id, err := repo.CreateBook(ctx, Book{ISBN: isbn, Title: "Mathe 7-9", Author: "Verlag", IstLernmittel: true, ZielJahrgang: 9})
	if err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-MJB-1')`, id); err != nil {
		t.Fatalf("Exemplar: %v", err)
	}

	lies := func(schritt string) (int, int) {
		t.Helper()
		ex, err := repository.NewBookRepository(pool).GetCopyByBarcode(ctx, "B-MJB-1")
		if err != nil || ex == nil {
			t.Fatalf("%s: Exemplar laden: %v", schritt, err)
		}
		buecher, err := repo.ListBooksByIDs(ctx, []string{id})
		if err != nil || len(buecher) != 1 {
			t.Fatalf("%s: Einzel-Read: %v (%d Titel)", schritt, err, len(buecher))
		}
		return ex.ZielJahrgang, buecher[0].ZielJahrgang
	}

	if frist, maske := lies("nach dem Anlegen"); frist != 9 || maske != 9 {
		t.Errorf("nach dem Anlegen: Fristregel sieht %d, Maske %d — erwartet 9 und 9", frist, maske)
	}

	buecher, err := repo.ListBooksByIDs(ctx, []string{id})
	if err != nil || len(buecher) != 1 {
		t.Fatalf("vor dem Ändern lesen: %v (%d Titel)", err, len(buecher))
	}
	geaendert := buecher[0]
	geaendert.ZielJahrgang = 10
	if err := repo.UpdateBook(ctx, id, geaendert, nil); err != nil {
		t.Fatalf("Titel ändern: %v", err)
	}
	if frist, maske := lies("nach dem Ändern"); frist != 10 || maske != 10 {
		t.Errorf("nach dem Ändern: Fristregel sieht %d, Maske %d — erwartet 10 und 10", frist, maske)
	}

	geaendert.ZielJahrgang = 0
	if err := repo.UpdateBook(ctx, id, geaendert, nil); err != nil {
		t.Fatalf("Zurück auf ein Schuljahr: %v", err)
	}
	if frist, maske := lies("zurück auf ein Schuljahr"); frist != 0 || maske != 0 {
		t.Errorf("zurück auf ein Schuljahr: Fristregel sieht %d, Maske %d — erwartet 0 und 0", frist, maske)
	}
}
