package katalog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Ein neuer Titel ohne Bestandsangabe hat kein Exemplar und steht deshalb in keinem Katalog
// (repository.sqlTitelHatExemplar). Wer dieselbe ISBN danach noch einmal anlegt, bekommt die
// Ablehnung der Dublettenkontrolle — und fand den Titel bis dahin über keine Suche. Die
// Antwort nennt deshalb den Titel, seine Kennung und ob er ohne Exemplar ist: Die Maske
// führt damit zu ihm.
func TestTitelAnlegen_DubletteNenntDenTitelUndWoErSteht(t *testing.T) {
	pool := pgtest.Pool(t)
	const isbn = "9780306406164"
	const titel = "Dubletten-Probe ohne Exemplar"
	loesche := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}, Katalogdienste: offlineMetadatenClient()}

	lege := func(bestand int) *httptest.ResponseRecorder {
		t.Helper()
		daten, err := json.Marshal(map[string]any{"isbn": isbn, "title": titel, "author": "Probe", "stock": bestand})
		if err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		tueren.TitelAnlegen()(rec, httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(daten)))
		return rec
	}
	// treffer zählt, wie oft die Liste die ISBN zeigt — so, wie die Titelliste sie abfragt.
	treffer := func(sicht string) int {
		t.Helper()
		rec := httptest.NewRecorder()
		tueren.TitelListe()(rec, httptest.NewRequest(http.MethodGet, "/api/books?q="+isbn+sicht, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Liste: %d %s", rec.Code, rec.Body.String())
		}
		var antwort struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatal(err)
		}
		return len(antwort.Data)
	}
	type dublette struct {
		Error     string `json:"error"`
		Vorhanden *struct {
			ID           string `json:"id"`
			Title        string `json:"title"`
			OhneExemplar bool   `json:"ohneExemplar"`
		} `json:"vorhanden"`
	}
	abgelehnt := func() dublette {
		t.Helper()
		rec := lege(0)
		if rec.Code != http.StatusConflict {
			t.Fatalf("zweites Anlegen: %d %s, erwartet 409", rec.Code, rec.Body.String())
		}
		var d dublette
		if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
			t.Fatal(err)
		}
		if d.Vorhanden == nil {
			t.Fatalf("die Ablehnung nennt den vorhandenen Titel nicht: %s", rec.Body.String())
		}
		return d
	}

	erster := lege(0)
	if erster.Code != http.StatusCreated {
		t.Fatalf("erstes Anlegen: %d %s", erster.Code, erster.Body.String())
	}
	var angelegt struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(erster.Body.Bytes(), &angelegt); err != nil {
		t.Fatal(err)
	}

	if n := treffer(""); n != 0 {
		t.Fatalf("Katalog zeigt den Titel ohne Exemplar %d-mal — die Probe stellt den Anlass nicht mehr nach", n)
	}
	if n := treffer("&bestand=ohne"); n != 1 {
		t.Fatalf("Sicht „Ohne Exemplare“ zeigt den Titel %d-mal, erwartet 1", n)
	}

	ohne := abgelehnt()
	if ohne.Vorhanden.ID != angelegt.Data.ID || ohne.Vorhanden.Title != titel {
		t.Errorf("vorhanden = %+v, erwartet Kennung %s und Titel %q", *ohne.Vorhanden, angelegt.Data.ID, titel)
	}
	if !ohne.Vorhanden.OhneExemplar {
		t.Error("ohneExemplar = false, der Titel hat aber kein Exemplar")
	}
	if !strings.Contains(ohne.Error, titel) || !strings.Contains(ohne.Error, "Ohne Exemplare") {
		t.Errorf("Meldung %q nennt weder den Titel noch die Sicht, in der er steht", ohne.Error)
	}

	testdaten.LegeExemplarAn(t, pool, angelegt.Data.ID, "B-DUBLETTE-"+isbn, true, "")
	mit := abgelehnt()
	if mit.Vorhanden.OhneExemplar {
		t.Error("ohneExemplar = true, der Titel hat inzwischen ein Exemplar")
	}
	if strings.Contains(mit.Error, "Ohne Exemplare") {
		t.Errorf("Meldung %q schickt in die Sicht „Ohne Exemplare“, dort steht der Titel nicht mehr", mit.Error)
	}
}
