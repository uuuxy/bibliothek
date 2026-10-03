package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Titel-Verwaltung und Lernmittel-Liste suchen am Server und vergleichen eine getippte ISBN
// als Teilstring. Die zehnstellige vom Titelblatt und die dreizehnstellige, unter der die
// Datenbank den Titel führt (Migration 157), enden auf verschiedene Prüfzeichen: Ohne den
// Vergleich mit der Normalform des Suchtexts fände die getippte zehnstellige den Titel nicht.
func TestSuche_FindetDenTitelUeberDieZehnstelligeISBN(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	const titel, zehn, dreizehn, barcode = "Advanced Organic Chemistry", "0306406152", "9780306406157", "B-ISBN-SUCHE-INV-1"

	raeume := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE barcode_id = $1`, barcode); err != nil {
			t.Errorf("aufräumen: Probe-Exemplar löschen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = ANY($1)`, []string{zehn, dreizehn}); err != nil {
			t.Errorf("aufräumen: Probe-Titel löschen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)

	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn, ist_lernmittel) VALUES ($1, $2, true) RETURNING id::text`,
		titel, zehn).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`, id, barcode); err != nil {
		t.Fatal(err)
	}

	for _, suchtext := range []string{zehn, "0-306-40615-2", dreizehn, "978-0-306-40615-7"} {
		liste, err := repo.ListBooks(ctx, "", nil, suchtext, false)
		if err != nil {
			t.Fatalf("ListBooks(%q): %v", suchtext, err)
		}
		if len(liste) != 1 || liste[0].ID != id {
			t.Errorf("Titel-Verwaltung, Suche %q: %d Treffer, erwartet den Titel %q", suchtext, len(liste), titel)
		}
		lernmittel, err := repo.GetLernmittelTitel(ctx, "", true, LernmittelFilter{Suche: suchtext})
		if err != nil {
			t.Fatalf("GetLernmittelTitel(%q): %v", suchtext, err)
		}
		if len(lernmittel) != 1 || lernmittel[0].ID != id {
			t.Errorf("Lernmittel, Suche %q: %d Treffer, erwartet den Titel %q", suchtext, len(lernmittel), titel)
		}
	}

	// Gegenprobe: Dieselben Ziffern mit falschem Prüfzeichen sind eine andere Nummer.
	if liste, err := repo.ListBooks(ctx, "", nil, "0306406153", false); err != nil || len(liste) != 0 {
		t.Errorf("Titel-Verwaltung mit falschem Prüfzeichen: %d Treffer, %v — erwartet keinen", len(liste), err)
	}
}
