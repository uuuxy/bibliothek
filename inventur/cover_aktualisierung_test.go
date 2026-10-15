package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/pashagolub/pgxmock/v5"
)

// mockTransportCover ist eine eigene Attrappe des Transports dieses Tests.
type mockTransportCover struct {
	roundTripFunc func(anfrage *http.Request) (*http.Response, error)
}

func (m *mockTransportCover) RoundTrip(anfrage *http.Request) (*http.Response, error) {
	return m.roundTripFunc(anfrage)
}

func TestCoverAktualisieren_UngueltigeKennung(t *testing.T) {
	for _, id := range []string{"", "123", "urn:uuid:0f8fad5b-d9cb-469f-a165-70867728950e"} {
		anfrage := httptest.NewRequest(http.MethodPost, "/api/books/x/refresh-cover", nil)
		anfrage.SetPathValue("id", id)
		w := httptest.NewRecorder()

		tueren := &Tueren{}
		tueren.CoverAktualisieren(w, anfrage)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Kennung %q: expected status %d, got %d", id, http.StatusBadRequest, w.Code)
		}
	}
}

func TestCoverAktualisieren_TitelFehlt(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	tueren := &Tueren{abfragen: NewTitelAbfragen(mock)}

	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs("00000000-0000-0000-0000-000000000000").
		WillReturnError(pgx.ErrNoRows)

	anfrage := httptest.NewRequest(http.MethodPost, "/api/books/00000000-0000-0000-0000-000000000000/refresh-cover", nil)
	anfrage.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()

	tueren.CoverAktualisieren(w, anfrage)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// Gegenprobe: Ein Fehler der Datenbank ist kein „nicht gefunden"; als 404 verkleidet bliebe ein
// Auseinanderlaufen des Schemas unsichtbar.
func TestCoverAktualisieren_DatenbankfehlerIst500(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	tueren := &Tueren{abfragen: NewTitelAbfragen(mock)}

	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs("00000000-0000-0000-0000-000000000000").
		WillReturnError(errTest)

	anfrage := httptest.NewRequest(http.MethodPost, "/api/books/00000000-0000-0000-0000-000000000000/refresh-cover", nil)
	anfrage.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()

	tueren.CoverAktualisieren(w, anfrage)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestCoverAktualisieren_SucheScheitert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	mockTr := &mockTransportCover{
		roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("")),
				Header:     make(http.Header),
			}, nil
		},
	}
	metadaten := &MetadatenClient{httpClient: &http.Client{Transport: mockTr}}

	tueren := &Tueren{
		abfragen:  NewTitelAbfragen(mock),
		metadaten: metadaten,
	}

	titelID := "11111111-1111-1111-1111-111111111111"
	zuletztGezaehlt := "2023-01-01"

	// LadeTitel nachspielen.
	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage"}).
			AddRow(titelID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", "Track", 1, &zuletztGezaehlt, 1, "Buch", 5, 10, map[string]any{}, "4. Aufl. 2023"))

	anfrage := httptest.NewRequest(http.MethodPost, "/api/books/"+titelID+"/refresh-cover", nil)
	anfrage.SetPathValue("id", titelID)
	anfrage = anfrage.WithContext(context.Background())
	w := httptest.NewRecorder()

	tueren.CoverAktualisieren(w, anfrage)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var antwort map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if antwort["error"] != "Kein Cover: Zu dieser ISBN kennen DNB, Google Books und OpenLibrary keinen Titel." {
		t.Errorf("unexpected error message: %v", antwort["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// Angaben gefunden, aber kein Bild: 404, und kein Schreibzugriff. Die Tür darf weder Titel noch
// Autor ins Buch schreiben, der Aufrufer wollte nur ein Cover; die Attrappe meldete jedes
// UPDATE, das käme.
func TestCoverAktualisieren_KeinCoverIst404OhneSchreiben(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	mockTr := &mockTransportCover{
		roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			if strings.Contains(anfrage.URL.String(), "googleapis.com") {
				googleJSON := `{
					"items": [{
						"volumeInfo": {
							"title": "New Title",
							"authors": ["New Author"],
							"imageLinks": {"thumbnail": ""}
						}
					}]
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(googleJSON)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("")),
				Header:     make(http.Header),
			}, nil
		},
	}
	metadaten := &MetadatenClient{httpClient: &http.Client{Transport: mockTr}}

	tueren := &Tueren{
		abfragen:  NewTitelAbfragen(mock),
		metadaten: metadaten,
	}

	titelID := "22222222-2222-2222-2222-222222222222"
	zuletztGezaehlt := "2023-01-01"

	// LadeTitel nachspielen.
	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage"}).
			AddRow(titelID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", "Track", 1, &zuletztGezaehlt, 1, "Buch", 5, 10, map[string]any{}, "4. Aufl. 2023"))

	anfrage := httptest.NewRequest(http.MethodPost, "/api/books/"+titelID+"/refresh-cover", nil)
	anfrage.SetPathValue("id", titelID)
	anfrage = anfrage.WithContext(context.Background())
	w := httptest.NewRecorder()

	tueren.CoverAktualisieren(w, anfrage)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "kein Cover") {
		t.Errorf("die Antwort sagt nicht, dass es KEIN Cover gibt: %s", w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// Erfolg: Google liefert Metadaten, DNB das Bild. Geschrieben wird nur das Cover — Titel
// und Autor gehen mit leeren Werten an AendereTitelMetadaten, das einen leeren Wert übergeht
// (NULLIF auf den leeren Text, dann COALESCE mit dem alten Wert); die Antwort trägt den alten
// Titel und den neuen lokalen Cover-Pfad.
func TestCoverAktualisieren_Erfolg(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	bild := image.NewRGBA(image.Rect(0, 0, 15, 15))
	for x := 0; x < 15; x++ {
		for y := 0; y < 15; y++ {
			bild.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var pngDaten bytes.Buffer
	if err := png.Encode(&pngDaten, bild); err != nil {
		t.Fatalf("png encode: %v", err)
	}

	mockTr := &mockTransportCover{
		roundTripFunc: func(anfrage *http.Request) (*http.Response, error) {
			if strings.Contains(anfrage.URL.String(), "portal.dnb.de/opac/mvb/cover") {
				antwort := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(pngDaten.Bytes())),
					Header:     make(http.Header),
				}
				antwort.Header.Set("Content-Type", "image/png")
				return antwort, nil
			}
			if strings.Contains(anfrage.URL.String(), "googleapis.com") {
				googleJSON := `{
					"items": [{
						"volumeInfo": {
							"title": "New Title",
							"authors": ["New Author"],
							"imageLinks": {"thumbnail": ""}
						}
					}]
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(googleJSON)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("")),
				Header:     make(http.Header),
			}, nil
		},
	}
	metadaten := &MetadatenClient{httpClient: &http.Client{Transport: mockTr}}

	tueren := &Tueren{
		abfragen:  NewTitelAbfragen(mock),
		metadaten: metadaten,
	}

	titelID := "33333333-3333-3333-3333-333333333333"
	zuletztGezaehlt := "2023-01-01"

	// LadeTitel nachspielen.
	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage"}).
			AddRow(titelID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", "Track", 1, &zuletztGezaehlt, 1, "Buch", 5, 10, map[string]any{}, "4. Aufl. 2023"))

	// mock AendereTitelMetadaten: Titel und Autor leer, nur das Cover.
	mock.ExpectExec(`UPDATE buecher_titel`).
		WithArgs("", "", pgxmock.AnyArg(), titelID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	anfrage := httptest.NewRequest(http.MethodPost, "/api/books/"+titelID+"/refresh-cover", nil)
	anfrage.SetPathValue("id", titelID)
	anfrage = anfrage.WithContext(context.Background())
	w := httptest.NewRecorder()

	tueren.CoverAktualisieren(w, anfrage)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var tuerAntwort map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &tuerAntwort); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if tuerAntwort["message"] != "cover aktualisiert" {
		t.Errorf("unexpected message: %v", tuerAntwort["message"])
	}

	daten, ok := tuerAntwort["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data field in response")
	}
	coverURL, ok := daten["coverUrl"].(string)
	if !ok || daten["title"] != "Old Title" || daten["author"] != "Old Author" || !strings.HasPrefix(coverURL, "/uploads/cover_auto_") {
		t.Fatalf("erwartet: Titel und Autor unverändert, Cover lokal — bekommen: %v", daten)
	}
	if err := os.Remove(filepath.Join("uploads", filepath.Base(coverURL))); err != nil {
		t.Errorf("heruntergeladenes Testbild aufräumen: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
