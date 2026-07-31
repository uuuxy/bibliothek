package api

import (
	"net/http"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPrintErsatzEtikettHandler_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}

	// Erwartungen:
	// 1. SELECT Query zum Lesen der Label-Details
	mock.ExpectQuery("SELECT e.barcode_id, t.titel, coalesce").
		WithArgs("EX-123").
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "titel", "autor", "isbn"}).
			AddRow("EX-123", "Test Titel", "Test Autor", "1234567890"))

	// 2. UPDATE Query in s.markEtikettGedruckt
	mock.ExpectExec("UPDATE buecher_exemplare SET etikett_gedruckt = true").
		WithArgs("EX-123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodGet, "/api/exemplare/EX-123/ersatzetikett", nil)
	req.SetPathValue("id", "EX-123")
	rec := httptest.NewRecorder()

	srv.PrintErsatzEtikettHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet Code 200, bekam %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("erwartet Content-Type application/pdf, bekam %q", contentType)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "filename=\"Ersatz_Etikett_EX-123.pdf\"") {
		t.Errorf("unerwarteter Content-Disposition Header: %s", contentDisposition)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nicht alle DB-Erwartungen wurden erfüllt: %v", err)
	}
}

func TestPrintErsatzEtikettHandler_MissingID(t *testing.T) {
	srv := &Server{} // DB wird hier nicht benötigt, bricht vorher ab

	req := httptest.NewRequest(http.MethodGet, "/api/exemplare//ersatzetikett", nil)
	// Absichtlich keine ID setzen
	rec := httptest.NewRecorder()

	srv.PrintErsatzEtikettHandler()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("erwartet Code 400 Bad Request, bekam %d", rec.Code)
	}
}

func TestPrintErsatzEtikettHandler_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}

	// Erwartung: SELECT Query wirft ErrNoRows
	mock.ExpectQuery("SELECT e.barcode_id, t.titel, coalesce").
		WithArgs("EX-999").
		WillReturnError(pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/exemplare/EX-999/ersatzetikett", nil)
	req.SetPathValue("id", "EX-999")
	rec := httptest.NewRecorder()

	srv.PrintErsatzEtikettHandler()(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("erwartet Code 404 Not Found, bekam %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nicht alle DB-Erwartungen wurden erfüllt: %v", err)
	}
}

func TestPrintErsatzEtikettHandler_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}

	// Erwartung: SELECT Query wirft generischen Fehler
	mock.ExpectQuery("SELECT e.barcode_id, t.titel, coalesce").
		WithArgs("EX-ERR").
		WillReturnError(errors.New("db error")) // or just errors.New("db error")

	req := httptest.NewRequest(http.MethodGet, "/api/exemplare/EX-ERR/ersatzetikett", nil)
	req.SetPathValue("id", "EX-ERR")
	rec := httptest.NewRecorder()

	srv.PrintErsatzEtikettHandler()(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("erwartet Code 500 Internal Server Error, bekam %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nicht alle DB-Erwartungen wurden erfüllt: %v", err)
	}
}
