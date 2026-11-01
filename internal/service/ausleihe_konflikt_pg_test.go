package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

// Nach einem abgeprallten Schreiben: Derselbe Leser hat sein Buch schon (der Scanner löste doppelt
// aus), und der Scan ist kein Fehler. Ein anderer Leser ist einer; sonst gäbe die Theke ein Buch
// heraus, das auf jemand anderem steht.
func TestDeuteAusleiheKonflikt(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	dienst := NewAusleihDienst(w.pool, repository.NewLeserAbfragen(w.pool), repository.NewKatalogAbfragen(w.pool),
		repository.NewAusleihAbfragen(w.pool), repository.NewProtokollAbfragen(w.pool))
	deute := func(leser string) error {
		t.Helper()
		tx, err := w.pool.Begin(ctx)
		if err != nil {
			t.Fatalf("Transaktion: %v", err)
		}
		defer db.Verwerfe(ctx, tx)
		return dienst.deuteAusleiheKonflikt(ctx, tx, &repository.Exemplar{ID: w.exemplarID}, &ausleihVorgang{leserID: leser})
	}

	if err := deute(w.anna); !errors.Is(err, ErrKonflikt) || MeldungAus(err) != "Dieses Exemplar konnte nicht verbucht werden, bitte erneut scannen" {
		t.Errorf("ohne offene Ausleihe: %v", err)
	}

	jetzt := time.Now()
	if _, err := w.pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, erfasst_am, rueckgabe_frist)
		VALUES ($1, $2, $3, $4::timestamptz, $4::timestamptz, $4::timestamptz + interval '14 days')`,
		w.exemplarID, w.anna, w.staff, jetzt.Add(-time.Hour)); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}

	var bereits errAusleiheBereitsVorhanden
	if err := deute(w.anna); !errors.As(err, &bereits) {
		t.Errorf("derselbe Leser: %v, erwartet die bestehende Ausleihe", err)
	}
	if err := deute(w.ben); !errors.Is(err, ErrKonflikt) || MeldungAus(err) != "Dieses Exemplar wurde soeben an einem anderen Arbeitsplatz verbucht" {
		t.Errorf("anderer Leser: %v", err)
	}
}
