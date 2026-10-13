package repository_test

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// TestLoescheTitel_TresenAuskunftFindetGeloeschteExemplare ist das Paar-Gate zum Löschen eines
// Titels: Die Tresen-Auskunft findet gelöschte Exemplare ausschließlich über audit_log-Zeilen
// mit tabelle='buecher_exemplare' und details->>'barcode_id' (repository.FindeTresenExemplare).
// SondereExemplarAus und das Verlust-Löschen schreiben diese Spur; fiele beim Löschen eines
// ganzen Titels jedes Exemplar ohne sie, bekäme, wer so ein Buch später am Tresen scannt, „nie
// gesehen" statt „gelöscht am …".
//
// Geprüft wird das Paar (Schreiber ↔ Leseweg), nicht die Zeile im Protokoll allein: Eine Zeile
// im falschen Format sähe im SELECT count(*) gut aus und bliebe für die Auskunft trotzdem
// unsichtbar.
func TestLoescheTitel_TresenAuskunftFindetGeloeschteExemplare(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := testdaten.SeedMonitorTitel(t, pool, "Snapshot-Titel", "Sig Snap", false, 0)
	testdaten.Exemplar(t, pool, titelID, "SNAP-TITELDEL-1", true, "")
	testdaten.Exemplar(t, pool, titelID, "SNAP-TITELDEL-2", true, "")
	bearbeiter := testdaten.SeedPortalLehrkraft(t, pool, "titelloescher@test.invalid")

	if err := repository.NewProtokollAbfragen(pool).LoescheTitel(ctx, titelID, bearbeiter); err != nil {
		t.Fatalf("LoescheTitel: %v", err)
	}

	for _, barcode := range []string{"SNAP-TITELDEL-1", "SNAP-TITELDEL-2"} {
		zeilen, err := repository.FindeTresenExemplare(ctx, pool, barcode)
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

	// Der Eintrag zum Titel selbst hängt an der Kennung des Titels und nennt, wer gelöscht hat.
	if n := pgtest.ZaehleZeilen(t, pool, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'buecher_titel' AND aktion = 'DELETE' AND datensatz_id = $1::uuid
		  AND bearbeiter_id = $2::uuid AND details->>'titel' = 'Snapshot-Titel'`, titelID, bearbeiter); n != 1 {
		t.Errorf("%d Einträge zum gelöschten Titel mit seiner Kennung und dem Bearbeiter, erwartet 1", n)
	}
	// Je Exemplar derselbe Bearbeiter und der Bezug zum Titel.
	if n := pgtest.ZaehleZeilen(t, pool, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'buecher_exemplare' AND aktion = 'DELETE' AND bearbeiter_id = $2::uuid
		  AND details->>'titel_id' = $1`, titelID, bearbeiter); n != 2 {
		t.Errorf("%d Einträge zu den Exemplaren mit Titelbezug und Bearbeiter, erwartet 2", n)
	}
}
