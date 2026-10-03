package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Das Speichern eines Titels fragt die Katalogdienste nicht und trägt nichts ein, was die
// Anfrage nicht nennt. Es wartete sonst auf DNB, Google Books und OpenLibrary, sobald Cover,
// Listenpreis oder Autor fehlten: Antworteten sie nicht, ließ sich nichts speichern, und
// antworteten sie, stand danach ein Autor oder ein Preis am Titel, den niemand gesehen hatte.
// Was die Dienste wissen, zeigt die Maske über die ISBN-Abfrage; die nennt dafür den Preis.
func TestSpeichern_FragtDieKatalogdiensteNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const isbn = "9783551551672"
	loesche := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	loesche()
	t.Cleanup(loesche)

	// Die Dienste kennen zur ISBN Titel, Autor und Preis: Käme beim Speichern eine Anfrage
	// an, stünde das danach am Titel.
	const satz = `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records><record><recordData>
		<record xmlns="http://www.loc.gov/MARC21/slim">
		  <datafield tag="020" ind1=" " ind2=" "><subfield code="a">9783551551672</subfield><subfield code="c">Festeinband : EUR 9.50 (DE)</subfield></datafield>
		  <datafield tag="100" ind1="1" ind2=" "><subfield code="a">Rowling, J. K.</subfield></datafield>
		  <datafield tag="245" ind1="1" ind2="0"><subfield code="a">Harry Potter und der Stein der Weisen</subfield></datafield>
		</record></recordData></record></records></searchRetrieveResponse>`
	var gefragt []string
	handler := &APIHandler{repo: &BookRepository{db: pool}, metadaten: &MetadatenClient{httpClient: &http.Client{
		Transport: &mockTransport{roundTripFunc: func(req *http.Request) (*http.Response, error) {
			gefragt = append(gefragt, req.URL.Host)
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(satz)), Header: make(http.Header)}, nil
		}},
	}}}

	sende := func(methode, id string, koerper map[string]any) *httptest.ResponseRecorder {
		t.Helper()
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
	type stand struct {
		Autor, Cover, Signatur string
		Listenpreis            *float64
	}
	lies := func() (s stand) {
		t.Helper()
		if err := pool.QueryRow(ctx,
			`SELECT COALESCE(autor, ''), COALESCE(cover_url, ''), COALESCE(signatur, ''), listenpreis::float8
			 FROM buecher_titel WHERE isbn = $1`, isbn).Scan(&s.Autor, &s.Cover, &s.Signatur, &s.Listenpreis); err != nil {
			t.Fatalf("Titel lesen: %v", err)
		}
		return s
	}

	// Anlegen: der Titel getippt, ohne Autor, Cover und Listenpreis.
	rec := sende(http.MethodPost, "", map[string]any{"isbn": isbn, "title": "Von Hand getippt", "signatur": "Spe 1"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("Anlegen: %d %s", rec.Code, rec.Body.String())
	}
	var angelegt struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &angelegt); err != nil {
		t.Fatal(err)
	}
	if s := lies(); s.Autor != "" || s.Cover != "" || s.Listenpreis != nil {
		t.Errorf("Anlegen trug ein, was die Anfrage nicht nannte: %+v", s)
	}

	// Ändern: Der Titel hat keinen Autor, die Maske schickt ihn leer zurück.
	ohneAutor := map[string]any{"isbn": isbn, "title": "Von Hand getippt", "author": "", "signatur": "Spe 2"}
	if rec := sende(http.MethodPut, angelegt.Data.ID, ohneAutor); rec.Code != http.StatusOK {
		t.Fatalf("Ändern ohne Autor: %d %s", rec.Code, rec.Body.String())
	}
	if s := lies(); s.Autor != "" || s.Cover != "" || s.Signatur != "Spe 2" {
		t.Errorf("Ändern ohne Autor: %+v — erwartet die neue Signatur, weiter kein Autor und kein Cover", s)
	}

	// Trägt der Titel einen Autor, leert ihn eine Änderung ohne Autor nicht.
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET autor = 'Von Hand, Autor' WHERE isbn = $1`, isbn); err != nil {
		t.Fatal(err)
	}
	ohneAutor["signatur"] = "Spe 3"
	if rec := sende(http.MethodPut, angelegt.Data.ID, ohneAutor); rec.Code != http.StatusBadRequest {
		t.Errorf("Ändern mit geleertem Autor: %d %s, erwartet 400", rec.Code, rec.Body.String())
	}
	if s := lies(); s.Autor != "Von Hand, Autor" || s.Signatur != "Spe 2" {
		t.Errorf("die abgelehnte Änderung hat geschrieben: %+v", s)
	}

	if len(gefragt) != 0 {
		t.Errorf("das Speichern hat die Katalogdienste gefragt: %v", gefragt)
	}

	// Die ISBN-Abfrage der Maske nennt den Preis.
	req := httptest.NewRequest(http.MethodGet, "/api/lookup/"+isbn, nil)
	req.SetPathValue("isbn", isbn)
	abfrage := httptest.NewRecorder()
	handler.handleLookup(abfrage, req)
	var antwort struct {
		Data struct {
			Preis float64 `json:"preis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(abfrage.Body.Bytes(), &antwort); err != nil || antwort.Data.Preis != 9.5 {
		t.Errorf("ISBN-Abfrage: preis %v (%v), erwartet 9.5 — %s", antwort.Data.Preis, err, abfrage.Body.String())
	}
}
