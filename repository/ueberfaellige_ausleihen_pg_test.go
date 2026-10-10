package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Sperr-Automatik zählt, was offen, ein Buch und seit mehr als der Kulanz überfällig ist:
// nicht die Ausleihe innerhalb der Kulanz, nicht die zurückgegebene, nicht die Dauerleihe,
// nicht das Gerät und nicht die Ausleihe eines anderen Lesers.
func TestZaehleUeberfaelligeBuecher_NurOffeneBuecherJenseitsDerKulanz(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	geraetID, leserID, bearbeiterID := geraeteAufbau(t, tx, "ueberfaellig")
	_, andererLeser, _ := geraeteAufbau(t, tx, "ueberfaellig-anderer")
	var titelID string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Überfällig-Probe') RETURNING id::text`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	buch := func(nummer string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id::text`,
			titelID, nummer).Scan(&id); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", nummer, err)
		}
		return id
	}
	// Jede Ausleihe begann vor 40 Tagen; die Zahl nennt, seit wie vielen Tagen ihre Frist vorbei ist.
	leihe := func(name, sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	const buchAusleihe = `INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, ist_handapparat)
		VALUES ($1, $2, $3, now() - interval '40 days', now() - make_interval(days => $4), $5::timestamptz, $6)`
	leihe("seit 20 Tagen überfällig", buchAusleihe, buch("UEB-20"), leserID, bearbeiterID, 20, nil, false)
	leihe("seit 10 Tagen überfällig", buchAusleihe, buch("UEB-10"), leserID, bearbeiterID, 10, nil, false)
	leihe("zurückgegeben", buchAusleihe, buch("UEB-ZURUECK"), leserID, bearbeiterID, 20, "now", false)
	leihe("Dauerleihe", buchAusleihe, buch("UEB-DAUER"), leserID, bearbeiterID, 20, nil, true)
	leihe("anderer Leser", buchAusleihe, buch("UEB-ANDERER"), andererLeser, bearbeiterID, 20, nil, false)
	leihe("Gerät", `INSERT INTO ausleihen (geraet_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, $3, now() - interval '40 days', now() - interval '20 days')`, geraetID, leserID, bearbeiterID)

	for kulanz, soll := range map[int]int{5: 2, 14: 1, 30: 0} {
		ist, err := ZaehleUeberfaelligeBuecher(ctx, tx, leserID, kulanz)
		if err != nil {
			t.Fatalf("Kulanz %d: %v", kulanz, err)
		}
		if ist != soll {
			t.Errorf("Kulanz %d Tage: %d überfällige Bücher gezählt, erwartet %d", kulanz, ist, soll)
		}
	}
}
