package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestHandleUpdateCover(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{repo: repo}

	// Helper function for making requests
	makeReq := func(method, path string, body map[string]any) *http.Request {
		var reqBody []byte
		if body != nil {
			reqBody, _ = json.Marshal(body)
		}
		req, _ := http.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(reqBody))
		return req
	}

	var dummyLastCounted *string

	tests := []struct {
		name           string
		method         string
		path           string
		body           map[string]any
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
	}{
		{
			name:           "Invalid Route Structure",
			method:         http.MethodPut,
			path:           "/api/books/123/wrong",
			body:           nil,
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty ID",
			method:         http.MethodPut,
			path:           "/api/books//cover",
			body:           nil,
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPut,
			path:           "/api/books/123/cover",
			body:           nil, // Sending nil body will fail decoding in handleUpdateCover
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty Cover URL",
			method:         http.MethodPut,
			path:           "/api/books/123/cover",
			body:           map[string]any{"coverUrl": "   "},
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Cover URL Prefix",
			method:         http.MethodPut,
			path:           "/api/books/123/cover",
			body:           map[string]any{"coverUrl": "http://example.com/cover.jpg"},
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "UpdateBookMetadata Error",
			method:         http.MethodPut,
			path:           "/api/books/123/cover",
			body:           map[string]any{"coverUrl": "https://example.com/cover.jpg"},
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", "https://example.com/cover.jpg", "123").
					WillReturnError(ErrBookNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "GetBookByID Error",
			method:         http.MethodPut,
			path:           "/api/books/123/cover",
			body:           map[string]any{"coverUrl": "https://example.com/cover.jpg"},
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", "https://example.com/cover.jpg", "123").
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))

				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs("123").
					WillReturnError(ErrBookNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Success HTTPS",
			method:         http.MethodPut,
			path:           "/api/books/123/cover",
			body:           map[string]any{"coverUrl": "https://example.com/cover.jpg"},
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", "https://example.com/cover.jpg", "123").
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))

				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs("123").
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften",
					}).AddRow(
						"123", "9781234567890", "Test Title", "Test Author", "", "https://example.com/cover.jpg", "", int16(0), "", 1, dummyLastCounted, 1, "Buch", 5, 10, map[string]any{},
					))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Success Local Uploads",
			method:         http.MethodPut,
			path:           "/api/books/123/cover",
			body:           map[string]any{"coverUrl": "/uploads/cover.jpg"},
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", "/uploads/cover.jpg", "123").
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))

				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs("123").
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften",
					}).AddRow(
						"123", "9781234567890", "Test Title", "Test Author", "", "/uploads/cover.jpg", "", int16(0), "", 1, dummyLastCounted, 1, "Buch", 5, 10, map[string]any{},
					))
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			req := makeReq(tt.method, tt.path, tt.body)
			w := httptest.NewRecorder()

			handler.handleUpdateCover(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
