package katalog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// Eine Zeile der Exemplarliste, die sich nicht lesen lässt, ist ein Fehler (500) — kein
// Exemplar, das still von der Buchakte verschwindet. Bis zum 29.09.2026 stand dort
// `if err := rows.Scan(...); err == nil { append }`: Die Liste kam mit 200 und ohne dieses
// Exemplar, und niemand erfuhr davon. Dieselbe Regel wie bei der Rechnung
// (api/schaden/rechnung.go): lieber gar keine Liste als eine, der still eine Zeile fehlt.
func TestTitelExemplare_UnlesbareZeileIstEinFehler(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: mock}}}

	// NULL in einer Spalte, die in string gelesen wird (NULL-Scan-Bugklasse): Der Scan scheitert.
	zeilen := pgxmock.NewRows([]string{"id", "barcode_id", "zustand_notiz", "ist_ausleihbar", "ist_ausgesondert",
		"zustand_abwertung_prozent", "ist_verfuegbar", "eigentum", "eigentum_herkunft", "littera_eigentumsvermerk"}).
		AddRow("e1", "B-1", "", true, false, 0, true, "land", "vorgabe", "").
		AddRow("e2", nil, "", true, false, 0, true, "land", "vorgabe", "")
	mock.ExpectQuery("SELECT e.id, e.barcode_id").WithArgs("t1").WillReturnRows(zeilen)

	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/t1/exemplare", nil)
	req.SetPathValue("id", "t1")
	rec := httptest.NewRecorder()
	srv.TitelExemplare(repository.NewBescheidAbfragen(mock))(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500 — eine unlesbare Zeile fiel still weg: %s", rec.Code, rec.Body.String())
	}
}
