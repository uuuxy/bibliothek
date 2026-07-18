package inventur

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// ensure mockTransport is either used or we redefine it if it's not exported.
// Since mockTransport is defined in metadaten_client_test.go without export,
// we can use it here if they are in the same package (inventur).

func TestValidiereBuchErstellenEingabe(t *testing.T) {
	tests := []struct {
		name         string
		isbn         string
		klassenStufe int16
		wantResult   bool
		wantStatus   int
	}{
		{
			name:         "Valid Input",
			isbn:         "978-3-16-148410-0",
			klassenStufe: 5,
			wantResult:   true,
			wantStatus:   http.StatusOK, // default status for httptest.ResponseRecorder if no error
		},
		{
			name:         "Empty ISBN",
			isbn:         "",
			klassenStufe: 5,
			wantResult:   false,
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "Invalid ISBN Format",
			isbn:         "123",
			klassenStufe: 5,
			wantResult:   false,
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "Invalid Grade Level Negative",
			isbn:         "978-3-16-148410-0",
			klassenStufe: -1,
			wantResult:   false,
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "Invalid Grade Level Too High",
			isbn:         "978-3-16-148410-0",
			klassenStufe: 14,
			wantResult:   false,
			wantStatus:   http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			result := validiereBuchErstellenEingabe(recorder, tt.isbn, tt.klassenStufe)
			if result != tt.wantResult {
				t.Errorf("got result %v, want %v", result, tt.wantResult)
			}
			if result == false {
				if recorder.Code != tt.wantStatus {
					t.Errorf("got status %d, want %d", recorder.Code, tt.wantStatus)
				}
			}
		})
	}
}

func TestBearbeiteBuecherLoeschen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	backup := NewBackupManager("dummy-url")
	defer backup.Stop()

	handler := &APIHandler{
		repo:   repo,
		backup: backup,
	}

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/books", strings.NewReader("invalid json"))
		rec := httptest.NewRecorder()
		handler.BearbeiteBuecherLoeschen(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("Empty IDs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/books", strings.NewReader(`{"ids": []}`))
		rec := httptest.NewRecorder()
		handler.BearbeiteBuecherLoeschen(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("Success", func(t *testing.T) {

		// Expected database operations for DeleteBooks
		mock.ExpectQuery(`SELECT COUNT\(\*\)`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery("SELECT cover_url").
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"cover_url"}).AddRow("/uploads/cover.jpg"))

		mock.ExpectExec("DELETE FROM schadensfaelle").
			WithArgs(pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		mock.ExpectExec("DELETE FROM ausleihen").
			WithArgs(pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		mock.ExpectExec("DELETE FROM buecher_titel").
			WithArgs(pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		body := `{"ids": ["11111111-1111-1111-1111-111111111111"]}`
		req := httptest.NewRequest(http.MethodDelete, "/api/books", strings.NewReader(body))
		rec := httptest.NewRecorder()
		handler.BearbeiteBuecherLoeschen(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestBearbeiteBuchErstellen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	backup := NewBackupManager("dummy-url")
	defer backup.Stop()

	// Mock HTTP Client for MetadatenClient
	mockHTTP := &http.Client{
		Transport: &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       http.NoBody,
				}, nil
			},
		},
	}
	metaClient := &MetadatenClient{httpClient: mockHTTP}

	handler := &APIHandler{
		repo:      repo,
		metadaten: metaClient,
		backup:    backup,
	}

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/books", strings.NewReader("invalid json"))
		rec := httptest.NewRecorder()
		handler.BearbeiteBuchErstellen(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("Invalid Input Validation", func(t *testing.T) {
		body := `{"isbn": "", "subject": "Math", "gradeLevel": 5}`
		req := httptest.NewRequest(http.MethodPost, "/api/books", strings.NewReader(body))
		rec := httptest.NewRecorder()
		handler.BearbeiteBuchErstellen(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("Success", func(t *testing.T) {
		body := `{
			"isbn": "978-3-16-148410-0",
			"subject": "Math",
			"gradeLevel": 5,
			"title": "Test Title",
			"author": "Test Author",
			"coverUrl": "test.jpg"
		}`

		mock.ExpectQuery(`INSERT INTO buecher_titel`).
			WithArgs(
				"978-3-16-148410-0", // isbn
				"Test Title",        // title
				"Test Author",       // author
				"test.jpg",          // cover_url
				"Math",              // subject
				int16(5),            // grade_level
				"",                  // track
				0,                   // stock
				pgxmock.AnyArg(),    // last_counted
				"Buch",              // medientyp
				pgxmock.AnyArg(),    // erweiterte_eigenschaften
				0,                   // jahrgang_von
				0,                   // jahrgang_bis
				"",                  // untertitel
				"",                  // verlag
				0,                   // erscheinungsjahr
				"",                  // beschreibung
				"",                  // signatur
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("11111111-1111-1111-1111-111111111111"))

		req := httptest.NewRequest(http.MethodPost, "/api/books", strings.NewReader(body))
		// Add context value if necessary (like auth), but it doesn't seem strictly required for these endpoints based on their code.
		rec := httptest.NewRecorder()
		handler.BearbeiteBuchErstellen(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusCreated)
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response body: %v", err)
		}
		if resp["message"] != "buch erstellt" {
			t.Errorf("got message %v, want 'buch erstellt'", resp["message"])
		}
	})
}
