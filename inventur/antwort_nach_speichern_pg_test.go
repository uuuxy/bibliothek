package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Antwort auf Anlegen und Ändern ist der Titel, wie er gespeichert ist, samt Bestand.
// Die Titelliste ersetzt ihre Zeile durch diese Antwort; aus den gesendeten Angaben allein
// stand dort nach jedem Speichern der Bestand 0, bis die Liste neu geladen wurde.
func TestSpeichern_AntwortTraegtDenBestand(t *testing.T) {
	pool := pgtest.Pool(t)
	const isbn = "9783423715669"
	loesche := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	handler := &APIHandler{repo: &BookRepository{db: pool}, metadaten: offlineMetadatenClient()}
	type antwort struct {
		Data struct {
			ID          string    `json:"id"`
			Stock       int       `json:"stock"`
			Gesamt      int       `json:"gesamt"`
			Verfuegbar  int       `json:"verfuegbar"`
			Signatur    string    `json:"signatur"`
			Schlagworte *[]string `json:"schlagworte"`
		} `json:"data"`
	}
	sende := func(methode, id string, extra map[string]any) antwort {
		t.Helper()
		koerper := map[string]any{"isbn": isbn, "title": "Antwortprobe", "author": "Test", "coverUrl": "/uploads/x.webp", "listenpreis": 9.99}
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
		if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("%s: %d %s", methode, rec.Code, rec.Body.String())
		}
		var a antwort
		if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	pruefe := func(schritt string, a antwort, signatur string) {
		t.Helper()
		d := a.Data
		if d.Stock != 2 || d.Gesamt != 2 || d.Verfuegbar != 2 {
			t.Errorf("%s: Antwort nennt stock %d, gesamt %d, verfügbar %d — der Titel hat zwei Exemplare",
				schritt, d.Stock, d.Gesamt, d.Verfuegbar)
		}
		if d.Signatur != signatur {
			t.Errorf("%s: Signatur %q, erwartet %q", schritt, d.Signatur, signatur)
		}
		if d.Schlagworte == nil {
			t.Errorf("%s: schlagworte ist null — die Antwort ist nicht der gespeicherte Titel", schritt)
		}
	}

	angelegt := sende(http.MethodPost, "", map[string]any{"stock": 2, "signatur": "Ant 1"})
	pruefe("Anlegen", angelegt, "Ant 1")
	geaendert := sende(http.MethodPut, angelegt.Data.ID, map[string]any{"signatur": "Ant 2"})
	pruefe("Ändern ohne Bestand", geaendert, "Ant 2")
}
