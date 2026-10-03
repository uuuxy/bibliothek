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

// Eine zehnstellige Nummer mit falschem Prüfzeichen ist keine andere Schreibweise der
// dreizehnstelligen (Migration 157): Am Testserver steht unter 3499500252 ein anderes
// Buch als unter 9783499500251, und die Rechnung führt von der einen auf die andere. Die
// Auskunft der Maske (GET /api/books/vorhanden) nennt deshalb zu der einen Nummer nicht den
// Titel der anderen, auch nicht als Vorschlag, und das Speichern legt an.
func TestBuchVorhanden_FalschesPruefzeichenWirdNichtGepaart(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const falsch, dreizehn = "3499500252", "9783499500251"
	loesche := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = ANY($1)`, []string{falsch, dreizehn}); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ('Ein Roman', $1)`, dreizehn); err != nil {
		t.Fatal(err)
	}

	durchlass := func(next http.Handler) http.Handler { return next }
	handler := NewAPIHandler(APIHandlerConfig{
		Repo: NewBookRepository(pool), Metadaten: offlineMetadatenClient(),
		RequireViewBooks: durchlass, RequireEditBooks: durchlass,
		RequireDeleteBooks: durchlass, RequireAuthenticated: durchlass,
	})
	// vorab liefert die Felder der Auskunft, wie sie in der Antwort stehen.
	vorab := func(isbn string) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/books/vorhanden?isbn="+url.QueryEscape(isbn), nil))
		var antwort struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("Auskunft zu %q: %d %v %s", isbn, rec.Code, err, rec.Body.String())
		}
		return antwort.Data
	}

	for _, isbn := range []string{falsch, "3-499-50025-2"} {
		a := vorab(isbn)
		if len(a) != 1 || a["vorhanden"] != nil {
			t.Errorf("%q: Auskunft %v — erwartet nur „vorhanden“ ohne Titel", isbn, a)
		}
	}

	daten, err := json.Marshal(map[string]any{"isbn": falsch, "title": "Ein anderes Buch", "author": "Probe"})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(daten)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("Anlegen unter der Nummer mit falschem Prüfzeichen: %d %s, erwartet 201", rec.Code, rec.Body.String())
	}

	// Danach trägt jede Nummer ihren eigenen Titel.
	for isbn, titel := range map[string]string{falsch: "Ein anderes Buch", dreizehn: "Ein Roman"} {
		vorhanden, ok := vorab(isbn)["vorhanden"].(map[string]any)
		if !ok || vorhanden["title"] != titel {
			t.Errorf("%q: Auskunft nennt %v, erwartet den Titel %q", isbn, vorhanden, titel)
		}
	}
}
