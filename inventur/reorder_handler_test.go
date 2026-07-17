package inventur

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestHandleReorderBooks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{
		repo: repo,
	}

	tests := []struct {
		name           string
		payload        any
		setupMock      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Invalid JSON payload",
			payload:        "invalid json",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ungültiges json",
		},
		{
			name: "Empty BookIDs",
			payload: ReorderRequest{
				BookIDs: []string{},
			},
			setupMock:      func() {},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"nichts zu speichern"}`,
		},
		{
			name: "Transaction Start Error",
			payload: ReorderRequest{
				BookIDs: []string{"id1", "id2"},
			},
			setupMock: func() {
				mock.ExpectBegin().WillReturnError(fmt.Errorf("db connection error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Ein interner Datenbankfehler ist aufgetreten.",
		},
		{
			name: "Database Exec Error",
			payload: ReorderRequest{
				BookIDs: []string{"id1", "id2"},
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE buecher_titel SET sort_order = daten.neue_reihenfolge").
					WithArgs([]string{"id1", "id2"}, []int{1, 2}).
					WillReturnError(fmt.Errorf("exec error"))
				mock.ExpectRollback()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Ein interner Datenbankfehler ist aufgetreten.",
		},
		{
			name: "Database Commit Error",
			payload: ReorderRequest{
				BookIDs: []string{"id1", "id2"},
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE buecher_titel SET sort_order = daten.neue_reihenfolge").
					WithArgs([]string{"id1", "id2"}, []int{1, 2}).
					WillReturnResult(pgxmock.NewResult("UPDATE", 2))
				mock.ExpectCommit().WillReturnError(fmt.Errorf("commit error"))
				// ExpectRollback is NOT needed here because pgx marks the tx as closed on a Commit failure,
				// so tx.Rollback() will return ErrTxClosed and won't hit the DB.
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Ein interner Datenbankfehler ist aufgetreten.",
		},
		{
			name: "Successful Reorder",
			payload: ReorderRequest{
				BookIDs: []string{"id1", "id2", "id3"},
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE buecher_titel SET sort_order = daten.neue_reihenfolge").
					WithArgs([]string{"id1", "id2", "id3"}, []int{1, 2, 3}).
					WillReturnResult(pgxmock.NewResult("UPDATE", 3))
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"erfolgreich 3 bücher sortiert"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var body []byte
			var err error
			if str, ok := tt.payload.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPut, "/api/admin/books/reorder", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.handleReorderBooks(w, req)

			res := w.Result()
			defer res.Body.Close() //nolint:errcheck

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			var respBody map[string]any
			if res.StatusCode == http.StatusOK {
				if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				expectedBodyMap := map[string]any{}
				if err := json.Unmarshal([]byte(tt.expectedBody), &expectedBodyMap); err != nil {
					t.Fatalf("failed to unmarshal expected body: %v", err)
				}

				if respBody["message"] != expectedBodyMap["message"] {
					t.Errorf("expected body %s, got %v", tt.expectedBody, respBody)
				}
			} else {
				var errorBody map[string]string
				if err := json.NewDecoder(res.Body).Decode(&errorBody); err != nil {
					t.Fatalf("failed to decode error body: %v", err)
				}
				if errorBody["error"] != tt.expectedBody {
					t.Errorf("expected error message %s, got %s", tt.expectedBody, errorBody["error"])
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}
