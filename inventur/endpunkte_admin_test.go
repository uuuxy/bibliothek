package inventur

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
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
		RequireDeleteBooks:   func(h http.Handler) http.Handler { return h },
		RequireAuthenticated: func(h http.Handler) http.Handler { return h },
	})

	// Keiner der Fälle erreicht die Datenbank: Das Mock hat keine Erwartung.
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
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
			// Das Umsortieren von Hand gibt es nicht mehr: Die Titelliste steht nach dem Titel.
			name:           "PUT /api/admin/books/reorder - zurückgebaut, 404",
			method:         http.MethodPut,
			path:           "/api/admin/books/reorder",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
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
