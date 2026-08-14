package inventur

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
					WithArgs("").
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt",
					}).AddRow(
						"5A", "b1", "Book 1", "Math", "G", "url", "123", 5, 10,
					))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"className":"5A","books":[{"id":"b1","title":"Book 1","subject":"Math","track":"G","coverUrl":"url","isbn":"123","stock":10,"verfuegbar":5,"gesamt":10}]}]}`,
		},
		{
			name:   "Success - Empty results fallback",
			branch: "",
			sort:   "",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("").
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt",
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
					WithArgs("F").
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt",
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
					WithArgs("").
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
