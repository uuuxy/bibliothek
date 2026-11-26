package repository_test

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Ein Eintrag der Datensatz-Historie steht in der Auskunft genau einmal, auch wenn Geräte im
// Bestand sind: Das Gerät gehört nur zu einem Eintrag, dessen Kennung seine ist.
func TestLadeDSGVOProtokollEintraege_GeraeteVervielfachenKeinenEintrag(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	sid := testdaten.LegeSchuelerAn(t, pool, "S-DSGVO-G1", "Mia", "5a")
	titelID := testdaten.LegeTitelMitCoverUndAlterAn(t, pool, "Auskunft-Titel", "Jug Aus", false, 0)
	exID := testdaten.LegeExemplarAn(t, pool, titelID, "AUSK-1", true, "")
	if _, err := pool.Exec(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES
		('Tablet A', 'GER-AUSK-1'), ('Tablet B', 'GER-AUSK-2')`); err != nil {
		t.Fatalf("Geräte anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO audit_log (tabelle, aktion, datensatz_id, details)
		VALUES ('ausleihen', 'INSERT', $1, jsonb_build_object('schueler_id', $2::text))`, exID, sid); err != nil {
		t.Fatalf("Ausleihe in der Historie anlegen: %v", err)
	}

	zeilen, err := repository.LadeDSGVOProtokollEintraege(ctx, pool, sid)
	if err != nil {
		t.Fatalf("LadeDSGVOProtokollEintraege: %v", err)
	}
	if len(zeilen) != 1 {
		t.Fatalf("%d Einträge in der Auskunft, erwartet einer: %+v", len(zeilen), zeilen)
	}
	if z := zeilen[0]; z.Gegenstand != "Auskunft-Titel" || z.Barcode != "AUSK-1" {
		t.Errorf("Gegenstand %q, Nummer %q; erwartet das ausgeliehene Buch", z.Gegenstand, z.Barcode)
	}
}
