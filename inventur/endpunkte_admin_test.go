package inventur

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestHandleAdminBooks_Routing(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockPool.Close()

	repo := NewBookRepository(mockPool)
	metadatenClient := &MetadatenClient{httpClient: &http.Client{}}

	handler := NewAPIHandler(APIHandlerConfig{
		Repo:                 repo,
		Metadaten:            metadatenClient,
		RequireViewBooks:     func(h http.Handler) http.Handler { return h },
		RequireEditBooks:     func(h http.Handler) http.Handler { return h },
		RequireAuthenticated: func(h http.Handler) http.Handler { return h },
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		setupMock      func()
	}{
		{
			// Der 501-Zweig zu diesem Pfad ist am 01.09.2026 zurückgebaut (C-Posten,
			// keine Aufrufer in Frontend oder Go): Der funktionierende Import ist
			// /api/books/import; dieser Pfad fällt jetzt ehrlich in den 404-Default.
			name:           "POST /api/admin/books/import - zurückgebaut, 404",
			method:         http.MethodPost,
			path:           "/api/admin/books/import",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "GET invalid path - Not Found",
			method:         http.MethodGet,
			path:           "/api/admin/books/invalid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "POST invalid path - Not Found",
			method:         http.MethodPost,
			path:           "/api/admin/books/invalid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "PUT invalid path - Not Found",
			method:         http.MethodPut,
			path:           "/api/admin/books/invalid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "DELETE invalid path - Not Found",
			method:         http.MethodDelete,
			path:           "/api/admin/books/invalid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "PATCH unsupported method - Not Found",
			method:         http.MethodPatch,
			path:           "/api/admin/books",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "PUT /api/admin/books/reorder - Success triggers backup",
			method:         http.MethodPut,
			path:           "/api/admin/books/reorder",
			expectedStatus: http.StatusOK,
			setupMock: func() {
				mockPool.ExpectBegin()
				mockPool.ExpectExec(`UPDATE buecher_titel SET sort_order = daten.neue_reihenfolge`).
					WithArgs(
						pgxmock.AnyArg(), // input.BookIDs []string
						pgxmock.AnyArg(), // sortOrders []int
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mockPool.ExpectCommit()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupMock != nil {
				tc.setupMock()
			}

			var req *http.Request
			if tc.method == http.MethodPut && tc.path == "/api/admin/books/reorder" {
				payload := `{"bookIds": ["b95d0df8-2b87-4d69-a1d2-069bc1399f57"]}`
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(payload))
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			rec := httptest.NewRecorder()

			handler.handleAdminBooks(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}

			if err := mockPool.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
