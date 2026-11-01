package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"

	"github.com/google/uuid"
)

// Die Rechnung nennt den Bescheid als Grund, wenn nichts mehr offen abzurechnen ist. Das stimmt nur
// für eine unbezahlte Forderung dieses Lesers, die auf einem Bescheid steht: Sonst hieße es bei einem
// Kind ohne Forderung, seine offenen Forderungen stünden auf einem Bescheid.
func TestHatOffeneForderungAufBescheid(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	testdaten.LeereBescheide(t, pool)
	ctx := context.Background()
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	titel := testdaten.LegeLernmittelAn(t, pool, "Bescheid-Testband")
	repo := NewBescheidAbfragen(pool)

	forderung := func(t *testing.T, kind, barcode string, aufBescheid bool) string {
		t.Helper()
		id := testdaten.LegeForderungAn(t, pool, kind, testdaten.LegeExemplarAn(t, pool, titel, barcode, true, ""), string(SchadensArtNichtZurueck), barcode)
		if aufBescheid {
			if _, err := repo.LegeAn(ctx, BescheidEingabe{
				SchuelerID: kind, Mittel: "land", Kassenjahr: time.Now().Year(), FristBis: time.Now().AddDate(0, 0, 28),
				Positionen:     []BescheidPositionEingabe{{SchadensfallID: id, Betrag: 1200}},
				Empfaenger:     map[string]string{"name": "Testkind"},
				ErstelltVon:    bearbeiter,
				Referenznummer: func(nr int) string { return barcode },
			}); err != nil {
				t.Fatalf("Bescheid %s: %v", barcode, err)
			}
		}
		return id
	}

	mitBescheid := testdaten.LegeSchuelerAn(t, pool, "S-AUFB-1", "Mitbescheid", "07B")
	forderung(t, mitBescheid, "AUFB-1", true)
	ohneBescheid := testdaten.LegeSchuelerAn(t, pool, "S-AUFB-2", "Ohnebescheid", "07B")
	forderung(t, ohneBescheid, "AUFB-2", false)
	bezahlt := testdaten.LegeSchuelerAn(t, pool, "S-AUFB-3", "Bezahlt", "07B")
	if _, err := pool.Exec(ctx, `UPDATE schadensfaelle SET ist_bezahlt = true WHERE id = $1`, forderung(t, bezahlt, "AUFB-3", true)); err != nil {
		t.Fatal(err)
	}
	ohneForderung := testdaten.LegeSchuelerAn(t, pool, "S-AUFB-4", "Ohneforderung", "07B")

	for _, f := range []struct {
		name string
		kind string
		soll bool
	}{
		{"unbezahlt auf einem Bescheid", mitBescheid, true},
		{"unbezahlt ohne Bescheid", ohneBescheid, false},
		{"bezahlt auf einem Bescheid", bezahlt, false},
		{"gar keine Forderung", ohneForderung, false},
	} {
		ist, err := HatOffeneForderungAufBescheid(ctx, pool, uuid.MustParse(f.kind))
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		if ist != f.soll {
			t.Errorf("%s: %v, erwartet %v", f.name, ist, f.soll)
		}
	}
}
