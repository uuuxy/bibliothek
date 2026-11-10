package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Scan und Aussondern im selben Augenblick: Der Scan hat das Exemplar als ausleihbar gelesen,
// bevor ein anderer Platz es aussondert oder der Abschluss einer Inventur es als Verlust bucht.
// Das Buch darf danach nicht ausgesondert und zugleich verliehen sein.

type aussonderWelt struct {
	pool                  *pgxpool.Pool
	bearbeiter, schueler  string
	signatur, exemplarID  string
	barcode, inventurLauf string
}

// newAussonderWelt legt ein ausleihbares Exemplar unter einer eigenen Signatur an und einen
// Inventurlauf über diese Signatur, der nach dem Exemplar beginnt.
func newAussonderWelt(t *testing.T) *aussonderWelt {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	w := &aussonderWelt{pool: pool, signatur: "AUSS" + suffix, barcode: "B-AUSS-" + suffix}
	w.bearbeiter = testdaten.LegeBearbeiterAn(t, pool)
	w.schueler = testdaten.LegeSchuelerAn(t, pool, "S-AUSS-"+suffix, "Nora", "07H1")
	titel := testdaten.LegeTitelMitSignaturAn(t, pool, "Aussonder-Band", w.signatur, 0)
	w.exemplarID = testdaten.LegeExemplarAn(t, pool, titel, w.barcode, true, "")
	// Ein Exemplar, das nach dem Beginn des Laufs angelegt wurde, bucht der Abschluss nicht.
	if _, err := pool.Exec(ctx, `SELECT pg_sleep(0.02)`); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO inventur_sessions (scope_type, scope_signatur, gestartet_von)
		VALUES ('signature', $1, $2) RETURNING id`, w.signatur, w.bearbeiter).Scan(&w.inventurLauf); err != nil {
		t.Fatalf("Inventurlauf anlegen: %v", err)
	}
	return w
}

// scanne bucht das Exemplar wie der Scan an der Theke in einer eigenen Goroutine; gelesen wird
// es vorher, mit dem Stand, der jetzt festgeschrieben ist.
func (w *aussonderWelt) scanne(t *testing.T) chan error {
	t.Helper()
	ctx := context.Background()
	katalog := repository.NewKatalogAbfragen(w.pool)
	dienst := NewAusleihDienst(w.pool, repository.NewLeserAbfragen(w.pool), katalog,
		repository.NewAusleihAbfragen(w.pool), repository.NewProtokollAbfragen(w.pool))
	ex, err := katalog.FindeExemplar(ctx, w.barcode)
	if err != nil || ex == nil {
		t.Fatalf("FindeExemplar(%q) = %v, %v", w.barcode, ex, err)
	}
	fertig := make(chan error, 1)
	go func() {
		_, err := dienst.BucheAmLeser(ctx, ex, &w.schueler, w.bearbeiter, false)
		fertig <- err
	}()
	return fertig
}

func (w *aussonderWelt) endzustand(t *testing.T) (ausgesondert bool, offen int) {
	t.Helper()
	if err := w.pool.QueryRow(context.Background(), `
		SELECT e.ist_ausgesondert,
		       (SELECT count(*) FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
		FROM buecher_exemplare e WHERE e.id = $1`, w.exemplarID).Scan(&ausgesondert, &offen); err != nil {
		t.Fatalf("Endzustand lesen: %v", err)
	}
	return ausgesondert, offen
}

// pruefeAbgewiesen verlangt, dass der Scan nach dem Warten abgewiesen hat und das Exemplar
// ausgesondert und frei ist.
func (w *aussonderWelt) pruefeAbgewiesen(t *testing.T, sperrer pgx.Tx, fertig chan error) {
	t.Helper()
	warteAufWartende(t, w.pool, sperrer, 1)
	if err := sperrer.Commit(context.Background()); err != nil {
		t.Fatalf("Aussondern festschreiben: %v", err)
	}
	if err := <-fertig; !errors.Is(err, ErrUngueltigerZustand) {
		t.Errorf("BucheAmLeser = %v, erwartet ErrUngueltigerZustand", err)
	}
	if ausgesondert, offen := w.endzustand(t); !ausgesondert || offen != 0 {
		t.Errorf("Endzustand: ausgesondert %v, offene Ausleihen %d, erwartet true und 0", ausgesondert, offen)
	}
}

func TestBucheAmLeser_EbenAusgesondertesExemplar(t *testing.T) {
	w := newAussonderWelt(t)
	ctx := context.Background()
	aussondern, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Verwerfe(ctx, aussondern)
	if _, err := aussondern.Exec(ctx, `UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'AUSSORTIERT'
		WHERE id = $1`, w.exemplarID); err != nil {
		t.Fatalf("aussondern: %v", err)
	}
	w.pruefeAbgewiesen(t, aussondern, w.scanne(t))
}

func TestBucheAmLeser_EbenAlsVerlustGebuchtesExemplar(t *testing.T) {
	w := newAussonderWelt(t)
	ctx := context.Background()
	abschluss, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Verwerfe(ctx, abschluss)
	verloren, err := repository.NewInventurAbfragen(abschluss).SchliesseLaufAb(ctx, w.inventurLauf,
		repository.InventurBereich{Signatur: &w.signatur}, w.bearbeiter)
	if err != nil || verloren != 1 {
		t.Fatalf("SchliesseLaufAb = %d, %v, erwartet 1 Verlust", verloren, err)
	}
	w.pruefeAbgewiesen(t, abschluss, w.scanne(t))
}

// Die andere Reihenfolge: Die Theke verleiht zuerst, der Abschluss wartet und lässt das Exemplar
// danach stehen, weil seine Bedingung den Stempel der Exemplarzeile liest.
func TestSchliesseLaufAb_LaesstEbenVerliehenesExemplarStehen(t *testing.T) {
	w := newAussonderWelt(t)
	ctx := context.Background()
	theke, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Verwerfe(ctx, theke)
	if _, err := repository.SperreExemplarzeile(ctx, theke, w.exemplarID); err != nil {
		t.Fatalf("SperreExemplarzeile: %v", err)
	}
	if _, err := repository.LegeAusleiheAn(ctx, theke, repository.AusleiheNeu{
		ExemplarID: w.exemplarID, LeserID: w.schueler, BearbeiterID: w.bearbeiter,
		RueckgabeFrist: time.Now().AddDate(0, 0, 21),
	}); err != nil {
		t.Fatalf("LegeAusleiheAn: %v", err)
	}

	type ergebnis struct {
		verloren int
		err      error
	}
	fertig := make(chan ergebnis, 1)
	go func() {
		tx, err := w.pool.Begin(ctx)
		if err != nil {
			fertig <- ergebnis{0, err}
			return
		}
		defer db.Verwerfe(ctx, tx)
		n, err := repository.NewInventurAbfragen(tx).SchliesseLaufAb(ctx, w.inventurLauf,
			repository.InventurBereich{Signatur: &w.signatur}, w.bearbeiter)
		if err == nil {
			err = tx.Commit(ctx)
		}
		fertig <- ergebnis{n, err}
	}()
	warteAufWartende(t, w.pool, theke, 1)
	if err := theke.Commit(ctx); err != nil {
		t.Fatalf("Theke festschreiben: %v", err)
	}
	if e := <-fertig; e.err != nil || e.verloren != 0 {
		t.Errorf("SchliesseLaufAb = %d, %v, erwartet 0 Verluste", e.verloren, e.err)
	}
	if ausgesondert, offen := w.endzustand(t); ausgesondert || offen != 1 {
		t.Errorf("Endzustand: ausgesondert %v, offene Ausleihen %d, erwartet false und 1", ausgesondert, offen)
	}
}
