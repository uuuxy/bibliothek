package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Auskunft vorab (GET /api/books/vorhanden) und die Ablehnung beim Speichern (POST
// /api/books) beantworten dieselbe Frage. Der Test hält sie als Paar fest: Für jede
// Schreibweise nennt die Auskunft genau dann einen Titel, wenn das Speichern mit 409 ablehnt —
// denselben Titel mit derselben Meldung. Fragten beide nach verschiedenen Regeln, hieße es in
// der Maske erst „neu" und beim Speichern „existiert bereits".
//
// Zu den Schreibweisen gehört die Länge: Die zehnstellige ISBN vom Titelblatt und die
// dreizehnstellige vom Strichcode sind dieselbe Nummer, und die Datenbank führt sie
// dreizehnstellig (Migration 157).
func TestBuchVorhanden_SagtVorabWasDasSpeichernSagt(t *testing.T) {
	pool := pgtest.Pool(t)
	const dreizehn = "9780306406171"
	const zehn, zehnGespeichert = "080442957X", "9780804429573"
	const frei = "9780306406188"
	loesche := func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM buecher_titel WHERE isbn = ANY($1)`, []string{dreizehn, zehn, zehnGespeichert, frei}); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	durchlass := func(next http.Handler) http.Handler { return next }
	handler := NewAPIHandler(APIHandlerConfig{
		Repo:                 NewBookRepository(pool),
		Metadaten:            offlineMetadatenClient(),
		RequireViewBooks:     durchlass,
		RequireEditBooks:     durchlass,
		RequireDeleteBooks:   durchlass,
		RequireAuthenticated: durchlass,
	})

	type titel struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		OhneExemplar bool   `json:"ohneExemplar"`
	}
	// vorab fragt die Auskunft; ohne Titel ist Vorhanden nil.
	vorab := func(isbn string) (vorhanden *titel, meldung string) {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/books/vorhanden?isbn="+url.QueryEscape(isbn), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Auskunft zu %q: %d %s", isbn, rec.Code, rec.Body.String())
		}
		var antwort struct {
			Data *struct {
				Vorhanden *titel `json:"vorhanden"`
				Meldung   string `json:"meldung"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatal(err)
		}
		if antwort.Data == nil {
			t.Fatalf("Auskunft zu %q ohne data: %s", isbn, rec.Body.String())
		}
		return antwort.Data.Vorhanden, antwort.Data.Meldung
	}
	// speichere legt über die Tür der Maske an und gibt Status und Körper zurück.
	speichere := func(isbn, name string) (int, []byte) {
		t.Helper()
		daten, err := json.Marshal(map[string]any{"isbn": isbn, "title": name, "author": "Probe", "stock": 0})
		if err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(daten)))
		return rec.Code, rec.Body.Bytes()
	}

	for _, isbn := range []string{dreizehn, zehn, frei} {
		if vorhanden, _ := vorab(isbn); vorhanden != nil {
			t.Fatalf("vor dem Anlegen nennt die Auskunft zu %q schon %+v", isbn, *vorhanden)
		}
	}
	for isbn, name := range map[string]string{dreizehn: "Vorab-Probe dreizehn", zehn: "Vorab-Probe zehn"} {
		if status, koerper := speichere(isbn, name); status != http.StatusCreated {
			t.Fatalf("Anlegen von %q: %d %s", isbn, status, koerper)
		}
	}

	faelle := []struct{ schreibweise, name string }{
		{dreizehn, "Vorab-Probe dreizehn"},
		{"978-0-306-40617-1", "Vorab-Probe dreizehn"},
		{"978 0 306 40617 1", "Vorab-Probe dreizehn"},
		{"  9780306406171  ", "Vorab-Probe dreizehn"},
		// Zehnstellig angelegt: in beiden Längen dieselbe Nummer.
		{zehn, "Vorab-Probe zehn"},
		{"0-8044-2957-x", "Vorab-Probe zehn"},
		{zehnGespeichert, "Vorab-Probe zehn"},
		{"978-0-8044-2957-3", "Vorab-Probe zehn"},
		// Dreizehnstellig angelegt, zehnstellig gefragt.
		{"0306406179", "Vorab-Probe dreizehn"},
		{"0-306-40617-9", "Vorab-Probe dreizehn"},
	}
	for _, f := range faelle {
		vorhanden, meldung := vorab(f.schreibweise)
		if vorhanden == nil {
			t.Errorf("%q: die Auskunft nennt keinen Titel, erwartet %q", f.schreibweise, f.name)
			continue
		}
		if vorhanden.Title != f.name || !vorhanden.OhneExemplar {
			t.Errorf("%q: Auskunft = %+v, erwartet %q ohne Exemplar", f.schreibweise, *vorhanden, f.name)
		}

		status, koerper := speichere(f.schreibweise, "Zweiter Versuch")
		if status != http.StatusConflict {
			t.Errorf("%q: Die Auskunft nennt einen Titel, das Speichern antwortet %d %s", f.schreibweise, status, koerper)
			continue
		}
		var abgelehnt struct {
			Error     string `json:"error"`
			Vorhanden *titel `json:"vorhanden"`
		}
		if err := json.Unmarshal(koerper, &abgelehnt); err != nil {
			t.Fatal(err)
		}
		if abgelehnt.Vorhanden == nil || *abgelehnt.Vorhanden != *vorhanden || abgelehnt.Error != meldung {
			t.Errorf("%q: Auskunft (%+v, %q) und Ablehnung (%+v, %q) nennen nicht dasselbe",
				f.schreibweise, *vorhanden, meldung, abgelehnt.Vorhanden, abgelehnt.Error)
		}
	}

	// Gegenrichtung: Nennt die Auskunft keinen Titel, legt das Speichern an.
	if vorhanden, _ := vorab(frei); vorhanden != nil {
		t.Fatalf("freie ISBN: die Auskunft nennt %+v", *vorhanden)
	}
	if status, koerper := speichere(frei, "Vorab-Probe frei"); status != http.StatusCreated {
		t.Errorf("freie ISBN: Die Auskunft nennt keinen Titel, das Speichern antwortet %d %s", status, koerper)
	}

	// Mit einem Exemplar steht der Titel im Katalog: Die Meldung schickt nicht mehr in die
	// Sicht „Ohne Exemplare".
	vorher, _ := vorab(dreizehn)
	if vorher == nil {
		t.Fatal("der angelegte Titel fehlt in der Auskunft")
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`, vorher.ID, "B-VORAB-"+dreizehn); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	if mit, _ := vorab(dreizehn); mit == nil || mit.OhneExemplar {
		t.Errorf("mit Exemplar: Auskunft = %+v, erwartet ohneExemplar=false", mit)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/books/vorhanden", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ohne isbn: %d %s, erwartet 400", rec.Code, rec.Body.String())
	}
}
