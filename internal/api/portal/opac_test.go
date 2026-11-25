package portal

import (
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"

	"github.com/pashagolub/pgxmock/v5"
)

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
