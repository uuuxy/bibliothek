package repository

import (
	"context"
	"testing"
)

// Ein Band für die Jahrgänge 5 und 6 gehört zu den Läufen beider Jahrgänge, steht aber in einem
// Regal und wird nur in einem Lauf gescannt. Laufen beide zugleich, zählt der Scan für beide;
// sonst buchte der Abschluss des anderen Laufs das Buch als Verlust, obwohl es im Regal steht.
// Auch ein Scan, bevor der zweite Lauf begann, zählt: Wer später dazukommt, scannt die Regale
// nicht noch einmal, die der erste schon durchgegangen ist.
func TestSchliesseLaufAb_ScanEinesGleichzeitigenLaufsZaehlt(t *testing.T) {
	for _, fall := range []struct {
		name          string
		scanVorBeginn bool
	}{
		{"Scan nach dem Beginn beider Läufe", false},
		{"Scan vor dem Beginn des zweiten Laufs", true},
	} {
		t.Run(fall.name, func(t *testing.T) {
			pool := pgTestPool(t)
			resetInventurDaten(t, pool)
			ctx := context.Background()
			repo := NewInventurAbfragen(pool)
			bearbeiter := legeBearbeiterAn(t, pool)

			band := legeFachExemplarAn(t, pool, "Mathematik", 5, 6, "BC-MA56")
			fuenf, sechs := 5, 6
			lauf5, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Jahrgang: &fuenf}, "Kl. 5", "")
			if err != nil {
				t.Fatalf("Lauf für Jahrgang 5 anlegen: %v", err)
			}
			scanne := func() {
				t.Helper()
				if _, err := repo.VerbucheScan(ctx, lauf5.ID, lauf5.Bereich(), band); err != nil {
					t.Fatalf("Scan im Lauf für Jahrgang 5: %v", err)
				}
			}
			if fall.scanVorBeginn {
				scanne()
			}
			lauf6, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Jahrgang: &sechs}, "Kl. 6", "")
			if err != nil {
				t.Fatalf("Lauf für Jahrgang 6 anlegen: %v", err)
			}
			if !fall.scanVorBeginn {
				scanne()
			}

			// Der Lauf für Jahrgang 5 schließt zuerst ab, wie im Befund.
			if verloren, err := repo.SchliesseLaufAb(ctx, lauf5.ID, lauf5.Bereich(), bearbeiter); err != nil {
				t.Fatalf("Abschluss Jahrgang 5: %v", err)
			} else if verloren != 0 {
				t.Errorf("Abschluss Jahrgang 5 buchte %d Verluste, erwartet 0", verloren)
			}

			stand, err := repo.ZaehleStand(ctx, lauf6.ID, lauf6.Bereich())
			if err != nil {
				t.Fatalf("Stand Jahrgang 6: %v", err)
			}
			if stand != (LaufStand{Erwartet: 1, Fehlend: 0}) {
				t.Errorf("Stand Jahrgang 6 = %+v, erwartet 1 erwartet und 0 fehlend", stand)
			}
			if verloren, err := repo.SchliesseLaufAb(ctx, lauf6.ID, lauf6.Bereich(), bearbeiter); err != nil {
				t.Fatalf("Abschluss Jahrgang 6: %v", err)
			} else if verloren != 0 {
				t.Errorf("Abschluss Jahrgang 6 buchte %d Verluste, erwartet 0", verloren)
			}
			if aus := ausgesonderteZahl(t, pool, []string{band}); aus != 0 {
				t.Error("der im Lauf für Jahrgang 5 gescannte Band ist im Lauf für Jahrgang 6 als Verlust gebucht")
			}
		})
	}
}

// Ein Lauf, der schon vor dem Beginn eines anderen abgeschlossen war, lief nicht zugleich: Sein
// Scan sagt nichts darüber, ob das Buch seit dem Beginn des anderen Laufs im Regal stand.
func TestSchliesseLaufAb_ScanEinesFrueherenLaufsZaehltNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	band := legeFachExemplarAn(t, pool, "Mathematik", 5, 6, "BC-MA56")
	fuenf, sechs := 5, 6
	lauf5, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Jahrgang: &fuenf}, "Kl. 5", "")
	if err != nil {
		t.Fatalf("Lauf für Jahrgang 5 anlegen: %v", err)
	}
	if _, err := repo.VerbucheScan(ctx, lauf5.ID, lauf5.Bereich(), band); err != nil {
		t.Fatalf("Scan im Lauf für Jahrgang 5: %v", err)
	}
	if _, err := repo.SchliesseLaufAb(ctx, lauf5.ID, lauf5.Bereich(), bearbeiter); err != nil {
		t.Fatalf("Abschluss Jahrgang 5: %v", err)
	}
	// Der erste Lauf liegt eine Stunde zurück, damit die Reihenfolge nicht an Mikrosekunden hängt.
	if _, err := pool.Exec(ctx, `
		UPDATE inventur_sessions
		SET gestartet_am = now() - interval '2 hours', abgeschlossen_am = now() - interval '1 hour'
		WHERE id = $1`, lauf5.ID); err != nil {
		t.Fatalf("Lauf 5 zurückdatieren: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE inventur_erfassungen SET erfasst_am = now() - interval '90 minutes' WHERE session_id = $1`,
		lauf5.ID); err != nil {
		t.Fatalf("Scan zurückdatieren: %v", err)
	}

	lauf6, err := repo.LegeLaufAn(ctx, "filter", InventurBereich{Jahrgang: &sechs}, "Kl. 6", "")
	if err != nil {
		t.Fatalf("Lauf für Jahrgang 6 anlegen: %v", err)
	}
	if verloren, err := repo.SchliesseLaufAb(ctx, lauf6.ID, lauf6.Bereich(), bearbeiter); err != nil {
		t.Fatalf("Abschluss Jahrgang 6: %v", err)
	} else if verloren != 1 {
		t.Errorf("Abschluss Jahrgang 6 buchte %d Verluste, erwartet 1", verloren)
	}
}
