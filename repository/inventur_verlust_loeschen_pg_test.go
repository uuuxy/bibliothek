package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das endgültige Löschen eines abgeschriebenen Exemplars an dem Zustand, den es in einer
// Schulbibliothek vorfindet: Ein Buch verschwindet gewöhnlich, nachdem es ausgeliehen war, und
// ausleihen.exemplar_id steht auf ON DELETE RESTRICT. Ein Test mit frischen, nie verliehenen
// Exemplaren prüfte den Fall, den es kaum gibt.
func TestLoescheVerlusteEndgueltig_MitAusleihhistorie(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	exemplarID := legeVerlorenesExemplarAn(t, ctx, repo, "Erdkundebuch 7", "VL-HIST", bearbeiter)
	schuelerID := legeSchuelerFuerVerlustAn(t, ctx, pool, "VL-S1")
	ausleiheID := legeZurueckgegebeneAusleiheAn(t, ctx, pool, exemplarID, schuelerID)

	// Eine bezahlte Forderung an derselben Ausleihe ist erledigt: Sie sperrt nicht und geht
	// mit, sonst hielte ihr RESTRICT das Exemplar fest.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		VALUES ($1, $2, $3, 'Eselsohr', 2.00, true)`, exemplarID, ausleiheID, schuelerID); err != nil {
		t.Fatalf("bezahlte Gebühr: %v", err)
	}

	geloescht, err := repo.LoescheVerlusteEndgueltig(ctx, []string{exemplarID}, bearbeiter)
	if err != nil {
		t.Fatalf("Exemplar mit Ausleihhistorie ließ sich nicht löschen: %v", err)
	}
	if len(geloescht) != 1 || geloescht[0] != exemplarID {
		t.Fatalf("zurückgemeldet %v, erwartet genau %s", geloescht, exemplarID)
	}

	// Der Barcode ist wieder frei: buecher_exemplare.barcode_id ist ohne Teilindex eindeutig.
	// Bliebe die Zeile stehen, wäre der Aufkleber für immer verbraucht.
	titelID := testdaten.LegeTitelAn(t, pool, "Erdkundebuch 7 (neu)", 5)
	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'VL-HIST')`, titelID); err != nil {
		t.Errorf("Barcode wurde nicht freigegeben, Wiedereinpflegen unmöglich: %v", err)
	}

	// Der Fehlbestandsbericht behält die Abschrift (ON DELETE SET NULL); sonst wäre nach dem
	// Aufräumen nicht mehr zu sehen, was abgeschrieben wurde.
	var abschrift int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM inventur_verluste WHERE barcode_id = 'VL-HIST'`).Scan(&abschrift); err != nil {
		t.Fatalf("Abschrift: %v", err)
	}
	if abschrift != 1 {
		t.Errorf("Fehlbestandsbericht hat die Abschrift verloren (%d Zeilen)", abschrift)
	}
}

// Eine unbezahlte Forderung gehört der Schule. Sie mitzulöschen hieße, sie aufzugeben, als
// Nebenwirkung eines Knopfs zum Aufräumen.
func TestLoescheVerlusteEndgueltig_OffeneForderungSperrt(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	frei := legeVerlorenesExemplarAn(t, ctx, repo, "Mathebuch 5", "VL-FREI", bearbeiter)
	gesperrt := legeVerlorenesExemplarAn(t, ctx, repo, "Mathebuch 6", "VL-GEBUEHR", bearbeiter)
	schuelerID := legeSchuelerFuerVerlustAn(t, ctx, pool, "VL-S2")
	ausleiheID := legeZurueckgegebeneAusleiheAn(t, ctx, pool, gesperrt, schuelerID)
	if _, err := pool.Exec(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		VALUES ($1, $2, $3, 'Buch verloren', 24.90, false)`, gesperrt, ausleiheID, schuelerID); err != nil {
		t.Fatalf("offene Gebühr: %v", err)
	}

	_, err := repo.LoescheVerlusteEndgueltig(ctx, []string{frei, gesperrt}, bearbeiter)
	if err == nil {
		t.Fatal("Stapel mit offener Gebühr durchgelaufen — die Forderung wäre weg")
	}
	// Die Tür macht daraus 409 mit dem Grund im Text; ohne den Fehlerwert würde daraus 500, und
	// der Sanitizer ersetzte die Begründung.
	if !errors.Is(err, ErrVerlustNochGebunden) {
		t.Fatalf("falscher Fehlertyp, wird zur 500 sanitisiert: %v", err)
	}
	// Der Barcode steht in der Meldung; sonst weiß niemand, welches Exemplar den Stapel
	// aufhält.
	if !strings.Contains(err.Error(), "VL-GEBUEHR") {
		t.Errorf("Meldung nennt das blockierende Exemplar nicht: %v", err)
	}

	// Alles oder nichts: Auch das unbelastete Exemplar bleibt stehen.
	for _, fall := range []struct{ id, name string }{{frei, "VL-FREI"}, {gesperrt, "VL-GEBUEHR"}} {
		var da bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM buecher_exemplare WHERE id = $1)`, fall.id).Scan(&da); err != nil {
			t.Fatal(err)
		}
		if !da {
			t.Errorf("%s wurde trotz abgewiesenem Stapel gelöscht", fall.name)
		}
	}
	var gebuehrDa bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schadensfaelle WHERE exemplar_id = $1 AND ist_bezahlt = false)`, gesperrt).Scan(&gebuehrDa); err != nil {
		t.Fatal(err)
	}
	if !gebuehrDa {
		t.Error("die offene Forderung wurde gelöscht")
	}
}

func legeSchuelerFuerVerlustAn(t *testing.T, ctx context.Context, pool *pgxpool.Pool, barcode string) string {
	t.Helper()
	id := testdaten.LegeSchuelerMitNamenAn(t, pool, barcode, "Vera", "Lust", "7a", 2030)
	return id
}

// legeZurueckgegebeneAusleiheAn legt eine abgeschlossene Ausleihe an: die Historie, an der ON
// DELETE RESTRICT hängt.
func legeZurueckgegebeneAusleiheAn(t *testing.T, ctx context.Context, pool *pgxpool.Pool, exemplarID, schuelerID string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, now() - interval '40 days', now() - interval '26 days', now() - interval '20 days')
		RETURNING id`, exemplarID, schuelerID).Scan(&id); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	return id
}
