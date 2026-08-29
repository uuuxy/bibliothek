package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// mockTransportCover is a local mock transport to avoid unexported type dependency
type mockTransportCover struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransportCover) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestFallbackString(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{
			name:     "non-empty value, empty fallback",
			value:    "primary",
			fallback: "",
			want:     "primary",
		},
		{
			name:     "empty value, non-empty fallback",
			value:    "",
			fallback: "secondary",
			want:     "secondary",
		},
		{
			name:     "both non-empty",
			value:    "primary",
			fallback: "secondary",
			want:     "primary",
		},
		{
			name:     "both empty",
			value:    "",
			fallback: "",
			want:     "",
		},
		{
			name:     "whitespace value is considered non-empty",
			value:    "   ",
			fallback: "secondary",
			want:     "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fallbackString(tt.value, tt.fallback); got != tt.want {
				t.Errorf("fallbackString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleRefreshCover_InvalidRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/books/123/wrong", nil)
	w := httptest.NewRecorder()

	handler := &APIHandler{}
	handler.handleRefreshCover(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleRefreshCover_EmptyID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/books//refresh-cover", nil)
	w := httptest.NewRecorder()

	handler := &APIHandler{}
	handler.handleRefreshCover(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
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
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften"}).
			AddRow(bookID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", int16(1), "Track", 1, &lastCounted, 1, "Buch", 5, 10, map[string]any{}))

	req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/refresh-cover", nil)
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

	if response["error"] != "Keine neuen Metadaten gefunden" {
		t.Errorf("unexpected error message: %v", response["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestHandleRefreshCover_UpdateFailure(t *testing.T) {
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
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften"}).
			AddRow(bookID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", int16(1), "Track", 1, &lastCounted, 1, "Buch", 5, 10, map[string]any{}))

	// mock UpdateBookMetadata returning error
	mock.ExpectExec(`UPDATE buecher_titel`).
		WithArgs("New Title", "New Author", "", bookID).
		WillReturnError(errTest)

	req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/refresh-cover", nil)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	handler.handleRefreshCover(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestHandleRefreshCover_Success(t *testing.T) {
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

	bookID := "33333333-3333-3333-3333-333333333333"
	lastCounted := "2023-01-01"

	// mock GetBookByID
	mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\)`).
		WithArgs(bookID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften"}).
			AddRow(bookID, "9783161484100", "Old Title", "Old Author", "Sig", "", "Subject", int16(1), "Track", 1, &lastCounted, 1, "Buch", 5, 10, map[string]any{}))

	// mock UpdateBookMetadata success
	mock.ExpectExec(`UPDATE buecher_titel`).
		WithArgs("New Title", "New Author", "", bookID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/refresh-cover", nil)
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
	if data["title"] != "New Title" || data["author"] != "New Author" || data["coverUrl"] != "" {
		t.Errorf("unexpected updated data: %v", data)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
