package portal

import (
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"

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

	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
	rec := httptest.NewRecorder()
	s.OPACSuche()(rec, httptest.NewRequest("GET", "/api/public/opac/suche?q=%25", nil))
	if rec.Code != 200 {
		t.Fatalf("Status = %d, Body %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Ein Suchtext über der Grenze fragt die Datenbank nicht: Die Tür steht ohne Anmeldung offen,
// und kein Titel, Autor oder Schlagwort ist so lang.
func TestOPACSuche_ZuLangerSuchtextFragtDieDatenbankNicht(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
	rec := httptest.NewRecorder()
	s.OPACSuche()(rec, httptest.NewRequest("GET", "/api/public/opac/suche?q="+strings.Repeat("ä", opacSuchtextMax+1), nil))
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("Status %d, Antwort %q, erwartet 200 mit leerer Liste", rec.Code, rec.Body.String())
	}
}
