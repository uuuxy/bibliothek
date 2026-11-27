package bestellung_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
	"bibliothek/internal/testdaten"
)

// Ein bestellter Titel lässt sich vormerken. Kommt die Lieferung, geht das Exemplar wie bei einer
// Rückgabe an den, der wartet: Die Vormerkung wird abholbereit, und die Antwort sagt, für wen das
// Exemplar ins Abholfach gehört. Sonst stünde es frei im Regal, während das Kind weiter wartet.
func TestWareneingang_BedientWartendeVormerkung(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	_, rufe := routertest.MitAufruf(t, pool)
	ctx := t.Context()
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Vorgemerkter Titel", "Vor 1")
	schuelerID := testdaten.LegeSchuelerMitNamenAn(t, pool, "S-WARTE-1", "Anna", "Warte", "07B", 2031)
	if _, err := pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')`,
		titelID, schuelerID); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}
	vorgemerkt := zulaufExemplarMitNotiz(t, pool, titelID, "VOR-A1")
	frei := zulaufExemplarMitNotiz(t, pool, titelID, "VOR-A2")

	rec := rufe(t, http.MethodPost, "/api/bestellungen/bulk-receive", `{"exemplar_ids":["`+vorgemerkt+`","`+frei+`"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Items []struct {
			BarcodeID string `json:"barcode_id"`
			Abholfach string `json:"abholfach"`
		} `json:"received_items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort unlesbar (%s): %v", rec.Body.String(), err)
	}
	abholfach := map[string]string{}
	for _, e := range antwort.Items {
		abholfach[e.BarcodeID] = e.Abholfach
	}
	if len(abholfach) != 2 || abholfach["VOR-A1"]+abholfach["VOR-A2"] != "Anna Warte, 07B" {
		t.Errorf("Abholfach je Exemplar = %v, erwartet genau eines für „Anna Warte, 07B“", abholfach)
	}

	var status string
	var bereit *string
	if err := pool.QueryRow(ctx, `SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen WHERE schueler_id = $1`,
		schuelerID).Scan(&status, &bereit); err != nil {
		t.Fatalf("Vormerkung lesen: %v", err)
	}
	if status != "abholbereit" || bereit == nil || (*bereit != vorgemerkt && *bereit != frei) {
		t.Errorf("Vormerkung: status %q, Exemplar %v; erwartet abholbereit mit einem der eingebuchten Exemplare", status, bereit)
	}
}
