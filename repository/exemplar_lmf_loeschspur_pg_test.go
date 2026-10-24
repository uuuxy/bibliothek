package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/testdaten"
)

// Ein gelöschtes Exemplar bleibt an der Theke über sein altes LMF-Etikett auffindbar: Beide
// Türen dieses Pakets, die ein Exemplar endgültig löschen, schreiben das alte Etikett in ihre
// Spur (merkeLMFBarcode), und FindeThekenExemplare sucht dort. Die dritte Tür, die
// Massenlöschung der Titel, prüft repository/titel_loeschen_lmf_pg_test.go.
func TestAltesLMFEtikett_GeloeschtesExemplarBleibtAuffindbar(t *testing.T) {
	pool := pgTestPool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bearbeiter := testdaten.LegeKollegiumskontoAn(t, pool, "lmf-loeschspur@test.invalid")
	marke := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)

	titelID := testdaten.LegeTitelAn(t, pool, "LMF-Löschspur-Titel", 0)
	mitTitel := testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFL-1-"+marke, true, "")
	verloren := testdaten.LegeExemplarAn(t, pool, testdaten.LegeTitelAn(t, pool, "LMF-Verlust-Titel", 0),
		"B-LMFL-2-"+marke, true, "")
	alt := map[string]string{
		mitTitel: "LMF-9783060158423-" + marke + "1",
		verloren: "LMF-9783060158423-" + marke + "2",
	}
	for id, nummer := range alt {
		if err := setzeLMFBarcode(t, pool, id, nummer); err != nil {
			t.Fatalf("altes Etikett setzen: %v", err)
		}
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM audit_log WHERE tabelle = 'buecher_exemplare' AND datensatz_id = ANY($1::uuid[])`,
			[]string{mitTitel, verloren}); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	if err := NewKatalogAbfragen(pool).LoescheTitel(ctx, titelID, bearbeiter); err != nil {
		t.Fatalf("LoescheTitel: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'VERLUST' WHERE id = $1`, verloren); err != nil {
		t.Fatalf("Verlust buchen: %v", err)
	}
	if geloescht, err := NewInventurAbfragen(pool).LoescheVerlusteEndgueltig(ctx, []string{verloren}, bearbeiter); err != nil || len(geloescht) != 1 {
		t.Fatalf("Verlust endgültig löschen: %v (gelöscht: %v)", err, geloescht)
	}

	for id, nummer := range alt {
		zeilen, err := FindeThekenExemplare(ctx, pool, nummer)
		if err != nil {
			t.Fatalf("Theken-Auskunft für %s: %v", nummer, err)
		}
		if len(zeilen) != 1 || zeilen[0].ExemplarID != id || zeilen[0].Status != "geloescht" {
			t.Errorf("Theken-Auskunft über das alte Etikett %s = %+v, erwartet das gelöschte Exemplar %s", nummer, zeilen, id)
		}
	}
}

// Die Suche unter „Fehlende Etiketten" findet ein Exemplar auch über sein altes LMF-Etikett, in
// der Liste und in der Zahl: Ist das Etikett beschädigt, ist es oft das, was noch zu lesen ist.
func TestAltesLMFEtikett_NachdruckSucheFindetDasExemplar(t *testing.T) {
	pool := pgTestPool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := testdaten.LegeTitelAn(t, pool, "LMF-Nachdruck-Titel", 0)
	exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFN-1", true, "")
	testdaten.LegeExemplarAn(t, pool, titelID, "B-LMFN-2", true, "")
	if err := setzeLMFBarcode(t, pool, exemplarID, "LMF-9783060158423-951"); err != nil {
		t.Fatalf("altes Etikett setzen: %v", err)
	}

	liste, err := LadeEtikettenExemplare(ctx, pool, "alle", "9783060158423-951", 50)
	if err != nil {
		t.Fatalf("LadeEtikettenExemplare: %v", err)
	}
	if len(liste) != 1 || liste[0].BarcodeID != "B-LMFN-1" {
		t.Errorf("Suche über das alte Etikett = %+v, erwartet das Exemplar B-LMFN-1", liste)
	}
	if n, err := ZaehleEtikettenExemplare(ctx, pool, "alle", nil, "lmf-9783060158423-951"); err != nil || n != 1 {
		t.Errorf("Zahl zur Suche über das alte Etikett = %d (%v), erwartet 1", n, err)
	}
}
