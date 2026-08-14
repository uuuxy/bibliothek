package inventur

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

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
