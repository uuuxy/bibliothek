package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"bibliothek/repository"
)

// GET /api/buecher/titel/suche — die Titel-Tür für Bildschirme außerhalb der Theke
// (Etiketten-Titelsuche im Druck-Center; Befund-Register, Entscheidung 3 vom 05.09.2026).
//
// Bis dahin rief der Druckbildschirm GET /api/search (perform_actions): dieselbe Suche,
// aber mit Schüler-Kiosk-Daten in der Antwort, die er nie zeigte — und gekoppelt an das
// Theken-Recht, obwohl der Bildschirm nur Titel braucht. Die Suchgüte muss die der Theke
// bleiben (SearchTitlesFuzzy: Signatur, ISBN, Umlaut-Normalisierung) — die Inventur-Liste
// GET /api/books kann das nicht, deshalb ist das eine eigene Tür und kein Umzug dorthin.
func TestTitelSuche_LiefertNurTitelMitThekenSuchguete(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := titelMitSignatur(t, pool, "Türsuche Testband", "TUER 7", 1)
	// Ein Schüler mit demselben Suchwort — er darf in der Antwort nicht auftauchen.
	if _, err := pool.Exec(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('S-TUER-1', 'Tilda', 'Türsuche', '07A', 2031)`); err != nil {
		t.Fatal(err)
	}

	s := &Server{}
	h := s.TitelSucheHandler(repository.NewBookRepository(pool))
	rufe := func(q string) (int, string, TitelSucheResult) {
		t.Helper()
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/api/buecher/titel/suche?q="+url.QueryEscape(q), nil))
		var res TitelSucheResult
		if w.Code == http.StatusOK {
			if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
				t.Fatal(err)
			}
		}
		return w.Code, w.Body.String(), res
	}

	// Freitext: der Titel, und KEIN Schüler im Körper — weder als Feld noch als Wert.
	code, body, res := rufe("Türsuche")
	if code != http.StatusOK || len(res.Books) != 1 || res.Books[0].ID != titelID {
		t.Errorf("Freitext: HTTP %d, %d Titel (%s)", code, len(res.Books), body)
	}
	if strings.Contains(body, "Tilda") || strings.Contains(body, "students") || strings.Contains(body, "S-TUER-1") {
		t.Errorf("die Titel-Tür trägt Schülerdaten: %s", body)
	}

	// Signatur: die Suchgüte der Theke, die GET /api/books nicht hätte.
	if _, body, r := rufe("TUER 7"); len(r.Books) != 1 {
		t.Errorf("Signatur-Suche: %d Treffer (%s)", len(r.Books), body)
	}

	// Leeres q: 400, wie an der Theke.
	if code, body, _ := rufe(""); code != http.StatusBadRequest {
		t.Errorf("leeres q: HTTP %d statt 400 (%s)", code, body)
	}
}
