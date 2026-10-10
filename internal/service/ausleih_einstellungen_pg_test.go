package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/lmfplan"
	"bibliothek/repository"
)

// Die Ausleihe liest ihre Einstellungen an der Datenbank: gesetzte Werte kommen an, eine Zeile
// ohne Wert oder mit leerem Wert lässt die Vorgabe stehen, und keine von beiden ist ein Fehler.
// In einer Transaktion, die am Ende zurückgerollt wird.
func TestLadeSystemEinstellungen_AnDerDatenbank(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	gesetzt := map[string]*string{
		"frist_buch_tage":              strPtr("30"),
		"frist_medien_tage":            strPtr(""),
		"max_ausleihen_schueler":       strPtr("8"),
		"lmf_stichtag":                 nil,
		"ferien_leseclub_aktiv":        strPtr("true"),
		"ferien_leseclub_zieldatum":    nil,
		"max_overdue_days":             strPtr("10"),
		"max_overdue_items":            strPtr("3"),
		lmfplan.SommerferienSchluessel: strPtr(`[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"}]`),
	}
	for schluessel, wert := range gesetzt {
		if _, err := tx.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, schluessel, wert); err != nil {
			t.Fatalf("%s setzen: %v", schluessel, err)
		}
	}

	ist, err := ladeSystemEinstellungen(ctx, tx)
	if err != nil {
		t.Fatalf("ladeSystemEinstellungen: %v", err)
	}
	soll := SystemEinstellungen{
		FristBuchTage:        30,
		FristMedienTage:      7,
		MaxAusleihenSchueler: 8,
		LmfStichtag:          repository.StandardLmfStichtag,
		FerienLeseclubAktiv:  true,
		MaxOverdueDays:       10,
		MaxOverdueItems:      3,
		Sommerferien:         *gesetzt[lmfplan.SommerferienSchluessel],
	}
	if *ist != soll {
		t.Errorf("Einstellungen der Ausleihe:\n  ist  %+v\n  soll %+v", *ist, soll)
	}
}
