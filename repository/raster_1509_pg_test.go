//go:build raster

package repository

// Nachstellung Rasterdurchgang 15.09.2026 abends (OFFEN.md 5.15). Build-Tag raster: läuft nur
// mit -tags raster. Rot heißt „bestätigt".

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Verdacht B: Der Stand der Barcode-Liste ist Anzahl + max(aktualisiert_am), und
// aktualisiert_am trägt den Beginn der Transaktion. Eine lange Transaktion, die ein Etikett
// ändert und nach einem kürzeren Schreiber committet, ändert keins von beiden.
func TestRaster_StandMerkerUebersiehtLangeTransaktion(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	x := stempelAufbau(t, pool)
	y := stempelAufbau(t, pool)

	lang, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = lang.Rollback(ctx) }()
	var beginn time.Time
	if err := lang.QueryRow(ctx, `SELECT now()`).Scan(&beginn); err != nil {
		t.Fatalf("now: %v", err)
	}
	warte(t, pool)
	warte(t, pool)

	// Kurzer Schreiber, committet sofort.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET zustand_notiz = 'kurz' WHERE id = $1`, y.exemplarID); err != nil {
		t.Fatalf("kurzer Schreiber: %v", err)
	}
	vorher, err := LiesBuchbarcodeStand(ctx, pool)
	if err != nil {
		t.Fatalf("Stand vorher: %v", err)
	}

	// Die lange Transaktion etikettiert X um (wie UpdateCopyBarcode) und committet danach.
	neu := fmt.Sprintf("B-RASTER-%d", time.Now().UnixNano())
	if _, err := lang.Exec(ctx, `UPDATE buecher_exemplare SET barcode_id = $1, aktualisiert_am = CURRENT_TIMESTAMP WHERE id = $2`, neu, x.exemplarID); err != nil {
		t.Fatalf("umetikettieren: %v", err)
	}
	if err := lang.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	nachher, err := LiesBuchbarcodeStand(ctx, pool)
	if err != nil {
		t.Fatalf("Stand nachher: %v", err)
	}
	liste, err := ListeBuchbarcodes(ctx, pool, 0)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	drin := false
	for _, b := range liste {
		drin = drin || b == neu
	}
	gleich := vorher.Anzahl == nachher.Anzahl && vorher.Juengste != nil && nachher.Juengste != nil && vorher.Juengste.Equal(*nachher.Juengste)
	t.Logf("Beginn lange Tx %v · Stand vorher %d/%v · nachher %d/%v · neues Etikett in der Liste: %v",
		beginn, vorher.Anzahl, vorher.Juengste, nachher.Anzahl, nachher.Juengste, drin)
	if gleich && drin {
		t.Errorf("NACHGESTELLT: Der Stand ist unverändert, obwohl die Liste jetzt %s führt — die Theke bekäme 304 und behielte die alte Nummer", neu)
	}
}
