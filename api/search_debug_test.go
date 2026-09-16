package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Die Suche antwortet auf eine Anfrage mit Doppelpunkt, ohne zu stolpern.
//
// Bis zum 17.09.2026 hing dieser Test an einer FESTEN Adresse auf die Entwicklungs-Datenbank
// (Port 5434, Passwort im Quelltext) und übersprang sich selbst, sobald sie nicht antwortete.
// Auf jedem anderen Rechner — und in der CI — lief er damit stillschweigend nie: ein Test,
// der grün aussieht, weil er gar nicht stattfindet (OFFEN.md 5.10).
//
// Jetzt derselbe Weg wie jeder andere Postgres-Test im Haus: `pgtest.Pool` überspringt nur,
// wenn TEST_DATABASE_URL fehlt, und das ist eine sichtbare Entscheidung statt eines Zufalls.
func TestSuche_AnfrageMitDoppelpunkt(t *testing.T) {
	pool := pgtest.Pool(t)
	database := &db.Database{Pool: pool}

	srv := NewServer(database, nil, nil, false)
	handler := srv.SearchHandler(repository.NewStudentRepository(pool), repository.NewBookRepository(pool))

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
