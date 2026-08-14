package inventur

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
		body, _ := json.Marshal(payload)
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
		body, _ := json.Marshal(payload)
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
		body, _ := json.Marshal(payload)
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
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.handleAddClassBooks(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})
}
