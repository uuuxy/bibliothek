package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func orderTestServer(t *testing.T) (*Server, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	return &Server{DB: &db.Database{Pool: mock}}, mock
}

func TestSupplierOrderHandler_Validation(t *testing.T) {
	srv, _ := orderTestServer(t)
	handler := srv.SupplierOrderHandler()

	tests := []struct {
		name     string
		body     map[string]interface{}
		wantCode int
	}{
		{
			name: "Menge 0",
			body: map[string]interface{}{
				"titel_id": "123e4567-e89b-12d3-a456-426614174000",
				"menge":    0,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Menge 201",
			body: map[string]interface{}{
				"titel_id": "123e4567-e89b-12d3-a456-426614174000",
				"menge":    201,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Missing Menge",
			body: map[string]interface{}{
				"titel_id": "123e4567-e89b-12d3-a456-426614174000",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Invalid JSON",
			body: nil,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			if tt.body != nil {
				reqBody, _ = json.Marshal(tt.body)
			} else {
				reqBody = []byte(`{"titel_id": "123", "menge": "invalid"}`)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/supplier-order", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func TestSupplierOrderHandler_TitleNotFound(t *testing.T) {
	srv, mock := orderTestServer(t)
	handler := srv.SupplierOrderHandler()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT titel, coalesce\\(autor, ''\\) FROM buecher_titel").
		WithArgs("00000000-0000-0000-0000-000000000000").
		WillReturnError(pgx.ErrNoRows)
	mock.ExpectRollback()

	reqBody := `{"titel_id":"00000000-0000-0000-0000-000000000000","menge":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/supplier-order", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("erwartet 404 für unbekannten Titel, bekam %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestSupplierOrderHandler_Success(t *testing.T) {
	srv, mock := orderTestServer(t)
	handler := srv.SupplierOrderHandler()

	titelID := "123e4567-e89b-12d3-a456-426614174000"

	mock.ExpectBegin()

	// 1. resolveOrderTitel
	mock.ExpectQuery("SELECT titel, coalesce\\(autor, ''\\) FROM buecher_titel").
		WithArgs(titelID).
		WillReturnRows(pgxmock.NewRows([]string{"titel", "autor"}).AddRow("Testbuch", "Testautor"))

	// 2. GetNextSequence
	// mock sequence lock and query
	mock.ExpectQuery("SELECT coalesce\\(max\\(substr\\(t.barcode_id, \\$2\\)::bigint\\), 0\\)").
		WithArgs("B-%", 3, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"max"}).AddRow(10000))

	// 3. insertVorabBarcodes (uses CopyFrom)
	mock.ExpectCopyFrom(pgx.Identifier{"buecher_exemplare"}, []string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar"}).WillReturnResult(3)

	mock.ExpectCommit()

	reqBody := `{"titel_id":"` + titelID + `","menge":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/supplier-order", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200 OK, bekam %d (Body: %s)", rec.Code, rec.Body.String())
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("erwartet Content-Type application/pdf, bekam %s", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=bestellung_barcodes_") {
		t.Errorf("erwartet Content-Disposition attachment; filename=..., bekam %s", cd)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
