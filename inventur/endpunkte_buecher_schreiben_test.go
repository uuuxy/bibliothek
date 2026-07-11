package inventur

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestBearbeiteBuecherLoeschen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{repo: repo}

	t.Run("InvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/buecher", nil)
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherLoeschen(rr, req)

		// The test expects writeError to be called. writeError uses apierrors.SendHTTPError
		// which writes JSON with {"error": "..."}.
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %v, got %v", http.StatusMethodNotAllowed, rr.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/buecher", strings.NewReader("invalid json"))
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherLoeschen(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %v, got %v", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("EmptyIDs", func(t *testing.T) {
		body := `{"ids": []}`
		req := httptest.NewRequest(http.MethodDelete, "/buecher", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherLoeschen(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %v, got %v", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("Success", func(t *testing.T) {
		body := `{"ids": ["123e4567-e89b-12d3-a456-426614174000"]}`
		req := httptest.NewRequest(http.MethodDelete, "/buecher", strings.NewReader(body))
		rr := httptest.NewRecorder()

		// pgxmock requires the precise regex for the ExpectQuery to match.
		// "SELECT COUNT(*) FROM ausleihen a JOIN buecher_exemplare e ON a.exemplar_id = e.id WHERE e.titel_id = ANY($1::uuid[]) AND a.rueckgabe_am IS NULL"
		// (?s) regex flag can help for multiline matching.
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM ausleihen a.*WHERE e\.titel_id = ANY\(\$1::uuid\[\]\)`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnRows(pgxmock.NewRows([]string{"cover_url"}))

		mock.ExpectExec(`DELETE FROM schadensfaelle`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		mock.ExpectExec(`DELETE FROM ausleihen`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		mock.ExpectExec(`DELETE FROM buecher_titel`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		handler.BearbeiteBuecherLoeschen(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %v, got %v", http.StatusOK, rr.Code)
		}

		var res map[string]string
		json.Unmarshal(rr.Body.Bytes(), &res)
		if res["message"] != "bücher gelöscht" {
			t.Errorf("expected message 'bücher gelöscht', got '%v'", res["message"])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		body := `{"ids": ["123e4567-e89b-12d3-a456-426614174000"]}`
		req := httptest.NewRequest(http.MethodDelete, "/buecher", strings.NewReader(body))
		rr := httptest.NewRecorder()

		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM ausleihen a.*WHERE e\.titel_id = ANY\(\$1::uuid\[\]\)`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnRows(pgxmock.NewRows([]string{"cover_url"}))

		mock.ExpectExec(`DELETE FROM schadensfaelle`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		mock.ExpectExec(`DELETE FROM ausleihen`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		mock.ExpectExec(`DELETE FROM buecher_titel`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnResult(pgxmock.NewResult("DELETE", 0)) // 0 rows affected indicates not found in db_books_delete.go

		handler.BearbeiteBuecherLoeschen(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %v, got %v", http.StatusNotFound, rr.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("ActiveLoans", func(t *testing.T) {
		body := `{"ids": ["123e4567-e89b-12d3-a456-426614174000"]}`
		req := httptest.NewRequest(http.MethodDelete, "/buecher", strings.NewReader(body))
		rr := httptest.NewRecorder()

		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM ausleihen a.*WHERE e\.titel_id = ANY\(\$1::uuid\[\]\)`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1)) // Active loan

		handler.BearbeiteBuecherLoeschen(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %v, got %v", http.StatusBadRequest, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "löschen abgebrochen") { // The db package returns lowercase 'löschen abgebrochen'
			t.Errorf("expected error containing 'löschen abgebrochen', got %v", rr.Body.String())
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("DBError", func(t *testing.T) {
		body := `{"ids": ["123e4567-e89b-12d3-a456-426614174000"]}`
		req := httptest.NewRequest(http.MethodDelete, "/buecher", strings.NewReader(body))
		rr := httptest.NewRecorder()

		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM ausleihen a.*WHERE e\.titel_id = ANY\(\$1::uuid\[\]\)`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel`).
			WithArgs([]string{"123e4567-e89b-12d3-a456-426614174000"}).
			WillReturnError(context.DeadlineExceeded)

		handler.BearbeiteBuecherLoeschen(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %v, got %v", http.StatusInternalServerError, rr.Code)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
