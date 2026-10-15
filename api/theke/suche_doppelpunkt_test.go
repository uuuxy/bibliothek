package theke

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Die Suche antwortet auf eine Anfrage mit Doppelpunkt, ohne zu stolpern.
//
// Der Test nimmt denselben Weg wie jeder andere Postgres-Test im Haus: `pgtest.Pool`
// überspringt nur, wenn TEST_DATABASE_URL fehlt. Eine feste Adresse auf die
// Entwicklungs-Datenbank, die sich bei fehlender Antwort selbst überspränge, liefe auf jedem
// anderen Rechner und in der CI still nie: ein Test, der grün aussieht, weil er gar nicht
// stattfindet.
func TestSuche_AnfrageMitDoppelpunkt(t *testing.T) {
	pool := pgtest.Pool(t)
	database := &db.Verbindung{Pool: pool}

	srv := &Tueren{Basis: &tuer.Basis{DB: database}}
	handler := srv.Suche(repository.NewLeserAbfragen(pool), repository.NewKatalogAbfragen(pool))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/search?q=max:1", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200. Rumpf: %s", rr.Code, rr.Body.String())
	}
	// Die Antwort muss lesbar sein — ein 200 mit kaputtem Rumpf wäre kein Erfolg.
	var beliebig any
	if err := json.Unmarshal(rr.Body.Bytes(), &beliebig); err != nil {
		t.Fatalf("Antwort ist kein JSON: %v — Rumpf: %s", err, rr.Body.String())
	}
}
