package inventur

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestHandleGetSubjects_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{repo: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/subjects", nil)
	rr := httptest.NewRecorder()

	rows := pgxmock.NewRows([]string{"id", "name", "is_active"}).
		AddRow(1, "Deutsch", true).
		AddRow(2, "Mathematik", true)

	mock.ExpectQuery("^SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC$").
		WillReturnRows(rows)

	handler.handleGetSubjects(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var response map[string][]Subject
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	subjects := response["data"]
	if len(subjects) != 2 {
		t.Fatalf("expected 2 subjects, got %d", len(subjects))
	}
	if subjects[0].Name != "Deutsch" {
		t.Errorf("expected first subject to be Deutsch, got %s", subjects[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestHandleGetSubjects_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{repo: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/subjects", nil)
	rr := httptest.NewRecorder()

	mock.ExpectQuery("^SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC$").
		WillReturnError(errors.New("db error"))

	handler.handleGetSubjects(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
