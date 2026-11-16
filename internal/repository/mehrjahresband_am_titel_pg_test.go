package repository

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgconn"
)

// Mehrjahresband am Werk: Was die Maske über Anlegen und Ändern schreibt, muss an der Tür
// ankommen, die die Regel der Frist liest; FindeExemplar füllt Exemplar.Mehrjahresband
// und Exemplar.JahrgangBis, und daraus rechnet internal/service/frist.go die Jahre über den
// Stichtag hinaus. Das Lesen eines einzelnen Titels (LadeTitelNachIDs) muss den Schalter
// ebenfalls liefern, sonst zeigt die Maske nach dem Öffnen „aus" und schreibt das beim Speichern
// zurück. Rot gesehen am Rückbau beider Schreibwege.
func TestMehrjahresband_StehtAmWerkUndErreichtDieFristregel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	abfragen := NewTitelAbfragen(pool)

	const isbn = "978-9-99-318001-1"
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
		t.Fatal(err)
	}
	id, err := abfragen.LegeTitelAn(ctx, KatalogTitel{ISBN: isbn, Titel: "Mathe 7-9", Autor: "Verlag", IstLernmittel: true,
		JahrgangVon: 7, JahrgangBis: 9, Mehrjahresband: true})
	if err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	testdaten.LegeExemplarAn(t, pool, id, "B-MJB-1", true, "")

	lies := func(schritt string) (fristAn bool, fristBis int, maskeAn bool) {
		t.Helper()
		ex, err := NewKatalogAbfragen(pool).FindeExemplar(ctx, "B-MJB-1")
		if err != nil || ex == nil {
			t.Fatalf("%s: Exemplar laden: %v", schritt, err)
		}
		buecher, err := abfragen.LadeTitelNachIDs(ctx, []string{id})
		if err != nil || len(buecher) != 1 {
			t.Fatalf("%s: Einzel-Read: %v (%d Titel)", schritt, err, len(buecher))
		}
		return ex.Mehrjahresband, ex.JahrgangBis, buecher[0].Mehrjahresband
	}

	if an, bis, maske := lies("nach dem Anlegen"); !an || bis != 9 || !maske {
		t.Errorf("nach dem Anlegen: Fristregel sieht an=%v bis=%d, Maske an=%v — erwartet an, 9, an", an, bis, maske)
	}

	buecher, err := abfragen.LadeTitelNachIDs(ctx, []string{id})
	if err != nil || len(buecher) != 1 {
		t.Fatalf("vor dem Ändern lesen: %v (%d Titel)", err, len(buecher))
	}
	geaendert := buecher[0]
	geaendert.JahrgangBis = 10
	if err := abfragen.AendereTitel(ctx, id, geaendert, []string{"jahrgangBis"}, nil); err != nil {
		t.Fatalf("Spanne ändern: %v", err)
	}
	if an, bis, _ := lies("nach dem Ändern der Spanne"); !an || bis != 10 {
		t.Errorf("nach dem Ändern der Spanne: Fristregel sieht an=%v bis=%d — erwartet an, 10", an, bis)
	}

	geaendert.Mehrjahresband = false
	if err := abfragen.AendereTitel(ctx, id, geaendert, []string{"mehrjahresband"}, nil); err != nil {
		t.Fatalf("Schalter aus: %v", err)
	}
	if an, _, maske := lies("Schalter aus"); an || maske {
		t.Errorf("Schalter aus: Fristregel sieht an=%v, Maske an=%v — erwartet aus, aus", an, maske)
	}

	// Die Datenbank hält die Regel auch für Schreiber, die nicht durch pruefeMehrjahresband
	// gehen: kein Schalter am Bibliotheksbuch, keiner an einer Spanne über einen Jahrgang,
	// keiner ohne Spanne. „bis" unter „von" weist schon die Regel der Spanne ab.
	for _, fall := range []struct{ name, sql, regel string }{
		{"Bibliotheksbuch", `UPDATE buecher_titel SET ist_lernmittel = false, mehrjahresband = true WHERE id = $1`, "chk_mehrjahresband_spanne"},
		{"Spanne 7 bis 7", `UPDATE buecher_titel SET jahrgang_von = 7, jahrgang_bis = 7, mehrjahresband = true WHERE id = $1`, "chk_mehrjahresband_spanne"},
		{"ohne Spanne", `UPDATE buecher_titel SET jahrgang_von = NULL, jahrgang_bis = NULL, mehrjahresband = true WHERE id = $1`, "chk_mehrjahresband_spanne"},
		{"bis unter von", `UPDATE buecher_titel SET jahrgang_von = 9, jahrgang_bis = 7, mehrjahresband = true WHERE id = $1`, "chk_jahrgang_spanne"},
	} {
		_, err := pool.Exec(ctx, fall.sql, id)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.ConstraintName != fall.regel {
			t.Errorf("%s: die Datenbank muss den Schalter abweisen (%s), bekam %v", fall.name, fall.regel, err)
		}
	}
}
