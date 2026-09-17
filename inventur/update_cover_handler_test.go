package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

func TestHandleUpdateCover(t *testing.T) {
	const buchID = "0f8fad5b-d9cb-469f-a165-70867728950e"

	// makeReq setzt den Platzhalter {id} selbst — im Betrieb füllt ihn der Mux (api_routen.go).
	makeReq := func(method, id string, body map[string]any) *http.Request {
		var reqBody []byte
		if body != nil {
			reqBody, _ = json.Marshal(body) //nolint:errcheck
		}
		req, _ := http.NewRequestWithContext(context.Background(), method, "/api/books/"+id+"/cover", bytes.NewReader(reqBody)) //nolint:errcheck
		req.SetPathValue("id", id)
		return req
	}

	tests := []struct {
		name           string
		method         string
		id             string
		body           map[string]any
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
	}{
		{
			name:           "Kennung ist keine UUID",
			method:         http.MethodPut,
			id:             "123",
			body:           map[string]any{"coverUrl": "/uploads/cover.jpg"},
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty ID",
			method:         http.MethodPut,
			id:             "",
			body:           nil,
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPut,
			id:             buchID,
			body:           nil, // Sending nil body will fail decoding in handleUpdateCover
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty Cover URL",
			method:         http.MethodPut,
			id:             buchID,
			body:           map[string]any{"coverUrl": "   "},
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Cover URL Prefix",
			method:         http.MethodPut,
			id:             buchID,
			body:           map[string]any{"coverUrl": "http://example.com/cover.jpg"},
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			// Der Befund: Vorher genügte das https://-Präfix, jede beliebige Fremd-URL
			// ließ sich dauerhaft in cover_url schreiben. Geladen wird sie anschließend
			// von jedem Browser, der Katalog oder Monitor öffnet.
			name:           "Fremder HTTPS-Host wird abgelehnt",
			method:         http.MethodPut,
			id:             buchID,
			body:           map[string]any{"coverUrl": "https://angreifer.example/zaehler.jpg"},
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			// Allowlist-Name im String, fremder Host in Wahrheit — der Vergleich muss
			// über url.Hostname() laufen, nicht über strings.Contains.
			name:           "Allowlist-Host als Subdomain eines Angreifers",
			method:         http.MethodPut,
			id:             buchID,
			body:           map[string]any{"coverUrl": "https://covers.openlibrary.org.angreifer.example/x.jpg"},
			setupMock:      func(m pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "UpdateBookMetadata Error",
			method: http.MethodPut,
			id:     buchID,
			body:   map[string]any{"coverUrl": "https://covers.openlibrary.org/b/isbn/9781234567890-L.jpg"},
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", "https://covers.openlibrary.org/b/isbn/9781234567890-L.jpg", buchID).
					WillReturnError(ErrBookNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "GetBookByID Error",
			method: http.MethodPut,
			id:     buchID,
			body:   map[string]any{"coverUrl": "https://covers.openlibrary.org/b/isbn/9781234567890-L.jpg"},
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", "https://covers.openlibrary.org/b/isbn/9781234567890-L.jpg", buchID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))

				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs(buchID).
					WillReturnError(ErrBookNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "Success HTTPS",
			method: http.MethodPut,
			id:     buchID,
			body:   map[string]any{"coverUrl": "https://covers.openlibrary.org/b/isbn/9781234567890-L.jpg"},
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", "https://covers.openlibrary.org/b/isbn/9781234567890-L.jpg", buchID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))

				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs(buchID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage",
					}).AddRow(
						buchID, "9781234567890", "Test Title", "Test Author", "", "https://covers.openlibrary.org/b/isbn/9781234567890-L.jpg", "", int16(0), "", 1, nil, 1, "Buch", 5, 10, nil, "",
					))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Success Local Uploads",
			method: http.MethodPut,
			id:     buchID,
			body:   map[string]any{"coverUrl": "/uploads/cover.jpg"},
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec("(?s)UPDATE buecher_titel.*").
					WithArgs("", "", "/uploads/cover.jpg", buchID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))

				m.ExpectQuery("(?s)SELECT id, COALESCE.*").
					WithArgs(buchID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften", "auflage",
					}).AddRow(
						buchID, "9781234567890", "Test Title", "Test Author", "", "/uploads/cover.jpg", "", int16(0), "", 1, nil, 1, "Buch", 5, 10, nil, "",
					))
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer mock.Close()

			repo := NewBookRepository(mock)
			handler := &APIHandler{repo: repo}

			tt.setupMock(mock)

			req := makeReq(tt.method, tt.id, tt.body)
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
