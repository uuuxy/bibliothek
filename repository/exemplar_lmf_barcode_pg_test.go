package repository

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgconn"
)

// Ein altes LMF-Etikett (lmf_barcode) führt zu seinem Exemplar wie dessen Nummer: an der Theke
// und in der Suche (FindeExemplar), in der Inventur, in der Theken-Auskunft und in der
// Exemplarliste des Titels. Die Datenbank nimmt nur LMF-Nummern an, jede an genau einem Exemplar.

// setzeLMFBarcode hängt einem Exemplar ein altes LMF-Etikett an.
func setzeLMFBarcode(t *testing.T, db Datenbank, exemplarID, nummer string) error {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`UPDATE buecher_exemplare SET lmf_barcode = $1 WHERE id = $2`, nummer, exemplarID)
	return err
}

func TestAltesLMFEtikett_FuehrtZumExemplar(t *testing.T) {
	pool := pgTestPool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := testdaten.LegeTitelAn(t, pool, "LMF-Etikett-Titel", 0)
	exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFR-1", true, "")
	if err := setzeLMFBarcode(t, pool, exemplarID, "LMF-9783060158423-801"); err != nil {
		t.Fatalf("altes Etikett setzen: %v", err)
	}

	gefunden, err := NewKatalogAbfragen(pool).FindeExemplar(ctx, "LMF-9783060158423-801")
	if err != nil {
		t.Fatalf("FindeExemplar: %v", err)
	}
	if gefunden == nil || gefunden.ID != exemplarID || gefunden.BarcodeID != "B-LMFR-1" {
		t.Fatalf("FindeExemplar über das alte Etikett = %+v, erwartet %s mit der Nummer B-LMFR-1", gefunden, exemplarID)
	}

	inventur, err := NewInventurAbfragen(pool).LadeExemplar(ctx, "LMF-9783060158423-801")
	if err != nil {
		t.Fatalf("Inventur: %v", err)
	}
	if inventur.ExemplarID != exemplarID || inventur.BarcodeID != "B-LMFR-1" {
		t.Errorf("Inventur über das alte Etikett = %+v, erwartet %s mit der Nummer B-LMFR-1", inventur, exemplarID)
	}

	auskunft, err := FindeThekenExemplare(ctx, pool, "LMF-9783060158423-801")
	if err != nil {
		t.Fatalf("Theken-Auskunft: %v", err)
	}
	if len(auskunft) != 1 || auskunft[0].ExemplarID != exemplarID {
		t.Errorf("Theken-Auskunft über das alte Etikett = %+v, erwartet das Exemplar %s", auskunft, exemplarID)
	}

	akte, err := LadeExemplareDesTitels(ctx, pool, titelID)
	if err != nil {
		t.Fatalf("Exemplarliste: %v", err)
	}
	if len(akte) != 1 || akte[0].LMFBarcode != "LMF-9783060158423-801" {
		t.Errorf("Exemplarliste = %+v, erwartet das alte Etikett am Exemplar", akte)
	}
}

// Eine LMF-Nummer ist nie die Nummer eines Exemplars, in keiner Schreibweise, weder beim Anlegen
// noch im Barcode-Editor (chk_exemplar_barcode_keine_lmf_nummer, Migration 168); der Editor
// erkennt die Ablehnung (IstLMFNummerAlsBarcode).
func TestLMFNummer_NieDieNummerEinesExemplars(t *testing.T) {
	pool := pgTestPool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := testdaten.LegeTitelAn(t, pool, "LMF-Regel-Titel", 0)
	exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFR-21", true, "")
	for _, nummer := range []string{"LMF-9783060158423-821", "lmf-9783060158423-822", "Lmf-9783060158423-823"} {
		_, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`, titelID, nummer)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.ConstraintName != "chk_exemplar_barcode_keine_lmf_nummer" {
			t.Errorf("neues Exemplar mit %s: %v, erwartet die Ablehnung durch chk_exemplar_barcode_keine_lmf_nummer", nummer, err)
		}
		if err := NewKatalogAbfragen(pool).SetzeExemplarBarcode(ctx, exemplarID, nummer); !IstLMFNummerAlsBarcode(err) {
			t.Errorf("Barcode-Editor mit %s: %v, erwartet IstLMFNummerAlsBarcode", nummer, err)
		}
	}
}

func TestAltesLMFEtikett_NurLMFNummernJeEinmal(t *testing.T) {
	pool := pgTestPool(t)
	testdaten.LeereBestandsdaten(t, pool)

	titelID := testdaten.LegeTitelAn(t, pool, "LMF-Regel-Titel", 0)
	eins := testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFR-11", true, "")
	zwei := testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFR-12", true, "")

	var pgErr *pgconn.PgError
	err := setzeLMFBarcode(t, pool, eins, "B-LMFR-99")
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "chk_exemplar_lmf_barcode" {
		t.Errorf("eine Nummer ohne LMF- als altes Etikett: %v, erwartet die Ablehnung durch chk_exemplar_lmf_barcode", err)
	}

	if err := setzeLMFBarcode(t, pool, eins, "LMF-9783060158423-811"); err != nil {
		t.Fatalf("altes Etikett setzen: %v", err)
	}
	err = setzeLMFBarcode(t, pool, zwei, "LMF-9783060158423-811")
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "buecher_exemplare_lmf_barcode_key" {
		t.Errorf("dasselbe alte Etikett an einem zweiten Exemplar: %v, erwartet die Ablehnung durch buecher_exemplare_lmf_barcode_key", err)
	}
}
