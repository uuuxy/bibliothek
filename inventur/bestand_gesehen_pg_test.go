package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Bestand eines vorhandenen Titels ändert sich nur, wenn die Maske den Stand gesehen
// hat, der jetzt gilt (stockGesehen). Ohne den Vergleich schrieb eine länger offene Maske
// ihre Zahl von vorhin zurück: Platz 1 öffnet bei einem Exemplar, Platz 2 erhöht auf sechs,
// Platz 1 speichert — fünf Exemplare ausgesondert, gemeldet „gespeichert". Die Fälle gehen
// über JSON, weil sich „Feld fehlt" und „Feld ist da" nur dort unterscheiden.
func TestBestand_GiltNurMitDemGesehenenStand(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const isbn = "9783551551672"
	loesche := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	handler := &APIHandler{repo: &BookRepository{db: pool}, metadaten: offlineMetadatenClient()}
	sende := func(methode, id string, extra map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		koerper := map[string]any{"isbn": isbn, "title": "Bestandsprobe", "author": "Test", "coverUrl": "/uploads/x.webp", "listenpreis": 9.99}
		for k, v := range extra {
			koerper[k] = v
		}
		daten, err := json.Marshal(koerper)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(methode, "/api/books/"+id, bytes.NewReader(daten))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		if methode == http.MethodPost {
			handler.BearbeiteBuchErstellen(rec, req)
		} else {
			handler.BearbeiteBuchAktualisieren(rec, req)
		}
		return rec
	}

	angelegt := sende(http.MethodPost, "", map[string]any{"stock": 1, "signatur": "Alt 1"})
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

	// {im Bestand, ausgesondert, Signatur}
	type stand struct {
		bestand, ausgesondert int
		signatur              string
	}
	lies := func() stand {
		t.Helper()
		var s stand
		if err := pool.QueryRow(ctx, `
			SELECT (SELECT count(*) FROM buecher_exemplare e WHERE e.titel_id = t.id AND NOT e.ist_ausgesondert),
			       (SELECT count(*) FROM buecher_exemplare e WHERE e.titel_id = t.id AND e.ist_ausgesondert),
			       COALESCE(t.signatur, '')
			FROM buecher_titel t WHERE t.id = $1`, id).Scan(&s.bestand, &s.ausgesondert, &s.signatur); err != nil {
			t.Fatalf("Stand lesen: %v", err)
		}
		return s
	}

	schritte := []struct {
		name    string
		extra   map[string]any
		code    int
		meldung string // Teil der Fehlermeldung, leer bei Erfolg
		erwarte stand
	}{
		{"Platz 2 erhöht von 1 auf 6", map[string]any{"stock": 6, "stockGesehen": 1, "signatur": "Alt 1"},
			http.StatusOK, "", stand{6, 0, "Alt 1"}},
		{"Platz 1 speichert die Signatur, der Bestand geht nicht mit", map[string]any{"signatur": "Neu 1"},
			http.StatusOK, "", stand{6, 0, "Neu 1"}},
		{"Platz 1 ändert den Bestand mit der Zahl von vorhin", map[string]any{"stock": 2, "stockGesehen": 1, "signatur": "Abgelehnt"},
			http.StatusConflict, "jetzt 6 statt 1", stand{6, 0, "Neu 1"}},
		{"eine vor dem Update geladene Seite bestätigt den Stand", map[string]any{"stock": 6, "signatur": "Neu 1"},
			http.StatusOK, "", stand{6, 0, "Neu 1"}},
		{"eine vor dem Update geladene Seite ändert den Bestand", map[string]any{"stock": 1, "signatur": "Abgelehnt"},
			http.StatusConflict, "Seite neu laden", stand{6, 0, "Neu 1"}},
		{"mit dem gesehenen Stand verringert", map[string]any{"stock": 4, "stockGesehen": 6, "signatur": "Neu 1"},
			http.StatusOK, "", stand{4, 2, "Neu 1"}},
		{"eine negative gesehene Zahl", map[string]any{"stock": 4, "stockGesehen": -1, "signatur": "Abgelehnt"},
			http.StatusBadRequest, "stockGesehen", stand{4, 2, "Neu 1"}},
	}
	for _, s := range schritte {
		rec := sende(http.MethodPut, id, s.extra)
		if rec.Code != s.code {
			t.Errorf("%s: Status %d, erwartet %d — %s", s.name, rec.Code, s.code, rec.Body.String())
		}
		if s.meldung != "" && !strings.Contains(rec.Body.String(), s.meldung) {
			t.Errorf("%s: die Antwort nennt %q nicht — %s", s.name, s.meldung, rec.Body.String())
		}
		if s.code == http.StatusConflict {
			var abgelehnt struct {
				Bestand *int `json:"bestand"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &abgelehnt); err != nil || abgelehnt.Bestand == nil || *abgelehnt.Bestand != s.erwarte.bestand {
				t.Errorf("%s: die Ablehnung nennt den Stand %d nicht unter „bestand“ — %s", s.name, s.erwarte.bestand, rec.Body.String())
			}
		}
		if ist := lies(); ist != s.erwarte {
			t.Errorf("%s: im Bestand %d, ausgesondert %d, Signatur %q — erwartet %d, %d und %q",
				s.name, ist.bestand, ist.ausgesondert, ist.signatur, s.erwarte.bestand, s.erwarte.ausgesondert, s.erwarte.signatur)
		}
	}
}
