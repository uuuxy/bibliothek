package katalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Die Tür des Status-Editors prüft die Ausleihe wie das Aussondern über DELETE.
//
// „Ausgesondert" erreicht man auf zwei Wegen: über DELETE (ExemplarAussondern) und über
// PUT …/status (ExemplarStatusAendern). Ließe einer davon ein verliehenes Buch aussondern,
// verschwände es aus Katalog, Theke und Inventur, der Schüler bliebe in der Mahnstrecke, und bei
// der Rückgabe wäre das Exemplar gesperrt, ohne dass jemand nachvollziehen könnte, wer es
// ausgesondert hat. Dieser Test hält die Schranke am Status-Editor, dazu 404 für eine
// unbekannte Kennung.
func TestAussondern_StatusTuerPrueftAusleihe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	bookRepo := repository.NewKatalogAbfragen(pool)

	titelID := testdaten.LegeTitelMitCoverUndAlterAn(t, pool, "Tueren-Titel", "Jug Tue", false, 0)
	verliehen := testdaten.LegeExemplarAn(t, pool, titelID, "TUER-VERLIEHEN", true, "")
	frei := testdaten.LegeExemplarAn(t, pool, titelID, "TUER-FREI", true, "")
	sid := testdaten.LegeSchuelerAn(t, pool, "S-TUER-1", "Mia", "5a")
	testdaten.LegeAusleiheVorZweiTagenAn(t, pool, verliehen, sid)

	statusAussondern := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+id+"/status",
			strings.NewReader(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"weg damit"}`))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		// Türen mit Datenbank: Der Erfolgsfall fragt den Ersatzwert nach.
		(&Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}).ExemplarStatusAendern(bookRepo, repository.NewBescheidAbfragen(pool))(rec, tuertest.AlsKonto(req, testdaten.LegeBearbeiterAn(t, pool)))
		return rec
	}
	istAusgesondert := func(id string) bool {
		var aus bool
		if err := pool.QueryRow(context.Background(),
			`SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = $1`, id).Scan(&aus); err != nil {
			t.Fatalf("ist_ausgesondert lesen: %v", err)
		}
		return aus
	}

	// PUT /status mit ist_ausgesondert=true auf ein verliehenes Exemplar — abgelehnt.
	if rec := statusAussondern(verliehen); rec.Code != http.StatusBadRequest {
		t.Errorf("status→ausgesondert verliehen: HTTP %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if istAusgesondert(verliehen) {
		t.Error("die Status-Tür hat ein verliehenes Exemplar ausgesondert")
	}

	// Unbekannte ID: 404 statt stillem Erfolg.
	unbekannt := "00000000-0000-0000-0000-000000000d0d"
	if rec := statusAussondern(unbekannt); rec.Code != http.StatusNotFound {
		t.Errorf("status unbekannt: HTTP %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}

	// Freies Exemplar: die Tür funktioniert weiter.
	if rec := statusAussondern(frei); rec.Code != http.StatusOK {
		t.Fatalf("status→ausgesondert frei: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if !istAusgesondert(frei) {
		t.Error("freies Exemplar wurde nicht ausgesondert")
	}
}
