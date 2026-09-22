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

// mockTransportCover is a local mock transport to avoid unexported type dependency
type mockTransportCover struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransportCover) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestHandleRefreshCover_UngueltigeKennung(t *testing.T) {
	for _, id := range []string{"", "123", "urn:uuid:0f8fad5b-d9cb-469f-a165-70867728950e"} {
		req := httptest.NewRequest(http.MethodPost, "/api/books/x/refresh-cover", nil)
		req.SetPathValue("id", id)
		w := httptest.NewRecorder()

		handler := &APIHandler{}
		handler.handleRefreshCover(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Kennung %q: expected status %d, got %d", id, http.StatusBadRequest, w.Code)
		}
	}
}

func TestHandleRefreshCover_BookNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	handler := &APIHandler{repo: NewBookRepository(mock)}

	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs("00000000-0000-0000-0000-000000000000").
		WillReturnError(pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodPost, "/api/books/00000000-0000-0000-0000-000000000000/refresh-cover", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()

	handler.handleRefreshCover(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// Gegenprobe: Ein DB-Fehler ist kein „nicht gefunden" — bis 29.08.2026 wurde jeder
// Fehler aus GetBookByID als 404 verkleidet (Schema-Drift wäre unsichtbar geblieben).
func TestHandleRefreshCover_DBFehlerIst500(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	handler := &APIHandler{repo: NewBookRepository(mock)}

	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs("00000000-0000-0000-0000-000000000000").
		WillReturnError(errTest)

	req := httptest.NewRequest(http.MethodPost, "/api/books/00000000-0000-0000-0000-000000000000/refresh-cover", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()

	handler.handleRefreshCover(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestHandleRefreshCover_MetadataSearchFailure(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	mockTr := &mockTransportCover{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("")),
				Header:     make(http.Header),
			}, nil
		},
	}
	metadaten := &MetadatenClient{httpClient: &http.Client{Transport: mockTr}}

	handler := &APIHandler{
		repo:      NewBookRepository(mock),
		metadaten: metadaten,
	}

	bookID := "11111111-1111-1111-1111-111111111111"
	lastCounted := "2023-01-01"

	// mock GetBookByID
	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs(bookID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage"}).
			AddRow(bookID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", "Track", 1, &lastCounted, 1, "Buch", 5, 10, map[string]any{}, "4. Aufl. 2023"))

	req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/refresh-cover", nil)
	req.SetPathValue("id", bookID)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	handler.handleRefreshCover(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "Kein Cover: Zu dieser ISBN kennen DNB, Google Books und OpenLibrary keinen Titel" {
		t.Errorf("unexpected error message: %v", response["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

// Metadaten gefunden, aber kein Bild: 404 — und KEIN Schreibzugriff. Bis zum 22.09.2026
// schrieb die Tür hier "New Title"/"New Author" ins Buch, obwohl der Aufrufer nur ein
// Cover wollte; der Mock hätte jedes UPDATE gemeldet, das jetzt noch käme.
func TestHandleRefreshCover_KeinCoverIst404OhneSchreiben(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	mockTr := &mockTransportCover{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "googleapis.com") {
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

	handler := &APIHandler{
		repo:      NewBookRepository(mock),
		metadaten: metadaten,
	}

	bookID := "22222222-2222-2222-2222-222222222222"
	lastCounted := "2023-01-01"

	// mock GetBookByID
	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs(bookID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage"}).
			AddRow(bookID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", "Track", 1, &lastCounted, 1, "Buch", 5, 10, map[string]any{}, "4. Aufl. 2023"))

	req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/refresh-cover", nil)
	req.SetPathValue("id", bookID)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	handler.handleRefreshCover(w, req)

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

// Erfolg: Google liefert Metadaten, DNB das Bild. Geschrieben wird NUR das Cover — Titel
// und Autor gehen mit leeren Werten an UpdateBookMetadata (COALESCE(NULLIF($1,”), titel)),
// die Antwort trägt den alten Titel und den neuen lokalen Cover-Pfad.
func TestHandleRefreshCover_Success(t *testing.T) {
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
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, bild); err != nil {
		t.Fatalf("png encode: %v", err)
	}

	mockTr := &mockTransportCover{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "portal.dnb.de/opac/mvb/cover") {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(pngBytes.Bytes())),
					Header:     make(http.Header),
				}
				resp.Header.Set("Content-Type", "image/png")
				return resp, nil
			}
			if strings.Contains(req.URL.String(), "googleapis.com") {
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

	handler := &APIHandler{
		repo:      NewBookRepository(mock),
		metadaten: metadaten,
	}

	bookID := "33333333-3333-3333-3333-333333333333"
	lastCounted := "2023-01-01"

	// mock GetBookByID
	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs(bookID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage"}).
			AddRow(bookID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", "Track", 1, &lastCounted, 1, "Buch", 5, 10, map[string]any{}, "4. Aufl. 2023"))

	// mock UpdateBookMetadata: Titel und Autor LEER, nur das Cover.
	mock.ExpectExec(`UPDATE buecher_titel`).
		WithArgs("", "", pgxmock.AnyArg(), bookID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/refresh-cover", nil)
	req.SetPathValue("id", bookID)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	handler.handleRefreshCover(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["message"] != "cover aktualisiert" {
		t.Errorf("unexpected message: %v", response["message"])
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data field in response")
	}
	coverURL, ok := data["coverUrl"].(string)
	if !ok || data["title"] != "Old Title" || data["author"] != "Old Author" || !strings.HasPrefix(coverURL, "/uploads/cover_auto_") {
		t.Fatalf("erwartet: Titel und Autor unverändert, Cover lokal — bekommen: %v", data)
	}
	if err := os.Remove(filepath.Join("uploads", filepath.Base(coverURL))); err != nil {
		t.Errorf("heruntergeladenes Testbild aufräumen: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
