package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestHandleClassBooks(t *testing.T) {
	tests := []struct {
		name           string
		branch         string
		sort           string
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success - Default (No query params)",
			branch: "",
			sort:   "",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", KlassensatzMindestLeser).
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "quelle", "leser",
					}).AddRow(
						"5A", "b1", "Book 1", "Math", "G", "url", "123", 5, 10, "hand", 0,
					))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"className":"5A","books":[{"id":"b1","title":"Book 1","subject":"Math","track":"G","coverUrl":"url","isbn":"123","verfuegbar":5,"gesamt":10,"quelle":"hand","leser":0}]}]}`,
		},
		{
			name:   "Success - Empty results fallback",
			branch: "",
			sort:   "",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", KlassensatzMindestLeser).
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "quelle", "leser",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[]}`,
		},
		{
			name:   "Success - With Query Params",
			branch: "F",
			sort:   "desc",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("F", KlassensatzMindestLeser).
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "quelle", "leser",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[]}`,
		},
		{
			name:   "Database Error",
			branch: "",
			sort:   "",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", KlassensatzMindestLeser).
					WillReturnError(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Ein interner Datenbankfehler ist aufgetreten."}`,
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

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/class-books", nil) //nolint:errcheck
			if tt.branch != "" || tt.sort != "" {
				q := req.URL.Query()
				if tt.branch != "" {
					q.Add("branch", tt.branch)
				}
				if tt.sort != "" {
					q.Add("sort", tt.sort)
				}
				req.URL.RawQuery = q.Encode()
			}

			w := httptest.NewRecorder()
			handler.handleClassBooks(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestFormatClassName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Single digit",
			input:    "1",
			expected: "01",
		},
		{
			name:     "Single digit 9",
			input:    "9",
			expected: "09",
		},
		{
			name:     "Two digits",
			input:    "10",
			expected: "10",
		},
		{
			name:     "Two digits 11",
			input:    "11",
			expected: "11",
		},
		{
			name:     "Starting with 0",
			input:    "05",
			expected: "05",
		},
		{
			name:     "Starting with 0 single digit",
			input:    "0",
			expected: "0",
		},
		{
			name:     "Digit and letter",
			input:    "5A",
			expected: "05A",
		},
		{
			name:     "Two digits and letter",
			input:    "10B",
			expected: "10B",
		},
		{
			name:     "With spaces single digit",
			input:    " 5 C ",
			expected: "05C",
		},
		{
			name:     "With spaces two digits",
			input:    " 10 B ",
			expected: "10B",
		},
		{
			name:     "Just letters",
			input:    "A",
			expected: "A",
		},
		{
			name:     "Letters and spaces",
			input:    " A B ",
			expected: "AB",
		},
		{
			name:     "Lowercase letters with digit",
			input:    "5 b",
			expected: "05b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatClassName(tt.input)
			if result != tt.expected {
				t.Errorf("formatClassName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHandleDeleteClassGroup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{repo: repo}

	t.Run("No class name provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/class-books", nil)
		rec := httptest.NewRecorder()
		handler.handleDeleteClassGroup(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("Database Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM class_books").
			WithArgs(pgxmock.AnyArg()).
			WillReturnError(fmt.Errorf("db error"))

		req := httptest.NewRequest(http.MethodDelete, "/api/class-books?className=5A", nil)
		rec := httptest.NewRecorder()
		handler.handleDeleteClassGroup(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM class_books").
			WithArgs(pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		req := httptest.NewRequest(http.MethodDelete, "/api/class-books?className=5A", nil)
		rec := httptest.NewRecorder()
		handler.handleDeleteClassGroup(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestHandleAddClassBooks(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockPool.Close()

		handler := &APIHandler{repo: NewBookRepository(mockPool)}

		req := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()

		handler.handleAddClassBooks(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("No class names", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockPool.Close()

		handler := &APIHandler{repo: NewBookRepository(mockPool)}

		payload := map[string]interface{}{
			"classNames": []string{"   ", ""},
			"bookIds":    []string{"123"},
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.handleAddClassBooks(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Class name too long", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockPool.Close()

		handler := &APIHandler{repo: NewBookRepository(mockPool)}

		payload := map[string]interface{}{
			"classNames": []string{"ThisClassNameIsWayTooLongToBeValidAndShouldBeRejected"},
			"bookIds":    []string{"123"},
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.handleAddClassBooks(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Repository error", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockPool.Close()

		handler := &APIHandler{repo: NewBookRepository(mockPool)}
		mockPool.ExpectExec("INSERT INTO class_books").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("db error"))

		payload := map[string]interface{}{
			"classNames": []string{"10A"},
			"bookIds":    []string{"123"},
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.handleAddClassBooks(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("Success", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mockPool.Close()

		mockPool.ExpectExec("INSERT INTO class_books").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 2))

		handler := &APIHandler{repo: NewBookRepository(mockPool)}

		payload := map[string]interface{}{
			"classNames": []string{"10A", "10B"},
			"bookIds":    []string{"123", "456"},
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.handleAddClassBooks(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})
}

func TestHandleUpdateClassBooks(t *testing.T) {
	errTest := errors.New("test error")

	tests := []struct {
		name           string
		method         string
		body           string
		mockSetup      func(mock pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Invalid JSON",
			method:         "POST",
			body:           `{invalid}`,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ungültiges JSON",
		},
		{
			name:           "Missing Class Name",
			method:         "POST",
			body:           `{"bookIds": ["123"]}`,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "es muss mindestens ein klassenname angegeben werden",
		},
		{
			name:           "Class Name Too Long",
			method:         "POST",
			body:           `{"className": "ThisClassNameIsWayTooLong", "bookIds": ["123"]}`,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "klassenname darf maximal 20 zeichen lang sein",
		},
		{
			name:   "Database Error",
			method: "POST",
			body:   `{"className": "5A", "bookIds": ["123"]}`,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin().WillReturnError(errTest)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Ein interner Datenbankfehler ist aufgetreten.",
		},
		{
			name:   "Success single className",
			method: "POST",
			body:   `{"oldClassName": "5A", "className": "5B", "bookIds": ["123"]}`,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("^DELETE FROM class_books WHERE class_name = \\$1$").
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
				mock.ExpectExec("^DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)$").
					WithArgs([]string{"05B"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
				mock.ExpectExec("^INSERT INTO class_books \\(class_name, book_id\\).*").
					WithArgs([]string{"05B"}, []string{"123"}).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "erfolgreich gespeichert",
		},
		{
			name:   "Success multiple classNames",
			method: "POST",
			body:   `{"classNames": ["5C", "5D"], "bookIds": ["123", "456"]}`,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("^DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)$").
					WithArgs([]string{"05C", "05D"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
				mock.ExpectExec("^INSERT INTO class_books \\(class_name, book_id\\).*").
					WithArgs([]string{"05C", "05C", "05D", "05D"}, []string{"123", "456", "123", "456"}).
					WillReturnResult(pgxmock.NewResult("INSERT", 4))
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "erfolgreich gespeichert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("failed to create pgxmock: %v", err)
			}
			defer mock.Close()

			repo := NewBookRepository(mock)
			handler := &APIHandler{repo: repo}

			tt.mockSetup(mock)

			req := httptest.NewRequest(tt.method, "/api/admin/class-books", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.handleUpdateClassBooks(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, w.Code)
			}

			bodyStr := strings.TrimSpace(w.Body.String())
			if !strings.Contains(bodyStr, tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, bodyStr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
