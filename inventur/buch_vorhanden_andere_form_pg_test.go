package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Dieselbe ISBN gibt es in zwei Längen, und die Normalform trennt sie: Ein Titel aus Littera
// trägt die zehnstellige vom Titelblatt, der Scanner liest vom Buch die dreizehnstellige. Die
// Auskunft der Maske (GET /api/books/vorhanden) nennt den Titel unter der anderen Länge als
// Vorschlag — nicht unter „vorhanden", denn das Speichern lehnt ihn nicht ab: Unter der
// anderen Form kann ein anderes Buch stehen.
func TestBuchVorhanden_NenntDenTitelUnterDerAnderenLaenge(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	// Zwei Bücher: eines mit zehnstelliger ISBN im Katalog, eines mit dreizehnstelliger.
	const zehn, zehnAlsDreizehn = "3551551677", "9783551551672"
	const dreizehn, dreizehnAlsZehn = "9783791504650", "3791504657"
	const ohneZehnerForm = "9791036700316"
	loesche := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = ANY($1)`,
			[]string{zehn, zehnAlsDreizehn, dreizehn, dreizehnAlsZehn, ohneZehnerForm}); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)
	for isbn, titel := range map[string]string{zehn: "Aus Littera, zehnstellig", dreizehn: "Dreizehnstellig"} {
		if _, err := pool.Exec(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2)`, titel, isbn); err != nil {
			t.Fatal(err)
		}
	}

	durchlass := func(next http.Handler) http.Handler { return next }
	handler := NewAPIHandler(APIHandlerConfig{
		Repo: NewBookRepository(pool), Metadaten: offlineMetadatenClient(),
		RequireViewBooks: durchlass, RequireEditBooks: durchlass,
		RequireDeleteBooks: durchlass, RequireAuthenticated: durchlass,
	})
	type vorschlag struct {
		Title string `json:"title"`
		ISBN  string `json:"isbn"`
	}
	type auskunft struct {
		Vorhanden  *vorschlag `json:"vorhanden"`
		AndereForm *vorschlag `json:"andereForm"`
		Meldung    string     `json:"meldung"`
	}
	vorab := func(isbn string) auskunft {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/books/vorhanden?isbn="+url.QueryEscape(isbn), nil))
		var antwort struct {
			Data auskunft `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("Auskunft zu %q: %d %v %s", isbn, rec.Code, err, rec.Body.String())
		}
		return antwort.Data
	}

	for _, f := range []struct{ gefragt, traegt, titel, laenge string }{
		{zehnAlsDreizehn, zehn, "Aus Littera, zehnstellig", "zehnstelliger"},
		{"978-3-551-55167-2", zehn, "Aus Littera, zehnstellig", "zehnstelliger"},
		{dreizehnAlsZehn, dreizehn, "Dreizehnstellig", "dreizehnstelliger"},
	} {
		a := vorab(f.gefragt)
		if a.Vorhanden != nil {
			t.Errorf("%q: die Auskunft nennt den Titel unter „vorhanden\" (%+v) — das Speichern lehnt ihn nicht ab", f.gefragt, *a.Vorhanden)
		}
		if a.AndereForm == nil || a.AndereForm.Title != f.titel || a.AndereForm.ISBN != f.traegt {
			t.Errorf("%q: andereForm = %+v, erwartet %q unter %s", f.gefragt, a.AndereForm, f.titel, f.traegt)
			continue
		}
		if !strings.Contains(a.Meldung, f.laenge+" Form ("+f.traegt+")") || !strings.Contains(a.Meldung, f.titel) {
			t.Errorf("%q: Meldung %q nennt weder Länge noch Titel", f.gefragt, a.Meldung)
		}
	}

	// Die eigene Form bleibt die Ablehnung, und ohne zehnstellige Form (979) gibt es keinen Vorschlag.
	if a := vorab(zehn); a.Vorhanden == nil || a.AndereForm != nil {
		t.Errorf("die ISBN des Titels selbst: %+v, erwartet vorhanden und keinen Vorschlag", a)
	}
	if a := vorab(ohneZehnerForm); a.Vorhanden != nil || a.AndereForm != nil {
		t.Errorf("979er ISBN: %+v, erwartet nichts", a)
	}

	// Der Vorschlag hält das Speichern nicht auf: Wer „ein anderes Buch" sagt, legt es an.
	daten, err := json.Marshal(map[string]any{"isbn": zehnAlsDreizehn, "title": "Ein anderes Buch", "author": "Probe"})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(daten)))
	if rec.Code != http.StatusCreated {
		t.Errorf("Anlegen unter der anderen Länge: %d %s, erwartet 201", rec.Code, rec.Body.String())
	}
}
