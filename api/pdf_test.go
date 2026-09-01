package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestGenerateDamagePDFHandler_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}

	caseID := "123e4567-e89b-12d3-a456-426614174000"

	// Mock fetchDamageCaseInfo query
	rows := pgxmock.NewRows([]string{"beschreibung", "betrag", "erstellt_am", "vorname", "nachname", "klasse", "strasse", "hausnummer", "plz", "ort", "titel", "barcode_id"}).
		AddRow("Wasserschaden", 12.50, time.Now(), "Max", "Muster", "7a", "Musterweg", "3", "61169", "Musterstadt", "Das Buch", "B-123")
	mock.ExpectQuery("SELECT sf.beschreibung, sf.betrag, sf.erstellt_am, s.vorname, s.nachname, s.klasse, COALESCE").
		WithArgs(caseID).
		WillReturnRows(rows)

	// Mock system settings query
	settingsRows := pgxmock.NewRows([]string{"schluessel", "wert"}).
		AddRow("schule_name", "Testschule").
		AddRow("schule_strasse", "Teststr").
		AddRow("schule_plz", "12345").
		AddRow("schule_ort", "Testort")

	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(settingsRows)

	// Mock markElternbriefGenerated
	mock.ExpectExec("UPDATE schadensfaelle").
		WithArgs(caseID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle/"+caseID+"/pdf", nil)
	req.SetPathValue("id", caseID)
	rec := httptest.NewRecorder()

	srv.GenerateDamagePDFHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if rec.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("expected Content-Type application/pdf, got %s", rec.Header().Get("Content-Type"))
	}

	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("expected PDF prefix in response body, got %q", string(rec.Body.Bytes()[:min(5, len(rec.Body.Bytes()))]))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestGenerateDamagePDFHandler_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}
	caseID := "00000000-0000-0000-0000-000000000000"

	mock.ExpectQuery("SELECT sf.beschreibung").
		WithArgs(caseID).
		WillReturnError(pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle/"+caseID+"/pdf", nil)
	req.SetPathValue("id", caseID)
	rec := httptest.NewRecorder()

	srv.GenerateDamagePDFHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGenerateDamagePDFHandler_MissingID(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle//pdf", nil)
	// We deliberately leave id empty
	rec := httptest.NewRecorder()

	srv.GenerateDamagePDFHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing ID, got %d", rec.Code)
	}
}

func TestGenerateDamagePDFHandler_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}
	caseID := "11111111-1111-1111-1111-111111111111"

	mock.ExpectQuery("SELECT sf.beschreibung").
		WithArgs(caseID).
		WillReturnError(context.DeadlineExceeded)

	req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle/"+caseID+"/pdf", nil)
	req.SetPathValue("id", caseID)
	rec := httptest.NewRecorder()

	srv.GenerateDamagePDFHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for DB error, got %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGenerateDamagePDFHandler_WriterError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}
	caseID := "123e4567-e89b-12d3-a456-426614174000"

	// Mock fetchDamageCaseInfo query
	rows := pgxmock.NewRows([]string{"beschreibung", "betrag", "erstellt_am", "vorname", "nachname", "klasse", "strasse", "hausnummer", "plz", "ort", "titel", "barcode_id"}).
		AddRow("Wasserschaden", 12.50, time.Now(), "Max", "Muster", "7a", "Musterweg", "3", "61169", "Musterstadt", "Das Buch", "B-123")
	mock.ExpectQuery("SELECT sf.beschreibung").
		WithArgs(caseID).
		WillReturnRows(rows)

	// Mock system settings query
	settingsRows := pgxmock.NewRows([]string{"schluessel", "wert"})
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(settingsRows)

	// Mock markElternbriefGenerated
	mock.ExpectExec("UPDATE schadensfaelle").
		WithArgs(caseID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle/"+caseID+"/pdf", nil)
	req.SetPathValue("id", caseID)
	rec := httptest.NewRecorder()

	// Use a mock response writer that simulates an error
	errWriter := &errorResponseWriter{ResponseWriter: rec}

	srv.GenerateDamagePDFHandler().ServeHTTP(errWriter, req)

	// Handler log output error but returns HTTP 200 before the error, so code should be 200
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for writer error (headers sent), got %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGenerateDamagePDFHandler_UpdateError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}
	caseID := "123e4567-e89b-12d3-a456-426614174000"

	// Mock fetchDamageCaseInfo query
	rows := pgxmock.NewRows([]string{"beschreibung", "betrag", "erstellt_am", "vorname", "nachname", "klasse", "strasse", "hausnummer", "plz", "ort", "titel", "barcode_id"}).
		AddRow("Wasserschaden", 12.50, time.Now(), "Max", "Muster", "7a", "Musterweg", "3", "61169", "Musterstadt", "Das Buch", "B-123")
	mock.ExpectQuery("SELECT sf.beschreibung").
		WithArgs(caseID).
		WillReturnRows(rows)

	// Mock system settings query
	settingsRows := pgxmock.NewRows([]string{"schluessel", "wert"})
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(settingsRows)

	// Mock markElternbriefGenerated with an error
	mock.ExpectExec("UPDATE schadensfaelle").
		WithArgs(caseID).
		WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle/"+caseID+"/pdf", nil)
	req.SetPathValue("id", caseID)
	rec := httptest.NewRecorder()

	srv.GenerateDamagePDFHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 despite update error, got %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGenerateDamagePDFHandler_InfoScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	srv := &Server{DB: &db.Database{Pool: mock}}
	caseID := "123e4567-e89b-12d3-a456-426614174000"

	// Return wrong type to trigger scan error
	rows := pgxmock.NewRows([]string{"beschreibung"}).
		AddRow("Wasserschaden")
	mock.ExpectQuery("SELECT sf.beschreibung").
		WithArgs(caseID).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle/"+caseID+"/pdf", nil)
	req.SetPathValue("id", caseID)
	rec := httptest.NewRecorder()

	srv.GenerateDamagePDFHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for info scan error, got %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

type errorResponseWriter struct {
	http.ResponseWriter
}

func (w *errorResponseWriter) Write(b []byte) (int, error) {
	return 0, context.DeadlineExceeded
}
