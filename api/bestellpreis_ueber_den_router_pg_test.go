package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
	"bibliothek/internal/testdaten"
)

// Den Preis einer Bestellposition über den ganzen Router ändern: Die Bestellansicht nennt die
// Kennung der Position, die Tür nimmt den Preis an, und die Ansicht zeigt ihn danach. Eine
// Kennung, die keine UUID ist, weist die Prüfung der Pfadkennungen ab, bevor die Datenbank fragt.
func TestBestellpreis_UeberDenRouter(t *testing.T) {
	pool := pgtest.Pool(t)
	_, rufe := routertest.MitAufruf(t, pool)
	titel := testdaten.LegeTitelAn(t, pool, "Router-Preis")
	bestellungID := testdaten.LegeBestellungOhneTopfAn(t, pool, "Router-Preis-Haendler", []string{titel})

	type position struct {
		ID          string  `json:"id"`
		Einzelpreis float64 `json:"einzelpreis"`
	}
	lies := func() position {
		t.Helper()
		rec := rufe(t, http.MethodGet, "/api/bestellhistorie/"+bestellungID, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("Bestellansicht: %d %s", rec.Code, rec.Body.String())
		}
		var antwort struct {
			Positionen []position `json:"positionen"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || len(antwort.Positionen) != 1 {
			t.Fatalf("Bestellansicht: %v, %d Positionen — %s", err, len(antwort.Positionen), rec.Body.String())
		}
		return antwort.Positionen[0]
	}

	vorher := lies()
	if vorher.ID == "" {
		t.Fatal("die Bestellansicht nennt die Kennung der Position nicht")
	}
	pfad := "/api/bestellungen/" + bestellungID + "/positionen/" + vorher.ID + "/einzelpreis"
	if rec := rufe(t, http.MethodPut, pfad, `{"einzelpreis":7.5}`); rec.Code != http.StatusOK {
		t.Fatalf("Preis ändern: %d %s", rec.Code, rec.Body.String())
	}
	if nachher := lies(); nachher.Einzelpreis != 7.5 {
		t.Errorf("Einzelpreis nach der Änderung %v, erwartet 7.5", nachher.Einzelpreis)
	}

	rec := rufe(t, http.MethodPut, "/api/bestellungen/"+bestellungID+"/positionen/keine-uuid/einzelpreis", `{"einzelpreis":1}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Kennung in der Adresse") {
		t.Errorf("Kennung, die keine UUID ist: %d %s, erwartet 400 von der Prüfung der Pfadkennungen", rec.Code, rec.Body.String())
	}
}
