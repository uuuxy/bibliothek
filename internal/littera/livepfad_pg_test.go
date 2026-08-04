package littera

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// TestUebernommeneDatenUeberDenLivePfad liest die geschriebenen Zeilen mit dem CODE DER
// ANWENDUNG zurück, nicht mit eigenem SQL.
//
// Der Grund ist eine Fehlerklasse, die dieses Projekt schon zweimal getroffen hat: Ein
// Import schreibt sauber, die Zählungen stimmen, und trotzdem fällt die Oberfläche mit
// „cannot scan NULL" um — weil eine nullbare Spalte in der Anwendung in einem
// nicht-nullbaren Go-Typ landet. Der Schreibpfad lässt bearbeiter_id und
// rueckgabe_bearbeiter_id leer (bei einer Altbestandsübernahme gibt es niemanden, der
// ausgegeben hat) und setzt schueler_id ODER ausleiher_benutzer_id, nie beide. Genau die
// Kombination muss scanLoan aushalten.
//
// Eigenes SELECT im Test würde das nie zeigen: Es scheitert nicht am NULL, sondern an
// der Struktur, in die die Anwendung hineinliest.
func TestUebernommeneDatenUeberDenLivePfad(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)
	ctx := context.Background()

	ab := bestand(titel("1", "Ausgeliehen an Schüler", ""), titel("2", "Ausgeliehen an Lehrkraft", ""))
	ab.Leser = []Leser{
		leser("S1", "101", "07H1", ArtSchueler),
		leser("L1", "201", "", ArtLehrkraft),
	}
	ab.Ausleihen = []Ausleihe{
		ausleihe("A1", "E1", "S1", 0),
		ausleihe("A2", "E2", "L1", 1),
	}
	bestandBericht, personenBericht := ausleihWelt(t, s, ab)
	if _, err := s.SchreibeAusleihen(ctx, ab, bestandBericht, personenBericht); err != nil {
		t.Fatalf("SchreibeAusleihen: %v", err)
	}

	loans := repository.NewLoanRepository(pool)
	for littera, name := range map[string]string{"E1": "Schülerausleihe", "E2": "Lehrerausleihe"} {
		exemplarID := bestandBericht.ExemplarIDs[littera]
		if exemplarID == "" {
			t.Fatalf("%s: Exemplar %s wurde nicht geschrieben", name, littera)
		}
		loan, err := loans.GetActiveLoanByCopyID(ctx, exemplarID)
		if err != nil {
			t.Fatalf("%s: die Anwendung kann die übernommene Ausleihe nicht lesen: %v", name, err)
		}
		if loan == nil {
			t.Fatalf("%s: die Anwendung findet die übernommene Ausleihe nicht", name)
		}
		if loan.BearbeiterID != nil {
			t.Errorf("%s: bearbeiter_id soll leer bleiben, gelesen: %v", name, *loan.BearbeiterID)
		}
		if (loan.SchuelerID == nil) == (loan.AusleiherBenutzerID == nil) {
			t.Errorf("%s: genau eine Entleiher-Spalte muss gesetzt sein", name)
		}
	}

	// Und der Katalogpfad: Ein Exemplar, das die Anwendung nicht über seinen Barcode
	// findet, ist an der Theke wertlos — egal wie sauber es in der Tabelle steht.
	books := repository.NewBookRepository(pool)
	kopie, err := books.GetCopyByBarcode(ctx, etikett(t, "101"))
	if err != nil {
		t.Fatalf("die Anwendung kann das übernommene Exemplar nicht über den Barcode lesen: %v", err)
	}
	if kopie == nil {
		t.Fatal("die Anwendung findet das übernommene Exemplar nicht über seinen Littera-Barcode")
	}
}
