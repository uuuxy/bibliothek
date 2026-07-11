package inventur

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestExtrahiereZahlenUndBasis(t *testing.T) {
	tests := []struct {
		name       string
		titel      string
		wantZahl   int
		wantBasis  string
	}{
		{"Einfacher Titel", "Mathematik", 0, "Mathematik"},
		{"Titel mit Zahl am Ende", "Mathematik 5", 5, "Mathematik"},
		{"Titel mit Zahl mittendrin", "Natur und 3 Technik", 3, "Natur und  Technik"},
		{"Titel mit zweistelliger Zahl", "Teil 10", 10, "Teil"},
		{"Titel mit mehreren Zahlen", "Teil 2 und 3", 2, "Teil  und"},
		{"Titel nur mit Zahl", "42", 42, ""},
		{"Leer", "", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotZahl, gotBasis := extrahiereZahlenUndBasis(tt.titel)
			if gotZahl != tt.wantZahl || gotBasis != tt.wantBasis {
				t.Errorf("extrahiereZahlenUndBasis(%q) = %v, %q, want %v, %q", tt.titel, gotZahl, gotBasis, tt.wantZahl, tt.wantBasis)
			}
		})
	}
}

func TestSortiereBuecherNatuerlich(t *testing.T) {
	buecher := []Book{
		{Title: "Mathematik 10", SortOrder: 0},
		{Title: "Mathematik 2", SortOrder: 0},
		{Title: "Deutsch", SortOrder: 0},
		{Title: "Wichtiges Buch", SortOrder: -1}, // Admin order
		{Title: "Mathematik 1", SortOrder: 0},
	}

	sortiereBuecherNatuerlich(buecher)

	expectedTitles := []string{
		"Wichtiges Buch", // sort order -1
		"Deutsch",        // basis "deutsch", zahl 0
		"Mathematik 1",   // basis "mathematik", zahl 1
		"Mathematik 2",   // basis "mathematik", zahl 2
		"Mathematik 10",  // basis "mathematik", zahl 10
	}

	for i, expected := range expectedTitles {
		if buecher[i].Title != expected {
			t.Errorf("An Index %d erwartete Titel %q, bekam %q", i, expected, buecher[i].Title)
		}
	}
}

func TestBearbeiteBuecherListe(t *testing.T) {
	tests := []struct {
		name           string
		urlQuery       string
		method         string
		mockSetup      func(pgxmock.PgxPoolIface)
		expectedStatus int
		expectedJSON   string // optional substring match
	}{
		{
			name:           "Method Not Allowed",
			urlQuery:       "",
			method:         http.MethodPost,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Search query too long",
			urlQuery:       "?q=" + strings.Repeat("a", 201),
			method:         http.MethodGet,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid grade level",
			urlQuery:       "?gradeLevel=invalid",
			method:         http.MethodGet,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Database error",
			urlQuery:       "?q=test",
			method:         http.MethodGet,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("(?s)SELECT .*").
					WithArgs("", pgxmock.AnyArg(), "test").
					WillReturnError(fmt.Errorf("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Success with synonym",
			urlQuery:       "?q=powi",
			method:         http.MethodGet,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				columns := []string{
					"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track",
					"verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
					"untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
				}
				rows := pgxmock.NewRows(columns).
					AddRow("1", "123", "Politik 1", "Autor", "Sig", "", "Politik", int16(5), "", 1, 1, nil, 0, "Buch", 5, 6, "", "", 2020, "", nil)

				// "powi" is translated to "politik"
				mock.ExpectQuery("(?s)SELECT .*").
					WithArgs("", pgxmock.AnyArg(), "politik").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			expectedJSON:   "Politik 1",
		},
		{
			name:           "Success with grade level",
			urlQuery:       "?gradeLevel=7&subject=Mathe",
			method:         http.MethodGet,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				columns := []string{
					"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track",
					"verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
					"untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
				}
				rows := pgxmock.NewRows(columns).
					AddRow("2", "456", "Mathe 7", "Autor", "Sig", "", "Mathe", int16(7), "", 1, 1, nil, 0, "Buch", 7, 8, "", "", 2020, "", nil)

				mock.ExpectQuery("(?s)SELECT .*").
					WithArgs("Mathe", pgxmock.AnyArg(), "").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			expectedJSON:   "Mathe 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("failed to create pgxmock: %v", err)
			}
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewBookRepository(mock)
			handler := &APIHandler{repo: repo}

			req := httptest.NewRequest(tt.method, "/api/books"+tt.urlQuery, nil)
			rr := httptest.NewRecorder()

			handler.BearbeiteBuecherListe(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedJSON != "" && !strings.Contains(rr.Body.String(), tt.expectedJSON) {
				t.Errorf("expected response to contain %q, got %q", tt.expectedJSON, rr.Body.String())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}
