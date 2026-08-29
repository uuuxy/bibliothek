package inventur

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtrahiereZahlenUndBasis(t *testing.T) {
	tests := []struct {
		name         string
		titel        string
		expectedZahl int
		expectedBase string
	}{
		{
			name:         "Normal string with number at the end",
			titel:        "Band 1",
			expectedZahl: 1,
			expectedBase: "Band",
		},
		{
			name:         "String without numbers",
			titel:        "Buch",
			expectedZahl: 0,
			expectedBase: "Buch",
		},
		{
			name:         "String with numbers at the start",
			titel:        "123 Test",
			expectedZahl: 123,
			expectedBase: "Test",
		},
		{
			name:         "String with multiple numbers",
			titel:        "Buch 42 mit 99",
			expectedZahl: 42,
			expectedBase: "Buch  mit",
		},
		{
			name:         "Empty string",
			titel:        "",
			expectedZahl: 0,
			expectedBase: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zahl, basis := extrahiereZahlenUndBasis(tt.titel)
			if zahl != tt.expectedZahl {
				t.Errorf("extrahiereZahlenUndBasis(%q) got zahl = %d, want %d", tt.titel, zahl, tt.expectedZahl)
			}
			if basis != tt.expectedBase {
				t.Errorf("extrahiereZahlenUndBasis(%q) got basis = %q, want %q", tt.titel, basis, tt.expectedBase)
			}
		})
	}
}

func TestBearbeiteBuecherListe(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - empty params",
			url:  "/api/books",
			setupMock: func(m pgxmock.PgxPoolIface) {
				dateStr := "2023-01-01"
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", pgxmock.AnyArg(), "", 50000).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
					}).AddRow(
						"b1", "123", "Book 1", "Author 1", "Sig 1", "url", "Math", int16(5), "G", int64(5), int64(10), &dateStr, 0, "Buch", 5, 10, "Sub", "Ver", 2020, "", map[string]any{},
					))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"id":"b1","isbn":"123","title":"Book 1","author":"Author 1","signatur":"Sig 1","coverUrl":"url","subject":"Math","gradeLevel":5,"track":"G","stock":10,"verfuegbar":5,"gesamt":10,"lastCounted":"2023-01-01","sortOrder":0,"medientyp":"Buch","jahrgangVon":5,"jahrgangBis":10,"untertitel":"Sub","verlag":"Ver","erscheinungsjahr":2020,"beschreibung":"","erweiterteEigenschaften":{}}]}`,
		},
		{
			name: "Success - synonym translation",
			url:  "/api/books?q=powi",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", pgxmock.AnyArg(), "politik", 50000).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[]}`,
		},
		{
			name: "Error - query string too long",
			url:  "/api/books?q=" + strings.Repeat("a", 201),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"suchbegriff zu lang (max. 200 zeichen)"}`,
		},
		{
			name: "Error - invalid gradeLevel",
			url:  "/api/books?gradeLevel=abc",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"ungültiger query-parameter gradeLevel"}`,
		},
		{
			name: "Error - invalid grade",
			url:  "/api/books?grade=xyz",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"ungültiger query-parameter gradeLevel"}`,
		},
		{
			name: "Success - grade param fallback",
			url:  "/api/books?grade=7",
			setupMock: func(m pgxmock.PgxPoolIface) {
				grade := int16(7)
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", &grade, "", 50000).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track", "verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis", "untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[]}`,
		},
		{
			name: "Error - DB fails",
			url:  "/api/books",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", pgxmock.AnyArg(), "", 50000).
					WillReturnError(errors.New("db fail"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Ein interner Datenbankfehler ist aufgetreten."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			repo := NewBookRepository(mock)
			handler := &APIHandler{repo: repo}

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.BearbeiteBuecherListe(w, req)

			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.expectedStatus, res.StatusCode)

			if tt.expectedBody != "" {
				body, _ := io.ReadAll(res.Body)
				assert.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}
