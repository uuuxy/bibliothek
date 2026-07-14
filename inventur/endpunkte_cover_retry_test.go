package inventur

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// coverQueryRegex matcht die SELECT-Query von handleListExternalCovers (LIMIT $1).
const coverQueryRegex = `(?s)SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url.*FROM buecher_titel.*WHERE cover_url LIKE 'http%'.*ORDER BY id ASC.*LIMIT \$1`

func TestHandleListExternalCovers(t *testing.T) {
	// Subtests in eigene Top-Level-Funktionen ausgelagert, damit die Assertions
	// nicht in der t.Run-Closure verschachtelt liegen (S3776). Namen unverändert.
	t.Run("Success", testListExternalCoversSuccess)
	t.Run("DatabaseError", testListExternalCoversDatabaseError)
}

func testListExternalCoversSuccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	handler := &APIHandler{repo: NewBookRepository(mock)}

	mock.ExpectQuery(coverQueryRegex).
		WithArgs(300).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
			AddRow("1", "1234567890", "Test Book 1", "http://example.com/cover1.jpg").
			AddRow("2", "0987654321", "Test Book 2", "https://example.com/cover2.jpg"))

	req := httptest.NewRequest(http.MethodGet, "/api/covers/external", nil)
	w := httptest.NewRecorder()

	handler.handleListExternalCovers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string][]Book
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	books, ok := response["data"]
	if !ok {
		t.Fatalf("Expected 'data' key in response")
	}

	if len(books) != 2 {
		t.Errorf("Expected 2 books, got %d", len(books))
	}
	if books[0].ID != "1" || books[1].ID != "2" {
		t.Errorf("Unexpected book IDs returned")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func testListExternalCoversDatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	handler := &APIHandler{repo: NewBookRepository(mock)}

	mock.ExpectQuery(coverQueryRegex).
		WithArgs(300).
		WillReturnError(errTest)

	req := httptest.NewRequest(http.MethodGet, "/api/covers/external", nil)
	w := httptest.NewRecorder()

	handler.handleListExternalCovers(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if response["error"] != "Ein interner Datenbankfehler ist aufgetreten." {
		t.Errorf("Unexpected error message: %s", response["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
