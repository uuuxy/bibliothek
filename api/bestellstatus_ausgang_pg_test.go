package api

import (
	"context"
	"testing"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// bestellstatus heißt „dieses Exemplar ist bestellt und noch nicht da" (Migration 071).
// Gesetzt wird er beim Bestellen, geräumt beim Wareneingang. Jeder ANDERE Ausgang aus dem
// Zulauf muss ihn ebenfalls räumen — sonst filtern alle Leser, die `bestellstatus IS NULL`
// voraussetzen (OPAC, Inventur, Lernmittel-Übersicht, Katalog/Monitor), das Exemplar für
// immer weg (Bestands-Durchgang 10.09.2026, Bugklasse „Zustands-Ausgang ohne Räumer").
//
// Die zwei Türen, die das nicht taten:
//   - der Status-Editor der Buchakte (UpdateCopyStatus) — ein angekommenes Buch auf
//     „verfügbar" gestellt statt über den Wareneingang: ausleihbar, aber nirgends gezählt;
//   - das Aussondern (DecommissionCopy) — ein Buch, das der Händler nicht liefert: Es
//     blieb für immer im Wareneingang, und „alle einbuchen" machte das AUSGESONDERTE
//     Exemplar wieder ausleihbar.
func TestBestellstatus_JederAusgangRaeumt(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)
	books := repository.NewBookRepository(pool)

	zulauf := func(titel, barcode string) string {
		t.Helper()
		var titelID, id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp)
			VALUES ($1, 'Zulauf', 'Buch') RETURNING id`, titel).Scan(&titelID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_notiz, ist_ausleihbar, bestellstatus)
			VALUES ($1, $2, 'Im Zulauf - Testverlag', false, 'im_zulauf') RETURNING id`, titelID, barcode).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	bestellstatus := func(id string) *string {
		t.Helper()
		var s *string
		if err := pool.QueryRow(ctx, `SELECT bestellstatus FROM buecher_exemplare WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	// 1. Status-Editor: auf „verfügbar" gestellt → kein Zulauf mehr, im OPAC gezählt.
	frei := zulauf("Zulauf-Editor-Titel", "ZL-EDIT")
	if err := books.UpdateCopyStatus(ctx, frei, true, false, ""); err != nil {
		t.Fatalf("UpdateCopyStatus: %v", err)
	}
	if s := bestellstatus(frei); s != nil {
		t.Errorf("Status-Editor ließ bestellstatus=%q stehen", *s)
	}
	if n := opacGesamt(t, pool, "Zulauf-Editor-Titel"); n != 1 {
		t.Errorf("freigegebenes Exemplar zählt im OPAC nicht (gesamt=%d)", n)
	}

	// 2. Aussondern: kein Zulauf mehr, nicht im Wareneingang, und einbuchen belebt es nicht.
	weg := zulauf("Zulauf-Storno-Titel", "ZL-STORNO")
	if err := books.DecommissionCopy(ctx, weg); err != nil {
		t.Fatalf("DecommissionCopy: %v", err)
	}
	if s := bestellstatus(weg); s != nil {
		t.Errorf("Aussondern ließ bestellstatus=%q stehen", *s)
	}
	gruppen, err := service.GetIncomingShipments(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range gruppen {
		for _, it := range g.Items {
			for _, id := range it.ExemplarIDs {
				if id == weg {
					t.Errorf("ausgesondertes Exemplar steht im Wareneingang")
				}
			}
		}
	}
	if _, err := service.BulkReceiveOrder(ctx, pool, repository.NewAuditRepository(pool), service.BulkReceiveParams{
		ExemplarIDs: []string{weg}, AdminID: adminFuerAudit(t, pool),
	}); err != nil {
		t.Logf("Wareneingang meldet: %v", err)
	}
	var ausleihbar, ausgesondert bool
	if err := pool.QueryRow(ctx, `SELECT ist_ausleihbar, ist_ausgesondert FROM buecher_exemplare WHERE id = $1`, weg).
		Scan(&ausleihbar, &ausgesondert); err != nil {
		t.Fatal(err)
	}
	if ausleihbar || !ausgesondert {
		t.Errorf("Wareneingang belebte ein ausgesondertes Exemplar: ausleihbar=%v ausgesondert=%v", ausleihbar, ausgesondert)
	}
}
