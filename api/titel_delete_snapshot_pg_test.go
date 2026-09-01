package api

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// TestDeleteTitle_TresenAuskunftFindetGeloeschteExemplare ist das Paar-Gate zum
// B-Posten „DeleteTitle löscht Exemplare ohne Barcode-Snapshot" (Befund-Register,
// 01.09.2026): Die Tresen-Auskunft findet gelöschte Exemplare AUSSCHLIESSLICH über
// audit_log-Zeilen mit tabelle='buecher_exemplare' und details->>'barcode_id'
// (repository.SucheTresenExemplare). DeleteCopy und das Verlust-Löschen schreiben
// diese Spur längst — beim Löschen eines ganzen TITELS fiel jedes Exemplar ohne
// sie: Wer so ein Buch später am Tresen scannte, bekam „nie gesehen" statt
// „gelöscht am …".
//
// Geprüft wird das PAAR (Schreiber ↔ Leseweg), nicht die Audit-Zeile allein: Eine
// Zeile im falschen Format sähe im SELECT count(*) gut aus und bliebe für die
// Auskunft trotzdem unsichtbar (Lehre LUSD-Paarungs-Gates).
func TestDeleteTitle_TresenAuskunftFindetGeloeschteExemplare(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := seedMonitorTitel(t, pool, "Snapshot-Titel", "Sig Snap", false, 0)
	exemplar(t, pool, titelID, "SNAP-TITELDEL-1", true, "")
	exemplar(t, pool, titelID, "SNAP-TITELDEL-2", true, "")
	bearbeiter := seedPortalLehrkraft(t, pool, "titelloescher@test.invalid")

	if err := repository.NewAuditRepository(pool).DeleteTitle(ctx, titelID, bearbeiter); err != nil {
		t.Fatalf("DeleteTitle: %v", err)
	}

	for _, barcode := range []string{"SNAP-TITELDEL-1", "SNAP-TITELDEL-2"} {
		zeilen, err := repository.SucheTresenExemplare(ctx, pool, barcode)
		if err != nil {
			t.Fatalf("Tresen-Auskunft für %s: %v", barcode, err)
		}
		if len(zeilen) != 1 {
			t.Fatalf("Barcode %s: %d Treffer in der Tresen-Auskunft, erwartet genau 1 — "+
				"ohne Snapshot ist das Exemplar nach dem Titel-Löschen unauffindbar", barcode, len(zeilen))
		}
		if zeilen[0].Status != "geloescht" {
			t.Errorf("Barcode %s: Status %q, erwartet geloescht", barcode, zeilen[0].Status)
		}
		if zeilen[0].Titel != "Snapshot-Titel" {
			t.Errorf("Barcode %s: Auskunft nennt Titel %q, erwartet Snapshot-Titel — "+
				"ohne Titel im Snapshot sagt die Trefferzeile nichts aus", barcode, zeilen[0].Titel)
		}
	}
}
