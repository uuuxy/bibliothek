package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestBearbeiteBuchErstellen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	// We'll mock metadaten using a dummy client with a small timeout, but we don't expect it to be called if title/author/cover are provided
	metadaten := NeuerMetadatenClient()

	handler := &APIHandler{
		repo:      repo,
		metadaten: metadaten,
		mux:       http.NewServeMux(),
	}

	tests := []struct {
		name           string
		method         string
		requestBody    any
		mockDB         func()
		expectedStatus int
		expectedBody   string // substring match
	}{
		{
			name:           "Invalid HTTP method",
			method:         http.MethodGet,
			requestBody:    nil,
			mockDB:         func() {},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "nur post-anfragen erlaubt",
		},
		{
			name:           "Invalid JSON body",
			method:         http.MethodPost,
			requestBody:    "{invalid json",
			mockDB:         func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ungültiges JSON",
		},
		{
			name:           "Missing ISBN",
			method:         http.MethodPost,
			requestBody:    map[string]any{"title": "Test Book", "author": "Test Author", "coverUrl": "test.jpg"},
			mockDB:         func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "isbn ist erforderlich",
		},
		{
			name:           "Invalid ISBN format",
			method:         http.MethodPost,
			requestBody:    map[string]any{"isbn": "123", "title": "Test Book", "author": "Test Author", "coverUrl": "test.jpg"},
			mockDB:         func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ungültiges ISBN-Format",
		},
		{
			name:           "Invalid GradeLevel",
			method:         http.MethodPost,
			requestBody:    map[string]any{"isbn": "978-3-16-148410-0", "gradeLevel": 15, "title": "Test Book", "author": "Test Author", "coverUrl": "test.jpg"},
			mockDB:         func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "gradeLevel muss zwischen 0 und 13 sein",
		},
		{
			name:   "Successful creation",
			method: http.MethodPost,
			requestBody: map[string]any{
				"isbn":       "978-3-16-148410-0",
				"title":      "Test Book",
				"author":     "Test Author",
				"coverUrl":   "http://example.com/cover.jpg",
				"subject":    "Math",
				"gradeLevel": 5,
				"track":      "Gymnasium",
				"stock":      0, // avoid syncBookStock
				"medientyp":  "Buch",
				"signatur":   "MAT-5",
			},
			mockDB: func() {
				mock.ExpectQuery(`(?s)INSERT INTO buecher_titel.*`).
					WithArgs(
						"978-3-16-148410-0", // isbn
						"Test Book",         // title
						"Test Author",       // author
						"http://example.com/cover.jpg", // coverUrl
						"Math",              // subject
						int16(5),            // gradeLevel
						"Gymnasium",         // track
						0,                   // stock
						(*string)(nil),      // lastCounted
						"Buch",              // medientyp
						pgxmock.AnyArg(),    // erweiterteEigenschaften
						0,                   // jahrgangVon
						0,                   // jahrgangBis
						"",                  // untertitel
						"",                  // verlag
						0,                   // erscheinungsjahr
						"",                  // beschreibung
						"MAT-5",             // signatur
					).
					WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("uuid-1234"))
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "buch erstellt",
		},
		{
			name:   "Duplicate ISBN",
			method: http.MethodPost,
			requestBody: map[string]any{
				"isbn":     "978-3-16-148410-0",
				"title":    "Duplicate Book",
				"author":   "Duplicate Author",
				"coverUrl": "http://example.com/dup.jpg",
				"stock":    0,
			},
			mockDB: func() {
				mock.ExpectQuery(`(?s)INSERT INTO buecher_titel.*`).
					WithArgs(
						"978-3-16-148410-0", // isbn
						"Duplicate Book",         // title
						"Duplicate Author",       // author
						"http://example.com/dup.jpg", // coverUrl
						"",              // subject
						int16(0),            // gradeLevel
						"",         // track
						0,                   // stock
						(*string)(nil),      // lastCounted
						"Buch",              // medientyp
						pgxmock.AnyArg(),    // erweiterteEigenschaften
						0,                   // jahrgangVon
						0,                   // jahrgangBis
						"",                  // untertitel
						"",                  // verlag
						0,                   // erscheinungsjahr
						"",                  // beschreibung
						"",             // signatur
					).
					WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "books_isbn_key"})
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "Ein Buch mit dieser ISBN existiert bereits in der Datenbank",
		},
		{
			name:   "Generic DB Error",
			method: http.MethodPost,
			requestBody: map[string]any{
				"isbn":     "978-3-16-148410-0",
				"title":    "Error Book",
				"author":   "Error Author",
				"coverUrl": "http://example.com/err.jpg",
				"stock":    0,
			},
			mockDB: func() {
				mock.ExpectQuery(`(?s)INSERT INTO buecher_titel.*`).
					WithArgs(
						"978-3-16-148410-0", // isbn
						"Error Book",         // title
						"Error Author",       // author
						"http://example.com/err.jpg", // coverUrl
						"",              // subject
						int16(0),            // gradeLevel
						"",         // track
						0,                   // stock
						(*string)(nil),      // lastCounted
						"Buch",              // medientyp
						pgxmock.AnyArg(),    // erweiterteEigenschaften
						0,                   // jahrgangVon
						0,                   // jahrgangBis
						"",                  // untertitel
						"",                  // verlag
						0,                   // erscheinungsjahr
						"",                  // beschreibung
						"",             // signatur
					).
					WillReturnError(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "buch konnte nicht erstellt werden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDB()

			var reqBodyBytes []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				reqBodyBytes = []byte(str)
			} else if tt.requestBody != nil {
				reqBodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/books", bytes.NewReader(reqBodyBytes))
			rr := httptest.NewRecorder()

			handler.BearbeiteBuchErstellen(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedBody != "" && !bytes.Contains(rr.Body.Bytes(), []byte(tt.expectedBody)) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, rr.Body.String())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
