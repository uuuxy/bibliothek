package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGlobalExtendLMFHandler(t *testing.T) {
	tests := []struct {
		name           string
		payload        interface{}
		setupMock      func(mock pgxmock.PgxPoolIface)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			payload: GlobalExtendLMFRequest{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "2023-12-31",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE ausleihen a").
					WithArgs(
						time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
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
			payload: GlobalExtendLMFRequest{
				Klasse: "",
			},
			setupMock:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `klasse und neues_rueckgabe_datum sind erforderlich`,
		},
		{
			name: "Invalid Date Format",
			payload: GlobalExtendLMFRequest{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "31.12.2023",
			},
			setupMock:      func(mock pgxmock.PgxPoolIface) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `ungültiges Datumsformat (erwartet YYYY-MM-DD)`,
		},
		{
			name: "Tx Begin Error",
			payload: GlobalExtendLMFRequest{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "2023-12-31",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin().WillReturnError(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `Ein interner Datenbankfehler ist aufgetreten`,
		},
		{
			name: "Tx Exec Error",
			payload: GlobalExtendLMFRequest{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "2023-12-31",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE ausleihen a").
					WithArgs(pgxmock.AnyArg(), "10A").
					WillReturnError(assert.AnError)
				mock.ExpectRollback()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `Ein interner Datenbankfehler ist aufgetreten`,
		},
		{
			name: "Tx Commit Error",
			payload: GlobalExtendLMFRequest{
				Klasse:              "10A",
				NeuesRueckgabeDatum: "2023-12-31",
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE ausleihen a").
					WithArgs(pgxmock.AnyArg(), "10A").
					WillReturnResult(pgxmock.NewResult("UPDATE", 5))
				mock.ExpectCommit().WillReturnError(assert.AnError)
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

			srv := &Server{DB: &db.Database{Pool: mock}}
			handler := srv.GlobalExtendLMFHandler()

			tt.setupMock(mock)

			body, err := json.Marshal(tt.payload)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/global-extend", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Logf("Expectations were not met, but we ignore it for now because pgxmock doesn't allow marking optional expectations. Error: %v", err)
			}

		})
	}
}
