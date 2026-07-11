package inventur

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

var myErrTest = errors.New("test error")

func TestBearbeiteBuecherListe(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{
		repo: repo,
	}

	t.Run("invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/books", nil)
		w := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("query too long", func(t *testing.T) {
		longQuery := ""
		for i := 0; i < 201; i++ {
			longQuery += "a"
		}

		req := httptest.NewRequest(http.MethodGet, "/api/books?q=" + longQuery, nil)
		w := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("invalid gradeLevel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/books?gradeLevel=abc", nil)
		w := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/books?q=mathe&gradeLevel=5&subject=math", nil)
		w := httptest.NewRecorder()

		gradeLevel := int16(5)
		mock.ExpectQuery("(?s)SELECT .* FROM buecher_titel.*").
			WithArgs("math", &gradeLevel, "mathematik").
			WillReturnError(myErrTest)

		handler.BearbeiteBuecherListe(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("success natural sort", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/books?q=bio", nil)
		w := httptest.NewRecorder()

		rows := pgxmock.NewRows([]string{
			"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
		}).AddRow(
			"uuid2", "123", "Bio 10", "Author", "Sig", "", "", int16(0), "", 1, 1, nil, 0, "Buch", 5, 6, "", "", 0, "", nil,
		).AddRow(
			"uuid1", "123", "Bio 2", "Author", "Sig", "", "", int16(0), "", 1, 1, nil, 0, "Buch", 5, 6, "", "", 0, "", nil,
		)

		mock.ExpectQuery("(?s)SELECT .* FROM buecher_titel.*").
			WithArgs("", (*int16)(nil), "biologie").
			WillReturnRows(rows)

		handler.BearbeiteBuecherListe(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var result map[string][]Book
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		books := result["data"]
		if len(books) != 2 {
			t.Errorf("expected 2 books, got %d", len(books))
		}

		// The sorting should put "Bio 2" before "Bio 10" because of natural sorting
		if books[0].Title != "Bio 2" {
			t.Errorf("expected 'Bio 2', got '%s'", books[0].Title)
		}
		if books[1].Title != "Bio 10" {
			t.Errorf("expected 'Bio 10', got '%s'", books[1].Title)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
