package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Aussondern und Ausleihe im selben Augenblick: Die Theke hat das Exemplar gesperrt und die
// Ausleihe geschrieben, aber noch nicht festgeschrieben. Ein Aussondern, das dann wartet, muss
// die Ausleihe danach sehen; sonst ist das Buch ausgesondert und zugleich verliehen.

// verleiheOhneFestschreiben schreibt die Ausleihe so, wie es der Scan an der Theke tut (Sperre
// der Exemplarzeile, dann Ausleihe und Stempel), und lässt die Transaktion offen.
func verleiheOhneFestschreiben(t *testing.T, pool *pgxpool.Pool, f stempelFall) pgx.Tx {
	t.Helper()
	ctx := context.Background()
	theke := beginne(t, pool)
	if _, err := SperreExemplarzeile(ctx, theke, f.exemplarID); err != nil {
		t.Fatalf("SperreExemplarzeile: %v", err)
	}
	if _, err := LegeAusleiheAn(ctx, theke, AusleiheNeu{
		ExemplarID: f.exemplarID, LeserID: f.schuelerID, BearbeiterID: f.bearbeiterID,
		RueckgabeFrist: time.Now().AddDate(0, 0, 21),
	}); err != nil {
		t.Fatalf("LegeAusleiheAn: %v", err)
	}
	return theke
}

// ausgesondertUndOffen liest den Endzustand des Exemplars.
func ausgesondertUndOffen(t *testing.T, pool *pgxpool.Pool, exemplarID string) (ausgesondert bool, offen int) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), `
		SELECT e.ist_ausgesondert,
		       (SELECT count(*) FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
		FROM buecher_exemplare e WHERE e.id = $1`, exemplarID).Scan(&ausgesondert, &offen); err != nil {
		t.Fatalf("Endzustand lesen: %v", err)
	}
	return ausgesondert, offen
}

func TestSondereExemplarAus_SiehtDieAusleiheDerTheke(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	f := stempelAufbau(t, pool)

	theke := verleiheOhneFestschreiben(t, pool, f)
	defer db.Verwerfe(ctx, theke)
	fertig := make(chan error, 1)
	go func() {
		fertig <- NewKatalogAbfragen(pool).SondereExemplarAus(ctx, f.exemplarID, "AUSSORTIERT", f.bearbeiterID)
	}()
	ueberschneidung(t, pool, theke, fertig)
	if err := theke.Commit(ctx); err != nil {
		t.Fatalf("Theke festschreiben: %v", err)
	}

	if err := <-fertig; !errors.Is(err, ErrExemplarNochVerliehen) {
		t.Errorf("SondereExemplarAus = %v, erwartet ErrExemplarNochVerliehen", err)
	}
	if ausgesondert, offen := ausgesondertUndOffen(t, pool, f.exemplarID); ausgesondert || offen != 1 {
		t.Errorf("Endzustand: ausgesondert %v, offene Ausleihen %d, erwartet false und 1", ausgesondert, offen)
	}
}

// Ein Titel mit einem verliehenen Exemplar, das niemand zum Mitlöschen bestätigt hat, bleibt;
// seine Exemplare im Haus sondert das Löschen mit dem Grund aus. Eines davon verleiht die Theke
// gerade.
func TestLoescheTitel_SondertEbenVerliehenesExemplarNichtAus(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	f := stempelAufbau(t, pool)
	var titelID, anderesID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT titel_id, barcode_id || '-2', true FROM buecher_exemplare WHERE id = $1
		RETURNING titel_id, id`, f.exemplarID).Scan(&titelID, &anderesID); err != nil {
		t.Fatalf("zweites Exemplar anlegen: %v", err)
	}
	vorher := beginne(t, pool)
	defer db.Verwerfe(ctx, vorher)
	if _, err := LegeAusleiheAn(ctx, vorher, AusleiheNeu{
		ExemplarID: anderesID, LeserID: f.schuelerID, BearbeiterID: f.bearbeiterID,
		RueckgabeFrist: time.Now().AddDate(0, 0, 21),
	}); err != nil {
		t.Fatalf("zweites Exemplar verleihen: %v", err)
	}
	if err := vorher.Commit(ctx); err != nil {
		t.Fatalf("zweites Exemplar verleihen: %v", err)
	}

	theke := verleiheOhneFestschreiben(t, pool, f)
	defer db.Verwerfe(ctx, theke)
	fertig := make(chan error, 1)
	go func() {
		_, err := NewTitelAbfragen(pool).LoescheTitel(ctx, TitelLoeschauftrag{
			TitelIDs: []string{titelID}, Grund: "AUSSORTIERT", BearbeiterID: f.bearbeiterID,
		})
		fertig <- err
	}()
	ueberschneidung(t, pool, theke, fertig)
	if err := theke.Commit(ctx); err != nil {
		t.Fatalf("Theke festschreiben: %v", err)
	}

	if err := <-fertig; err != nil {
		t.Fatalf("LoescheTitel: %v", err)
	}
	if ausgesondert, offen := ausgesondertUndOffen(t, pool, f.exemplarID); ausgesondert || offen != 1 {
		t.Errorf("Endzustand: ausgesondert %v, offene Ausleihen %d, erwartet false und 1", ausgesondert, offen)
	}
}

// Die Bestandskorrektur sondert zuerst Exemplare aus, die nicht verliehen sind. Von zweien
// verleiht die Theke gerade das erste; ausgesondert werden muss das andere.
func TestGleicheBestandAb_SondertEbenVerliehenesExemplarNichtAus(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	f := stempelAufbau(t, pool)
	var titelID, anderesID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT titel_id, barcode_id || '-2', true FROM buecher_exemplare WHERE id = $1
		RETURNING titel_id, id`, f.exemplarID).Scan(&titelID, &anderesID); err != nil {
		t.Fatalf("zweites Exemplar anlegen: %v", err)
	}

	theke := verleiheOhneFestschreiben(t, pool, f)
	defer db.Verwerfe(ctx, theke)
	fertig := make(chan error, 1)
	go func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			fertig <- err
			return
		}
		defer db.Verwerfe(ctx, tx)
		if err := NewTitelAbfragen(pool).gleicheBestandAb(ctx, tx, titelID, 1, f.bearbeiterID); err != nil {
			fertig <- err
			return
		}
		fertig <- tx.Commit(ctx)
	}()
	ueberschneidung(t, pool, theke, fertig)
	if err := theke.Commit(ctx); err != nil {
		t.Fatalf("Theke festschreiben: %v", err)
	}

	if err := <-fertig; err != nil {
		t.Fatalf("gleicheBestandAb: %v", err)
	}
	if ausgesondert, offen := ausgesondertUndOffen(t, pool, f.exemplarID); ausgesondert || offen != 1 {
		t.Errorf("verliehenes Exemplar: ausgesondert %v, offene Ausleihen %d, erwartet false und 1", ausgesondert, offen)
	}
	if ausgesondert, _ := ausgesondertUndOffen(t, pool, anderesID); !ausgesondert {
		t.Errorf("freies Exemplar: ausgesondert %v, erwartet true", ausgesondert)
	}
}
