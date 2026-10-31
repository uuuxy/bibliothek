package katalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/repository"
)

func TestKlassensaetze(t *testing.T) {
	faelle := []struct {
		name           string
		branch         string
		sort           string
		setupMock      func(pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success - Default (No query params)",
			branch: "",
			sort:   "",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", repository.KlassensatzMindestLeser).
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
					}).AddRow(
						"5A", "b1", "Book 1", "Math", "G", "url", "123", 5, 10, 0, "hand", 0, nil,
					))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"className":"5A","books":[{"id":"b1","title":"Book 1","subject":"Math","track":"G","coverUrl":"url","isbn":"123","verfuegbar":5,"gesamt":10,"imZulauf":0,"quelle":"hand","leser":0}]}]`,
		},
		{
			// Gemischte Auflagen: Die Schlüssel der Aufschlüsselung
			// liest die Kachel (KlassenBuchKachel) — id, auflage, erscheinungsjahr, kinder.
			name:   "Success - Auflagen der Klasse",
			branch: "",
			sort:   "",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", repository.KlassensatzMindestLeser).
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
					}).AddRow(
						"07B", "b1", "Mathe 7", "Math", "G", "", "", 0, 4, 0, "ausleihe", 6,
						[]byte(`[{"id":"b1","auflage":"4. Aufl.","erscheinungsjahr":2023,"kinder":4},{"id":"b0","auflage":"3. Aufl.","erscheinungsjahr":2019,"kinder":2}]`),
					))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"className":"07B","books":[{"id":"b1","title":"Mathe 7","subject":"Math","track":"G","coverUrl":"","isbn":"","verfuegbar":0,"gesamt":4,"imZulauf":0,"quelle":"ausleihe","leser":6,"auflagen":[{"id":"b1","auflage":"4. Aufl.","erscheinungsjahr":2023,"kinder":4},{"id":"b0","auflage":"3. Aufl.","erscheinungsjahr":2019,"kinder":2}]}]}]`,
		},
		{
			name:   "Success - Empty results fallback",
			branch: "",
			sort:   "",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", repository.KlassensatzMindestLeser).
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		{
			name:   "Success - With Query Params",
			branch: "F",
			sort:   "desc",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("F", repository.KlassensatzMindestLeser).
					WillReturnRows(pgxmock.NewRows([]string{
						"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt", "im_zulauf", "quelle", "leser", "auflagen",
					}))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		{
			name:   "Database Error",
			branch: "",
			sort:   "",
			setupMock: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery("(?s)SELECT.*").
					WithArgs("", repository.KlassensatzMindestLeser).
					WillReturnError(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Ein interner Serverfehler ist aufgetreten"}`,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock.NewPool() = %v", err)
			}
			defer mock.Close()

			tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

			f.setupMock(mock)

			anfrage, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/class-books", nil) //nolint:errcheck
			if f.branch != "" || f.sort != "" {
				q := anfrage.URL.Query()
				if f.branch != "" {
					q.Add("branch", f.branch)
				}
				if f.sort != "" {
					q.Add("sort", f.sort)
				}
				anfrage.URL.RawQuery = q.Encode()
			}

			w := httptest.NewRecorder()
			tueren.Klassensaetze()(w, anfrage)

			if w.Code != f.expectedStatus {
				t.Errorf("Status = %v, erwartet %v", w.Code, f.expectedStatus)
			}
			if diff := cmp.Diff(jsonWert(t, f.expectedBody), jsonWert(t, w.Body.String())); diff != "" {
				t.Errorf("Antwort weicht ab (-erwartet +ist):\n%s", diff)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestNormalisiereKlassenname(t *testing.T) {
	faelle := []struct {
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

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			ergebnis := normalisiereKlassenname(f.input)
			if ergebnis != f.expected {
				t.Errorf("formatClassName(%q) = %q, erwartet %q", f.input, ergebnis, f.expected)
			}
		})
	}
}

func TestKlassensatzGruppeLoeschen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	t.Run("No class name provided", func(t *testing.T) {
		anfrage := httptest.NewRequest(http.MethodDelete, "/api/class-books", nil)
		rec := httptest.NewRecorder()
		tueren.KlassensatzGruppeLoeschen()(rec, anfrage)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("Database Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM class_books").
			WithArgs(pgxmock.AnyArg()).
			WillReturnError(fmt.Errorf("db error"))
		// Die Zeile im Log nennt die Klasse: Die Adresse im Log trägt den Abfrageteil nicht.
		var protokoll bytes.Buffer
		vorher := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&protokoll, nil)))
		t.Cleanup(func() { slog.SetDefault(vorher) })

		anfrage := httptest.NewRequest(http.MethodDelete, "/api/class-books?className=5A", nil)
		rec := httptest.NewRecorder()
		tueren.KlassensatzGruppeLoeschen()(rec, anfrage)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusInternalServerError)
		}
		if zeile := protokoll.String(); !strings.Contains(zeile, "5A") || !strings.Contains(zeile, "db error") {
			t.Errorf("Log ohne Klasse oder Ursache:\n%s", zeile)
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM class_books").
			WithArgs(pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		anfrage := httptest.NewRequest(http.MethodDelete, "/api/class-books?className=5A", nil)
		rec := httptest.NewRecorder()
		tueren.KlassensatzGruppeLoeschen()(rec, anfrage)

		if rec.Code != http.StatusOK {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusOK)
		}
	})
}

