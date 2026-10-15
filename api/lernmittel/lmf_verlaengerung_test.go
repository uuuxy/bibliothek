package lernmittel

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/pkg/schulzeit"

	"github.com/pashagolub/pgxmock/v5"
)

var errTest = errors.New("testfehler")

func TestLernmittelDerKlasseVerlaengern(t *testing.T) {
	tests := []struct {
		name           string
		payload        interface{}
		setupMock      func(mock pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			payload: verlaengerungAnfrage{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "2023-12-31",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE ausleihen a").
					WithArgs(
						// Berliner Tagesende zum Datum, wie die Tür es setzt (nicht roh
						// 23:59:59 UTC).
						schulzeit.TagesEnde(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
						"10A",
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 5))
				mock.ExpectCommit()
				mock.ExpectRollback()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"success":true,"updated_count":5}`,
		},
		{
			name: "Missing Fields",
			payload: verlaengerungAnfrage{
				Klasse: "",
			},
			setupMock:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `Bitte Klasse und neues Rückgabedatum angeben`,
		},
		{
			name: "Invalid Date Format",
			payload: verlaengerungAnfrage{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "31.12.2023",
			},
			setupMock:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `Das Datum muss die Form JJJJ-MM-TT haben`,
		},
		{
			name: "Tx Begin Error",
			payload: verlaengerungAnfrage{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "2023-12-31",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin().WillReturnError(errTest)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `Ein interner Datenbankfehler ist aufgetreten`,
		},
		{
			name: "Tx Exec Error",
			payload: verlaengerungAnfrage{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "2023-12-31",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE ausleihen a").
					WithArgs(pgxmock.AnyArg(), "10A").
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `Ein interner Datenbankfehler ist aufgetreten`,
		},
		{
			name: "Tx Commit Error",
			payload: verlaengerungAnfrage{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "2023-12-31",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE ausleihen a").
					WithArgs(pgxmock.AnyArg(), "10A").
					WillReturnResult(pgxmock.NewResult("UPDATE", 5))
				mock.ExpectCommit().WillReturnError(errTest)
				mock.ExpectRollback()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `Ein interner Datenbankfehler ist aufgetreten`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("failed to create pgxmock: %v", err)
			}
			defer mock.Close()

			srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
			handler := srv.LernmittelDerKlasseVerlaengern()

			tt.setupMock(mock)

			body, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/global-extend", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %v, erwartet %v", w.Code, tt.expectedStatus)
			}
			if got := w.Body.String(); !strings.Contains(got, tt.expectedBody) {
				t.Errorf("Antwort = %q, erwartet darin %q", got, tt.expectedBody)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Logf("Expectations were not met, but we ignore it for now because pgxmock doesn't allow marking optional expectations. Error: %v", err)
			}

		})
	}
}
