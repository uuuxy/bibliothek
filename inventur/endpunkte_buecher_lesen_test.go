package inventur

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestBearbeiteBuecherListe(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	handler := &APIHandler{repo: repo}

	columns := []string{
		"id", "isbn", "title", "author", "signatur", "cover_url", "subject",
		"grade_level", "track", "verfuegbar", "gesamt", "last_counted",
		"sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
		"untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
	}

	t.Run("successful GET without parameters", func(t *testing.T) {
		mock.ExpectQuery(`(?s)SELECT.*FROM buecher_titel.*`).
			WithArgs("", (*int16)(nil), "").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("1", "123", "Buch A", "Autor A", "SigA", "url", "Math", int16(5), "", 5, 5, nil, 0, "Buch", 5, 10, "", "", 0, "", nil))

		req := httptest.NewRequest(http.MethodGet, "/buecher", nil)
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response struct {
			Data []Book `json:"data"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(response.Data) != 1 {
			t.Errorf("expected 1 book, got %d", len(response.Data))
		} else if response.Data[0].Title != "Buch A" {
			t.Errorf("expected title 'Buch A', got '%s'", response.Data[0].Title)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/buecher", nil)
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(rr, req)

		if status := rr.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
		}
	})

	t.Run("invalid grade parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/buecher?gradeLevel=invalid", nil)
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("query too long", func(t *testing.T) {
		longQuery := ""
		for i := 0; i < 201; i++ {
			longQuery += "a"
		}
		req := httptest.NewRequest(http.MethodGet, "/buecher?q="+longQuery, nil)
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("synonym matching", func(t *testing.T) {
		mock.ExpectQuery(`(?s)SELECT.*FROM buecher_titel.*`).
			WithArgs("", (*int16)(nil), "politik").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("2", "456", "Politik Buch", "Autor B", "SigB", "url", "Politik", int16(9), "", 2, 2, nil, 0, "Buch", 9, 10, "", "", 0, "", nil))

		req := httptest.NewRequest(http.MethodGet, "/buecher?q=powi", nil)
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("with grade and subject parameter", func(t *testing.T) {
		mock.ExpectQuery(`(?s)SELECT.*FROM buecher_titel.*`).
			WithArgs("Math", pgxmock.AnyArg(), "").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("1", "123", "Buch A", "Autor A", "SigA", "url", "Math", int16(5), "", 5, 5, nil, 0, "Buch", 5, 10, "", "", 0, "", nil))

		req := httptest.NewRequest(http.MethodGet, "/buecher?subject=Math&gradeLevel=5", nil)
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`(?s)SELECT.*FROM buecher_titel.*`).
			WithArgs("", (*int16)(nil), "").
			WillReturnError(func() error { return http.ErrNotSupported }())

		req := httptest.NewRequest(http.MethodGet, "/buecher", nil)
		rr := httptest.NewRecorder()

		handler.BearbeiteBuecherListe(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})
}
