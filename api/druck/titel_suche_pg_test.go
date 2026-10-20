package druck

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// GET /api/buecher/titel/suche sucht Titel für Bildschirme außerhalb der Theke, etwa die
// Etiketten im Druck-Center. Sie liefert keine Schüler und hängt nicht am Recht der Theke
// (perform_actions). Die Suchgüte bleibt die der Theke (KatalogAbfragen.FindeTitelUnscharf:
// Signatur, ISBN, Umlaut-Normalform); die Liste GET /api/books kann das nicht.
func TestTitelSuche_LiefertNurTitelMitThekenSuchguete(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Türsuche Testband", "TUER 7", 1)
	testdaten.LegeExemplarAn(t, pool, titelID, "B-TUER-1", true, "") // ohne Exemplar zeigt die Tür keinen Titel
	// Ein Schüler mit demselben Suchwort — er darf in der Antwort nicht auftauchen.
	testdaten.LegeSchuelerMitNamenAn(t, pool, "S-TUER-1", "Tilda", "Türsuche", "07A", 2031)

	s := &Tueren{Basis: &tuer.Basis{}}
	h := s.TitelSuche(repository.NewKatalogAbfragen(pool))
	rufe := func(q string) (int, string, titelSucheAntwort) {
		t.Helper()
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/api/buecher/titel/suche?q="+url.QueryEscape(q), nil))
		var res titelSucheAntwort
		if w.Code == http.StatusOK {
			if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
				t.Fatal(err)
			}
		}
		return w.Code, w.Body.String(), res
	}

	// Freitext: der Titel, und kein Schüler im Körper — weder als Feld noch als Wert.
	code, body, res := rufe("Türsuche")
	if code != http.StatusOK || len(res.Titel) != 1 || res.Titel[0].ID != titelID {
		t.Errorf("Freitext: HTTP %d, %d Titel (%s)", code, len(res.Titel), body)
	}
	if strings.Contains(body, "Tilda") || strings.Contains(body, "students") || strings.Contains(body, "S-TUER-1") {
		t.Errorf("die Titel-Tür trägt Schülerdaten: %s", body)
	}

	// Signatur: die Suchgüte der Theke, die GET /api/books nicht hätte.
	if _, body, r := rufe("TUER 7"); len(r.Titel) != 1 {
		t.Errorf("Signatur-Suche: %d Treffer (%s)", len(r.Titel), body)
	}

	// Leeres q: 400, wie an der Theke.
	if code, body, _ := rufe(""); code != http.StatusBadRequest {
		t.Errorf("leeres q: HTTP %d statt 400 (%s)", code, body)
	}
}
