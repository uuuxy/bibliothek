package portal

import (
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v5"
)

// Die OPAC-Suche schickt die Eingabe zweimal: roh für die Volltextsuche, maskiert für die
// ILIKE-Vergleiche. Mit nur einem Argument trifft die Erwartung nicht.
func TestOPACSuche_LikeJokerWerdenMaskiert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("FROM buecher_titel bt").
		WithArgs("%", `\%`).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "cover_url", "verfuegbar", "gesamt"}))

	s := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: mock}}}
	rec := httptest.NewRecorder()
	s.OPACSuche()(rec, httptest.NewRequest("GET", "/api/public/opac/suche?q=%25", nil))
	if rec.Code != 200 {
		t.Fatalf("Status = %d, Body %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
