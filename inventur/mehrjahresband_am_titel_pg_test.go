package inventur

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

// Mehrjahresband am Werk (Migration 134, Antwort der Schule vom 22.09.2026, docs/OFFEN.md
// 9.6): Was die Maske über Anlegen und Ändern schreibt, muss an der Tür ankommen, die die
// Fristregel liest — repository.GetCopyByBarcode füllt BookCopy.Mehrjahresband und
// BookCopy.JahrgangBis, und daraus rechnet internal/service/loan_rules.go die Jahre über
// den Stichtag hinaus. Der Einzel-Read (ListBooksByIDs) muss den Schalter ebenfalls liefern,
// sonst zeigt die Maske nach dem Öffnen „aus" und schreibt das beim Speichern zurück
// (Upsert-Blanking-Bugklasse). Rot gesehen am Rückbau beider Schreibpfade (22.09.2026).
func TestMehrjahresband_StehtAmWerkUndErreichtDieFristregel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	const isbn = "978-9-99-318001-1"
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
		t.Fatal(err)
	}
	id, err := repo.CreateBook(ctx, Book{ISBN: isbn, Title: "Mathe 7-9", Author: "Verlag", IstLernmittel: true,
		JahrgangVon: 7, JahrgangBis: 9, Mehrjahresband: true})
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

	lies := func(schritt string) (fristAn bool, fristBis int, maskeAn bool) {
		t.Helper()
		ex, err := repository.NewBookRepository(pool).GetCopyByBarcode(ctx, "B-MJB-1")
		if err != nil || ex == nil {
			t.Fatalf("%s: Exemplar laden: %v", schritt, err)
		}
		buecher, err := repo.ListBooksByIDs(ctx, []string{id})
		if err != nil || len(buecher) != 1 {
			t.Fatalf("%s: Einzel-Read: %v (%d Titel)", schritt, err, len(buecher))
		}
		return ex.Mehrjahresband, ex.JahrgangBis, buecher[0].Mehrjahresband
	}

	if an, bis, maske := lies("nach dem Anlegen"); !an || bis != 9 || !maske {
		t.Errorf("nach dem Anlegen: Fristregel sieht an=%v bis=%d, Maske an=%v — erwartet an, 9, an", an, bis, maske)
	}

	buecher, err := repo.ListBooksByIDs(ctx, []string{id})
	if err != nil || len(buecher) != 1 {
		t.Fatalf("vor dem Ändern lesen: %v (%d Titel)", err, len(buecher))
	}
	geaendert := buecher[0]
	geaendert.JahrgangBis = 10
	if err := repo.UpdateBook(ctx, id, geaendert, nil); err != nil {
		t.Fatalf("Spanne ändern: %v", err)
	}
	if an, bis, _ := lies("nach dem Ändern der Spanne"); !an || bis != 10 {
		t.Errorf("nach dem Ändern der Spanne: Fristregel sieht an=%v bis=%d — erwartet an, 10", an, bis)
	}

	geaendert.Mehrjahresband = false
	if err := repo.UpdateBook(ctx, id, geaendert, nil); err != nil {
		t.Fatalf("Schalter aus: %v", err)
	}
	if an, _, maske := lies("Schalter aus"); an || maske {
		t.Errorf("Schalter aus: Fristregel sieht an=%v, Maske an=%v — erwartet aus, aus", an, maske)
	}

	// Die Datenbank hält die Regel auch für Schreiber, die nicht durch pruefeMehrjahresband
	// gehen: kein Schalter am Bibliotheksbuch, keiner an einer Spanne über einen Jahrgang.
	for name, sql := range map[string]string{
		"Bibliotheksbuch": `UPDATE buecher_titel SET ist_lernmittel = false, mehrjahresband = true WHERE id = $1`,
		"Spanne 7 bis 7":  `UPDATE buecher_titel SET jahrgang_von = 7, jahrgang_bis = 7, mehrjahresband = true WHERE id = $1`,
		"bis unter von":   `UPDATE buecher_titel SET jahrgang_von = 9, jahrgang_bis = 7, mehrjahresband = true WHERE id = $1`,
	} {
		_, err := pool.Exec(ctx, sql, id)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.ConstraintName != "chk_mehrjahresband_spanne" {
			t.Errorf("%s: die Datenbank muss den Schalter abweisen (chk_mehrjahresband_spanne), bekam %v", name, err)
		}
	}
}
