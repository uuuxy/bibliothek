package repository_test

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Die Regalansicht zeigt je Titel, wie viele Exemplare im Bestand sind und wie viele davon gerade
// verliehen. Ein ausgesondertes Exemplar zählt nicht, eine zurückgegebene Ausleihe auch nicht.
func TestLadeTitelUnterSignatur_ZaehltBestandUndVerliehene(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := testdaten.LegeTitelMitCoverUndAlterAn(t, pool, "Regal-Titel", "REG Zahl 1", false, 0)
	for i, leser := range []string{"S-SIG-Z1", "S-SIG-Z2"} {
		sid := testdaten.LegeSchuelerAn(t, pool, leser, "Mia", "5a")
		testdaten.LegeAusleiheVorZweiTagenAn(t, pool, testdaten.LegeExemplarAn(t, pool, titelID, fmt.Sprintf("REG-V%d", i+1), true, ""), sid)
	}
	zurueck := testdaten.LegeExemplarAn(t, pool, titelID, "REG-Z1", true, "")
	sid := testdaten.LegeSchuelerAn(t, pool, "S-SIG-Z3", "Ben", "5a")
	if _, err := pool.Exec(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, NOW() - interval '9 days', NOW() + interval '5 days', NOW() - interval '1 day')`, zurueck, sid); err != nil {
		t.Fatalf("zurückgegebene Ausleihe anlegen: %v", err)
	}
	ausgesondert := testdaten.LegeExemplarAn(t, pool, titelID, "REG-A1", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false,
		aussonderung_grund = 'AUSSORTIERT' WHERE id = $1`, ausgesondert); err != nil {
		t.Fatalf("Exemplar aussondern: %v", err)
	}

	zeilen, err := repository.LadeTitelUnterSignatur(ctx, pool, "REG Zahl", 10, false)
	if err != nil {
		t.Fatalf("LadeTitelUnterSignatur: %v", err)
	}
	if len(zeilen) != 1 {
		t.Fatalf("%d Titel unter der Signatur, erwartet einer", len(zeilen))
	}
	if z := zeilen[0]; z.Exemplare != 3 || z.Verliehen != 2 {
		t.Errorf("%d Exemplare, %d verliehen; erwartet 3 im Bestand, davon 2 verliehen", z.Exemplare, z.Verliehen)
	}
}
