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

func TestHandleUpdateCover(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{repo: repo}

	validBookID := "56fb9216-16e7-4bd9-9da8-111111111111"
	validURL := "https://example.com/cover.jpg"
	validUpload := "/uploads/cover.jpg"

	tests := []struct {
		name           string
		path           string
		method         string
		body           interface{}
		mockSetup      func()
		expectedStatus int
	}{

		{
			name:           "Invalid Method",
			path:           "/api/books/" + validBookID + "/cover",
			method:         http.MethodGet,
			body:           nil,
			mockSetup:      func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid route structure",
			path:           "/api/wrong/123/cover",
			body:           nil,
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty ID in route",
			path:           "/api/books//cover",
			body:           nil,
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			path:           "/api/books/" + validBookID + "/cover",
			body:           nil, // Custom handling below for raw string
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty coverUrl",
			path:           "/api/books/" + validBookID + "/cover",
			body:           map[string]string{"coverUrl": "   "},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid coverUrl prefix",
			path:           "/api/books/" + validBookID + "/cover",
			body:           map[string]string{"coverUrl": "http://example.com/image.jpg"},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid coverUrl but UpdateBookMetadata fails",
			path:           "/api/books/" + validBookID + "/cover",
			body:           map[string]string{"coverUrl": validURL},
			mockSetup: func() {
				mock.ExpectExec("(?s)UPDATE .*").
					WithArgs("", "", validURL, validBookID).
					WillReturnError(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "UpdateBookMetadata succeeds but GetBookByID fails",
			path:           "/api/books/" + validBookID + "/cover",
			body:           map[string]string{"coverUrl": validUpload},
			mockSetup: func() {
				mock.ExpectExec("(?s)UPDATE .*").
					WithArgs("", "", validUpload, validBookID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))

				mock.ExpectQuery("(?s)SELECT .*").
					WithArgs(validBookID).
					WillReturnError(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Success",
			path:           "/api/books/" + validBookID + "/cover",
			body:           map[string]string{"coverUrl": validURL},
			mockSetup: func() {
				mock.ExpectExec("(?s)UPDATE .*").
					WithArgs("", "", validURL, validBookID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))

				rows := pgxmock.NewRows([]string{"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "stock", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "erweiterte_eigenschaften"}).
					AddRow(validBookID, "123", "Title", "Author", "Sig", validURL, "Math", int16(5), "A", 10, nil, 1, "Buch", 5, 10, map[string]any{})

				mock.ExpectQuery("(?s)SELECT .*").
					WithArgs(validBookID).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var bodyBytes []byte
			if tt.name == "Invalid JSON" {
				bodyBytes = []byte("{invalid-json")
			} else if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			reqMethod := http.MethodPost
			if tt.method != "" {
				reqMethod = tt.method
			}
			req := httptest.NewRequest(reqMethod, tt.path, bytes.NewReader(bodyBytes))
			rr := httptest.NewRecorder()

			handler.handleUpdateCover(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
