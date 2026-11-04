package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/schulzeit"
)

// Die Massenaktion „Titel löschen" vermerkt an der Spur jedes Exemplars, ob es im Bestand
// war, und das Abgangsbuch behält die Zeile des Exemplars aus dem Bestand; ein bestelltes, nie
// eingetroffenes Exemplar ist kein Abgang.
func TestLoescheTitel_VermerktObDasExemplarImBestandWar(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var titelID, imBestand, imZulauf string
	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `DELETE FROM audit_log WHERE datensatz_id = ANY($1)`,
			[]string{titelID, imBestand, imZulauf}); err != nil {
			t.Errorf("Protokoll aufräumen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM geloeschte_exemplare WHERE exemplar_id = ANY($1)`,
			[]string{imBestand, imZulauf}); err != nil {
			t.Errorf("Buchzeilen aufräumen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = 'Bestand und Zulauf'`); err != nil {
			t.Errorf("Titel aufräumen: %v", err)
		}
	})
	titelID = testdaten.LegeTitelAn(t, pool, "Bestand und Zulauf", 5)
	imBestand = testdaten.LegeExemplarAn(t, pool, titelID, "DB-BESTAND", true, "")
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
		VALUES ($1, 'DB-ZULAUF', false, 'im_zulauf') RETURNING id`, titelID).Scan(&imZulauf); err != nil {
		t.Fatalf("Exemplar im Zulauf: %v", err)
	}

	// Über drei Tage, damit Mitternacht nicht dazwischenfällt. Andere Tests hinterlassen auch
	// Zeilen; gezählt werden nur die beiden Barcodes dieses Tests.
	jetzt := schulzeit.Jetzt()
	zeilen := func() map[string]int {
		t.Helper()
		buch, err := LadeAbgangsbuch(ctx, pool, jetzt.AddDate(0, 0, -1), jetzt.AddDate(0, 0, 1))
		if err != nil {
			t.Fatalf("Abgangsbuch: %v", err)
		}
		gezaehlt := map[string]int{}
		for _, z := range buch.Zeilen {
			gezaehlt[z.Barcode]++
		}
		return gezaehlt
	}

	if _, err := NewTitelAbfragen(pool).LoescheTitel(ctx, TitelLoeschauftrag{TitelIDs: []string{titelID}, Grund: "AUSSORTIERT", BearbeiterID: testdaten.LegeBearbeiterAn(t, pool)}); err != nil {
		t.Fatalf("Titel löschen: %v", err)
	}

	if gezaehlt := zeilen(); gezaehlt["DB-BESTAND"] != 1 || gezaehlt["DB-ZULAUF"] != 0 {
		t.Errorf("Abgangsbuch nach dem Löschen: DB-BESTAND %d-mal, DB-ZULAUF %d-mal, erwartet 1 und 0",
			gezaehlt["DB-BESTAND"], gezaehlt["DB-ZULAUF"])
	}

	for _, fall := range []struct {
		id, barcode string
		erwartet    bool
	}{
		{imBestand, "DB-BESTAND", true},
		{imZulauf, "DB-ZULAUF", false},
	} {
		var vermerk *bool
		if err := pool.QueryRow(ctx, `
			SELECT (details->>$2)::boolean FROM audit_log
			WHERE tabelle = 'buecher_exemplare' AND aktion = 'DELETE' AND datensatz_id = $1`,
			fall.id, ProtokollDetailWarImBestand).Scan(&vermerk); err != nil {
			t.Fatalf("Spur von %s: %v", fall.barcode, err)
		}
		if vermerk == nil || *vermerk != fall.erwartet {
			t.Errorf("%s: Vermerk %v, erwartet %v", fall.barcode, vermerk, fall.erwartet)
		}
	}
}

// Nur die Ausleihen dieses Titels halten das Löschen auf. Griffe die Bedingung weiter, ließe sich im
// Betrieb kein Titel mehr löschen, solange irgendein anderes Buch verliehen ist.
func TestLoescheTitel_AusleiheEinesAnderenTitelsHaeltNichtAuf(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	bearbeiter := legeBearbeiterAn(t, pool)
	eigen := legeSignaturMitExemplarenAn(t, pool, "LoeschEigen", 1)
	fremd := legeSignaturMitExemplarenAn(t, pool, "LoeschFremd", 1)
	legeAusleiheAn(t, pool, fremd[0], legeSchuelerAn(t, pool, "LOESCH-A", "Cora", "8a"), bearbeiter)

	if _, err := NewTitelAbfragen(pool).LoescheTitel(context.Background(), TitelLoeschauftrag{TitelIDs: []string{titelIDVonExemplar(t, pool, eigen[0])}, Grund: "AUSSORTIERT", BearbeiterID: bearbeiter}); err != nil {
		t.Errorf("LoescheTitel = %v, erwartet Erfolg", err)
	}
}
