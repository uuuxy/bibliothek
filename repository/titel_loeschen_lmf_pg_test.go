package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Löscht die Bestandstabelle einen Titel, bleibt sein Exemplar an der Theke auch über das alte
// LMF-Etikett auffindbar: Die Spur nennt es (merkeLMFBarcode).
func TestLoescheTitel_AltesLMFEtikettBleibtAuffindbar(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	marke := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	nummer, alt := "B-LMFM-"+marke, "LMF-9783060158423-"+marke

	var titelID, exemplarID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, medientyp) VALUES ('LMF-Massenlöschung', 'Buch') RETURNING id`).
		Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, lmf_barcode) VALUES ($1, $2, $3) RETURNING id`,
		titelID, nummer, alt).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		for _, sql := range []string{
			`DELETE FROM audit_log WHERE tabelle IN ('buecher_exemplare', 'buecher_titel') AND datensatz_id = ANY($1::uuid[])`,
			`DELETE FROM buecher_exemplare WHERE id = ANY($1::uuid[]) OR titel_id = ANY($1::uuid[])`,
			`DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`,
		} {
			if _, err := pool.Exec(ctx, sql, []string{exemplarID, titelID}); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	})

	if err := NewTitelAbfragen(pool).LoescheTitelSamtAusleihen(ctx, []string{titelID}, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("LoescheTitelSamtAusleihen: %v", err)
	}

	zeilen, err := FindeThekenExemplare(ctx, pool, alt)
	if err != nil {
		t.Fatalf("Theken-Auskunft: %v", err)
	}
	if len(zeilen) != 1 || zeilen[0].ExemplarID != exemplarID || zeilen[0].Status != "geloescht" {
		t.Errorf("Theken-Auskunft über das alte Etikett = %+v, erwartet das gelöschte Exemplar %s", zeilen, exemplarID)
	}
}
