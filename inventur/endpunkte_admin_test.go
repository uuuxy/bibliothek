package inventur

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

func TestTitelVerwaltung_Routen(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() = %v", err)
	}
	defer mockPool.Close()

	abfragen := NewTitelAbfragen(mockPool)
	metadatenClient := &MetadatenClient{httpClient: &http.Client{}}

	tueren := NewTueren(TuerenKonfig{
		Abfragen:          abfragen,
		Metadaten:         metadatenClient,
		VerlangeLesen:     func(h http.Handler) http.Handler { return h },
		VerlangeAendern:   func(h http.Handler) http.Handler { return h },
		VerlangeLoeschen:  func(h http.Handler) http.Handler { return h },
		VerlangeAnmeldung: func(h http.Handler) http.Handler { return h },
	})

	// Keiner der Fälle erreicht die Datenbank: Das Mock hat keine Erwartung.
	faelle := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			// Der Pfad hat keine Tür (keine Aufrufer in Oberfläche oder Go): Der Import ist
			// /api/books/import; dieser Pfad fällt in die Vorgabe 404.
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

	for _, tc := range faelle {
		t.Run(tc.name, func(t *testing.T) {
			anfrage := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			tueren.TitelVerwaltung(rec, anfrage)

			if rec.Code != tc.expectedStatus {
				t.Errorf("rec.Code = %d, erwartet %d", rec.Code, tc.expectedStatus)
			}

			if err := mockPool.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
