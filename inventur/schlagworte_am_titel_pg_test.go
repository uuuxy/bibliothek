package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Schlagworte gehen im Buchformular mit dem ganzen Titel hin und her (Migration 138):
// GET /api/books/{id} lädt sie, PUT /api/books/{id} schickt sie zurück. Drei Eingaben
// müssen sich dabei unterscheiden — und genau das sieht nur ein Test über JSON:
//
//   - Feld fehlt oder ist null: Der Aufrufer kennt es nicht (Scanner, ältere Clients) —
//     die vorhandenen Schlagworte bleiben. Sonst löschte jeder Aufrufer ohne das Feld sie.
//   - `[]`: Der Mensch hat alle entfernt — sie fallen.
//   - Liste: genau diese, in der vorhandenen Schreibweise.
//
// Dazu die andere Richtung: Ein Titel OHNE Schlagworte kommt im Einzel-Read als [] an,
// nicht als null. Die Maske schickt zurück, was sie gelesen hat.
func TestSchlagworte_BuchformularUnterscheidetFehlendLeerUndListe(t *testing.T) {
	pool := pgtest.Pool(t)
	const isbn = "9780306406157"
	loesche := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	handler := &APIHandler{repo: &BookRepository{db: pool}, metadaten: offlineMetadatenClient()}

	sende := func(methode string, id string, koerper map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		daten, err := json.Marshal(koerper)
		if err != nil {
			t.Fatal(err)
		}
		pfad := "/api/books"
		if id != "" {
			pfad += "/" + id
		}
		req := httptest.NewRequest(methode, pfad, bytes.NewReader(daten))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		if methode == http.MethodPost {
			handler.BearbeiteBuchErstellen(rec, req)
		} else {
			handler.BearbeiteBuchAktualisieren(rec, req)
		}
		return rec
	}
	// lies holt die Schlagworte so, wie die Maske sie bekommt: als JSON des Einzel-Reads.
	lies := func(id string) []string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/books/"+id, nil)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		handler.BearbeiteBuchLesen(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Einzel-Read: %d %s", rec.Code, rec.Body.String())
		}
		var buch struct {
			Schlagworte *[]string `json:"schlagworte"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &buch); err != nil {
			t.Fatal(err)
		}
		if buch.Schlagworte == nil {
			t.Fatalf("Einzel-Read liefert schlagworte = null — die Maske schickte „nichts gesagt“ zurück: %s", rec.Body.String())
		}
		return *buch.Schlagworte
	}
	titel := func(extra map[string]any) map[string]any {
		koerper := map[string]any{"isbn": isbn, "title": "Drachenreiter", "author": "Cornelia Funke"}
		for k, v := range extra {
			koerper[k] = v
		}
		return koerper
	}

	angelegt := sende(http.MethodPost, "", titel(map[string]any{"schlagworte": []string{"Fantasy", "fantasy", "  Drachen "}}))
	if angelegt.Code != http.StatusCreated {
		t.Fatalf("Anlegen: %d %s", angelegt.Code, angelegt.Body.String())
	}
	var antwort struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(angelegt.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	id := antwort.Data.ID

	schritte := []struct {
		name    string
		extra   map[string]any
		code    int
		erwarte []string
	}{
		{"nach dem Anlegen", nil, 0, []string{"Drachen", "Fantasy"}},
		{"Feld fehlt", map[string]any{}, http.StatusOK, []string{"Drachen", "Fantasy"}},
		{"Feld ist null", map[string]any{"schlagworte": nil}, http.StatusOK, []string{"Drachen", "Fantasy"}},
		{"neue Liste", map[string]any{"schlagworte": []string{"Drachen", "Freundschaft"}}, http.StatusOK, []string{"Drachen", "Freundschaft"}},
		{"zu viele", map[string]any{"schlagworte": zuVieleSchlagworte()}, http.StatusBadRequest, []string{"Drachen", "Freundschaft"}},
		{"leere Liste", map[string]any{"schlagworte": []string{}}, http.StatusOK, []string{}},
	}
	for _, s := range schritte {
		if s.extra != nil {
			rec := sende(http.MethodPut, id, titel(s.extra))
			if rec.Code != s.code {
				t.Fatalf("%s: Status %d, erwartet %d: %s", s.name, rec.Code, s.code, rec.Body.String())
			}
		}
		if got := lies(id); !slices.Equal(got, s.erwarte) {
			t.Errorf("%s: Schlagworte %q, erwartet %q", s.name, got, s.erwarte)
		}
	}
}

func zuVieleSchlagworte() []string {
	woerter := make([]string, 31)
	for i := range woerter {
		woerter[i] = fmt.Sprintf("Wort %d", i)
	}
	return woerter
}

// Der Text eines 400 muss sagen, was falsch ist — die Maske zeigt ihn als Meldung.
func TestSchlagworte_ZuLangesWortNenntDieGrenze(t *testing.T) {
	woerter, err := schlagworteAusEingabe(&[]string{strings.Repeat("x", 81)})
	if err == nil {
		t.Fatalf("81 Zeichen angenommen: %q", woerter)
	}
	if !strings.Contains(err.Error(), "80 Zeichen") {
		t.Errorf("Meldung %q nennt die Grenze nicht", err.Error())
	}
}
