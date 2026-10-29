package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/schulzeit"
)

// Die Massenaktion „Titel löschen" vermerkt an der Spur jedes Exemplars, ob es im Bestand
// war. Das Abgangsbuch zählt danach; ein bestelltes, nie eingetroffenes Exemplar ist kein
// Abgang.
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

	// Das Abgangsbuch zählt gelöschte Exemplare nur über ihre Spur. Gemessen wird der Zuwachs, weil
	// andere Tests auch Spuren hinterlassen, über drei Tage, damit Mitternacht nicht dazwischenfällt.
	jetzt := schulzeit.Jetzt()
	abgaenge := func() int {
		t.Helper()
		buch, err := LadeAbgangsbuch(ctx, pool, jetzt.AddDate(0, 0, -1), jetzt.AddDate(0, 0, 1))
		if err != nil {
			t.Fatalf("Abgangsbuch: %v", err)
		}
		return buch.AusKatalogGeloescht
	}
	vorher := abgaenge()

	if err := NewTitelAbfragen(pool).LoescheTitelSamtAusleihen(ctx, []string{titelID}, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("Titel löschen: %v", err)
	}

	if zuwachs := abgaenge() - vorher; zuwachs != 1 {
		t.Errorf("Das Abgangsbuch zählt %d gelöschte Exemplare mehr, erwartet 1: das im Bestand, nicht das im Zulauf", zuwachs)
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
