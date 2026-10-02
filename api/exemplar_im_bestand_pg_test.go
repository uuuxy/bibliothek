package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Die Exemplarliste sagt je Exemplar, ob es zum Bestand des Titels zählt — mit derselben Regel,
// nach der das Feld „Aktueller Bestand" zählt (repository.SQLExemplarImBestand). Die Buchmaske
// listet nur diese Exemplare; ausgesonderte und bestellte führte sie sonst als „Gesperrt" mit,
// und über der Liste stand eine andere Zahl als im Feld.
func TestExemplarliste_SagtWasZumBestandZaehlt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	ctx := context.Background()

	titel := titelMitMeldebestand(t, pool, "Im Bestand", 1)
	exemplar(t, pool, titel, "BC-REGAL", true, "")
	gesperrt := exemplar(t, pool, titel, "BC-GESPERRT", true, "")
	ausgesondert := exemplar(t, pool, titel, "BC-AUSGESONDERT", true, "")
	bestellt := exemplar(t, pool, titel, "BC-BESTELLT", true, "")
	zulauf := exemplar(t, pool, titel, "BC-ZULAUF", true, "")
	for anweisung, id := range map[string]string{
		`UPDATE buecher_exemplare SET ist_ausleihbar = false WHERE id = $1`:                                                              gesperrt,
		`UPDATE buecher_exemplare SET ist_ausgesondert = true, aussonderung_grund = 'AUSSORTIERT', ist_ausleihbar = false WHERE id = $1`: ausgesondert,
		`UPDATE buecher_exemplare SET bestellstatus = 'bestellt', ist_ausleihbar = false WHERE id = $1`:                                  bestellt,
		`UPDATE buecher_exemplare SET bestellstatus = 'im_zulauf', ist_ausleihbar = false WHERE id = $1`:                                 zulauf,
	} {
		if _, err := pool.Exec(ctx, anweisung, id); err != nil {
			t.Fatalf("%s: %v", anweisung, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/"+titel+"/exemplare", nil)
	req.SetPathValue("id", titel)
	rec := httptest.NewRecorder()
	srv.GetTitleCopiesHandler(repository.NewBescheidRepository(pool))(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Exemplarliste: Status %d, Body %s", rec.Code, rec.Body.String())
	}
	var zeilen []struct {
		BarcodeID string `json:"barcode_id"`
		ImBestand *bool  `json:"im_bestand"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &zeilen); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}

	erwartet := map[string]bool{
		"BC-REGAL": true, "BC-GESPERRT": true,
		"BC-AUSGESONDERT": false, "BC-BESTELLT": false, "BC-ZULAUF": false,
	}
	if len(zeilen) != len(erwartet) {
		t.Fatalf("%d Exemplare in der Antwort, erwartet %d: die Liste nennt weiter alle", len(zeilen), len(erwartet))
	}
	imBestand := 0
	for _, z := range zeilen {
		if z.ImBestand == nil {
			t.Fatalf("%s: die Antwort trägt kein im_bestand", z.BarcodeID)
		}
		if *z.ImBestand != erwartet[z.BarcodeID] {
			t.Errorf("%s: im_bestand = %v, erwartet %v", z.BarcodeID, *z.ImBestand, erwartet[z.BarcodeID])
		}
		if *z.ImBestand {
			imBestand++
		}
	}

	// Dieselbe Zahl wie das Feld der Maske.
	var bestand int
	if err := pool.QueryRow(ctx, `SELECT `+repository.SQLBestandGesamt+` FROM buecher_titel b WHERE b.id = $1`, titel).Scan(&bestand); err != nil {
		t.Fatal(err)
	}
	if imBestand != bestand {
		t.Errorf("die Liste zählt %d Exemplare zum Bestand, das Feld der Maske %d", imBestand, bestand)
	}
}
