package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Auflage ist ein eigenes Feld am Titel (OFFEN.md 4.18, Stufe 1).
//
// Ein Schulbuch wird nachbestellt, es gibt es aber nur noch in der nächsten Auflage: neue
// ISBN, also ein neuer Titel — zwei Zeilen „Lambacher Schweizer 7" im Katalog. Ohne dieses
// Feld sind die beiden in keiner Liste auseinanderzuhalten; wer eines der beiden an eine
// Klasse ausgibt, weiß nicht, welches.
//
// Bis zum 17.09.2026 gab es das Wort im ganzen System nur als Freitext-Schlüssel in
// erweiterte_eigenschaften, geschrieben von niemandem — und damit auch in keiner Maske.
//
// Der dritte Fall ist die Upsert-Blanking-Bugklasse: Der Listenimport („Sammelkäufe und
// Spenden") schickt nur ISBN und Stückzahl. Er darf eine gepflegte Auflage nicht leeren.
func TestAuflage_StehtAlsFeldAmTitel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	const isbn = "978-9-99-318000-4"
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
		t.Fatal(err)
	}

	id, err := repo.CreateBook(ctx, Book{
		ISBN:    isbn,
		Title:   "Lambacher Schweizer 7",
		Verlag:  "Klett",
		Auflage: "4. Aufl. 2023",
	})
	if err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	lies := func(schritt string) Book {
		t.Helper()
		buecher, err := repo.ListBooksByIDs(ctx, []string{id})
		if err != nil {
			t.Fatalf("%s: lesen: %v", schritt, err)
		}
		if len(buecher) != 1 {
			t.Fatalf("%s: %d Titel gelesen, erwartet 1", schritt, len(buecher))
		}
		return buecher[0]
	}

	if got := lies("nach dem Anlegen").Auflage; got != "4. Aufl. 2023" {
		t.Errorf("nach dem Anlegen: Auflage %q, erwartet %q", got, "4. Aufl. 2023")
	}

	geaendert := lies("vor dem Ändern")
	geaendert.Auflage = "5. Aufl. 2026"
	if err := repo.UpdateBook(ctx, id, geaendert, nil); err != nil {
		t.Fatalf("Titel ändern: %v", err)
	}
	if got := lies("nach dem Ändern").Auflage; got != "5. Aufl. 2026" {
		t.Errorf("nach dem Ändern: Auflage %q, erwartet %q", got, "5. Aufl. 2026")
	}

	if _, err := repo.UpsertBooksBatch(ctx, []Book{{ISBN: isbn, Stock: 2}}); err != nil {
		t.Fatalf("Listenimport: %v", err)
	}
	if got := lies("nach dem Listenimport").Auflage; got != "5. Aufl. 2026" {
		t.Errorf("nach dem Listenimport: Auflage %q, erwartet %q — eine Datei ohne Auflagen-Spalte "+
			"darf die gepflegte Angabe nicht leeren (Upsert-Blanking)", got, "5. Aufl. 2026")
	}
}
