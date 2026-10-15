package schaden

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/google/uuid"
)

// Der Schadensersatz folgt dem Eigentum: wem die Forderung zusteht, nach welcher Regel ihr
// Betrag vorgeschlagen wird und auf welches Konto der Brief verweist. ist_lernmittel des Titels
// allein entschiede falsch.
//
// Die beiden Fälle, in denen das auseinanderläuft: eine Lektüre aus LMF-Mitteln (kein
// Lernmittel-Titel, Littera sagt „Land Hessen") und ein Schulbuch, das der Schulträger
// bezahlt hat (Lernmittel-Titel, Bestellung aus seinen Mitteln).
func TestSchadensersatz_FolgtDemEigentum(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	testdaten.LeereBescheide(t, pool)
	testdaten.SetzeBescheidAngaben(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	// Gleiche Preise, gleiches Alter: Der Unterschied im Betrag kommt allein aus der Regel.
	vorbereiten := func(titelID, barcode string) string {
		t.Helper()
		id := testdaten.LegeExemplarAn(t, pool, titelID, barcode, true, "")
		if _, err := pool.Exec(ctx, `
			UPDATE buecher_exemplare SET einkaufspreis = 8.00,
			       zugang_am = CURRENT_DATE - INTERVAL '10 years', erworben_am = CURRENT_DATE - INTERVAL '10 years'
			WHERE id = $1`, id); err != nil {
			t.Fatalf("Exemplar %s vorbereiten: %v", barcode, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET listenpreis = 14.00 WHERE id = $1`, titelID); err != nil {
			t.Fatalf("Listenpreis: %v", err)
		}
		return id
	}

	lektuere := testdaten.LegeTitelMitCoverUndAlterAn(t, pool, "Nathan der Weise", "Ga Les", true, 0)
	exLektuere := vorbereiten(lektuere, "EB-LEKTUERE")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET eigentum = 'land', eigentum_quelle = 'littera' WHERE id = $1`, exLektuere); err != nil {
		t.Fatalf("Eigentum setzen: %v", err)
	}
	schulbuch := testdaten.LegeLernmittelAn(t, pool, "Mathematik 7")
	exSchulbuch := vorbereiten(schulbuch, "EB-SCHULBUCH")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET bestellung_id = $2 WHERE id = $1`,
		exSchulbuch, testdaten.LegeBestellungAn(t, mitteltopf.Schultraeger)); err != nil {
		t.Fatalf("Bestellung zuordnen: %v", err)
	}

	sidLand := testdaten.LegeSchuelerAn(t, pool, "S-EB-LAND", "Landkind", "08G2")
	sidTraeger := testdaten.LegeSchuelerAn(t, pool, "S-EB-TRAEGER", "Traegerkind", "08G2")
	fLand := testdaten.LegeForderungAn(t, pool, sidLand, exLektuere, "beschaedigt", "Lektüre beschädigt")
	fTraeger := testdaten.LegeForderungAn(t, pool, sidTraeger, exSchulbuch, "beschaedigt", "Schulbuch beschädigt")

	// 1. Der Vorschlag: Topf und Regel.
	vLand := bescheidVorschlagUeberTuer(t, srv, pool, sidLand)
	if len(vLand.Positionen) != 1 || vLand.Positionen[0].Topf != mitteltopf.Land ||
		!strings.Contains(vLand.Positionen[0].Herleitung, "Verleihjahr") {
		t.Errorf("Lektüre des Landes: %+v — erwartet Topf land und die Staffel", vLand.Positionen)
	}
	vTraeger := bescheidVorschlagUeberTuer(t, srv, pool, sidTraeger)
	if len(vTraeger.Positionen) != 1 || vTraeger.Positionen[0].Topf != mitteltopf.Schultraeger ||
		strings.Contains(vTraeger.Positionen[0].Herleitung, "Verleihjahr") {
		t.Errorf("Schulbuch des Schulträgers: %+v — erwartet Topf schultraeger und den Neuwert", vTraeger.Positionen)
	}

	// 2. Der Reiter „Schadensersatz": Wer einen Landes-Bescheid bekommen kann.
	zeilen, err := repository.NewBescheidAbfragen(pool).LadeForderungenOhneBescheid(ctx)
	if err != nil {
		t.Fatalf("LadeForderungenOhneBescheid: %v", err)
	}
	land := map[string]bool{}
	for _, z := range zeilen {
		land[z.SchuelerID] = z.Land
	}
	if !land[sidLand] || land[sidTraeger] {
		t.Errorf("LadeForderungenOhneBescheid: Land %v bei der Lektüre, %v beim Schulbuch des Trägers — erwartet true, false",
			land[sidLand], land[sidTraeger])
	}

	// 3. Rechnung und Elternbrief nennen das Konto nach dem Eigentum.
	for sid, want := range map[string]bool{sidLand: true, sidTraeger: false} {
		items, err := ladeRechnungPosten(ctx, pool, uuid.MustParse(sid))
		if err != nil || len(items) != 1 || items[0].Land != want {
			t.Errorf("Rechnung %s: %+v (%v) — erwartet Land=%v", sid, items, err, want)
		}
	}
	for fid, want := range map[string]bool{fLand: true, fTraeger: false} {
		info, _, err := srv.ladeSchadensfallBrief(ctx, fid)
		if err != nil || info.Land != want {
			t.Errorf("Elternbrief %s: Land=%v (%v), erwartet %v", fid, info.Land, err, want)
		}
	}

	// 4. Der Brief: Die Lektüre darf auf den Landes-Bescheid, das Schulbuch des Trägers nicht.
	if rec := bescheidErstellenUeberTuer(t, srv, pool, sidLand,
		testdaten.BescheidRumpf(testdaten.In28Tagen(), map[string]float64{fLand: 1.40})); rec.Code != http.StatusCreated {
		t.Errorf("Landes-Bescheid für die Lektüre: Status %d, want 201: %s", rec.Code, rec.Body.String())
	}
	if rec := bescheidErstellenUeberTuer(t, srv, pool, sidTraeger,
		testdaten.BescheidRumpf(testdaten.In28Tagen(), map[string]float64{fTraeger: 14.00})); rec.Code != http.StatusConflict {
		t.Errorf("Landes-Bescheid für ein Buch des Schulträgers: Status %d, want 409: %s", rec.Code, rec.Body.String())
	}
}