func TestKlassensaetzeZuordnen(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mockPool.Close()

		tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mockPool}}}

		anfrage := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()

		tueren.KlassensaetzeZuordnen()(rr, anfrage)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("status = %v, erwartet %v", status, http.StatusBadRequest)
		}
	})

	t.Run("No class names", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mockPool.Close()

		tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mockPool}}}

		nutzlast := map[string]any{
			"classNames": []string{"   ", ""},
			"bookIds":    []string{"11111111-1111-1111-1111-111111111111"},
		}
		rumpf, err := json.Marshal(nutzlast)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		anfrage := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(rumpf))
		rr := httptest.NewRecorder()

		tueren.KlassensaetzeZuordnen()(rr, anfrage)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("status = %v, erwartet %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Class name too long", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mockPool.Close()

		tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mockPool}}}

		nutzlast := map[string]any{
			"classNames": []string{"ThisClassNameIsWayTooLongToBeValidAndShouldBeRejected"},
			"bookIds":    []string{"11111111-1111-1111-1111-111111111111"},
		}
		rumpf, err := json.Marshal(nutzlast)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		anfrage := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(rumpf))
		rr := httptest.NewRecorder()

		tueren.KlassensaetzeZuordnen()(rr, anfrage)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("status = %v, erwartet %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Repository error", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mockPool.Close()

		tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mockPool}}}
		mockPool.ExpectExec("INSERT INTO class_books").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("db error"))

		nutzlast := map[string]any{
			"classNames": []string{"10A"},
			"bookIds":    []string{"11111111-1111-1111-1111-111111111111"},
		}
		rumpf, err := json.Marshal(nutzlast)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		anfrage := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(rumpf))
		rr := httptest.NewRecorder()

		tueren.KlassensaetzeZuordnen()(rr, anfrage)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("status = %v, erwartet %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("Success", func(t *testing.T) {
		mockPool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool() = %v", err)
		}
		defer mockPool.Close()

		mockPool.ExpectExec("INSERT INTO class_books").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 2))

		tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mockPool}}}

		nutzlast := map[string]any{
			"classNames": []string{"10A", "10B"},
			"bookIds":    []string{"11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"},
		}
		rumpf, err := json.Marshal(nutzlast)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		anfrage := httptest.NewRequest(http.MethodPost, "/add-class-books", bytes.NewBuffer(rumpf))
		rr := httptest.NewRecorder()

		tueren.KlassensaetzeZuordnen()(rr, anfrage)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("status = %v, erwartet %v", status, http.StatusOK)
		}
	})
}

func TestKlassensaetzeAendern(t *testing.T) {
	errTest := errors.New("test error")

	faelle := []struct {
		name           string
		method         string
		rumpf          string
		mockSetup      func(mock pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Invalid JSON",
			method:         "POST",
			rumpf:          `{invalid}`,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Die Anfrage ließ sich nicht lesen",
		},
		{
			name:           "Missing Class Name",
			method:         "POST",
			rumpf:          `{"bookIds": ["11111111-1111-1111-1111-111111111111"]}`,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Bitte mindestens eine Klasse angeben",
		},
		{
			name:           "Class Name Too Long",
			method:         "POST",
			rumpf:          `{"classNames": ["ThisClassNameIsWayTooLong"], "bookIds": ["11111111-1111-1111-1111-111111111111"]}`,
			mockSetup:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Der Name einer Klasse hat höchstens 20 Zeichen",
		},
		{
			name:   "Database Error",
			method: "POST",
			rumpf:  `{"classNames": ["5A"], "bookIds": ["11111111-1111-1111-1111-111111111111"]}`,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin().WillReturnError(errTest)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Ein interner Serverfehler ist aufgetreten",
		},
		{
			name:   "Success with oldClassName",
			method: "POST",
			rumpf:  `{"oldClassName": "5A", "classNames": ["5B"], "bookIds": ["11111111-1111-1111-1111-111111111111"]}`,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("^DELETE FROM class_books WHERE class_name = \\$1$").
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
				mock.ExpectExec("^DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)$").
					WithArgs([]string{"05B"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
				mock.ExpectExec("^INSERT INTO class_books \\(class_name, book_id\\).*").
					WithArgs([]string{"05B"}, []string{"11111111-1111-1111-1111-111111111111"}).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"message":"Gespeichert"`,
		},
		{
			name:   "Success multiple classNames",
			method: "POST",
			rumpf:  `{"classNames": ["5C", "5D"], "bookIds": ["11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"]}`,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("^DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)$").
					WithArgs([]string{"05C", "05D"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
				mock.ExpectExec("^INSERT INTO class_books \\(class_name, book_id\\).*").
					WithArgs([]string{"05C", "05C", "05D", "05D"}, []string{"11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222", "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"}).
					WillReturnResult(pgxmock.NewResult("INSERT", 4))
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"message":"Gespeichert"`,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("failed to create pgxmock: %v", err)
			}
			defer mock.Close()

			tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

			f.mockSetup(mock)

			anfrage := httptest.NewRequest(f.method, "/api/admin/class-books", strings.NewReader(f.rumpf))
			anfrage.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			tueren.KlassensaetzeAendern()(w, anfrage)

			if w.Code != f.expectedStatus {
				t.Errorf("w.Code = %v, erwartet %v", w.Code, f.expectedStatus)
			}

			rumpfText := strings.TrimSpace(w.Body.String())
			if !strings.Contains(rumpfText, f.expectedBody) {
				t.Errorf("Rumpf %q, erwartet darin %q", rumpfText, f.expectedBody)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
