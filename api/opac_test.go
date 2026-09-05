package api

import (
	"net/http/httptest"
	"testing"

	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

func TestMaskiereLikeJoker(t *testing.T) {
	faelle := map[string]string{
		"Harry":      "Harry",
		"100%":       `100\%`,
		"a_b":        `a\_b`,
		`back\slash`: `back\\slash`,
		"%_%":        `\%\_\%`,
	}
	for in, will := range faelle {
		if got := maskiereLikeJoker(in); got != will {
			t.Errorf("maskiereLikeJoker(%q) = %q; want %q", in, got, will)
		}
	}
}

// Die OPAC-Suche schickt die Eingabe zweimal: roh für die Volltextsuche, maskiert für
// die ILIKE-Vergleiche. Mit dem alten Code (ein Argument) trifft die Erwartung nicht.
func TestPublicCatalogSearch_LikeJokerWerdenMaskiert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("FROM buecher_titel bt").
		WithArgs("%", `\%`).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "cover_url", "verfuegbar", "gesamt"}))

	s := &Server{DB: &db.Database{Pool: mock}}
	rec := httptest.NewRecorder()
	s.PublicCatalogSearchHandler()(rec, httptest.NewRequest("GET", "/api/public/opac/suche?q=%25", nil))
	if rec.Code != 200 {
		t.Fatalf("Status = %d, Body %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
