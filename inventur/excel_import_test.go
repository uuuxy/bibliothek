package inventur

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"
	"encoding/json"
	"errors"

	"github.com/pashagolub/pgxmock/v4"
)

func TestHandleImportExcel_MethodNotAllowed(t *testing.T) {
	handler := &APIHandler{}

	req := httptest.NewRequest(http.MethodGet, "/import", nil)
	rr := httptest.NewRecorder()

	handler.handleImportExcel(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHandleImportExcel_FileMissing(t *testing.T) {
	handler := &APIHandler{}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	// Missing file part
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler.handleImportExcel(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	if !strings.Contains(rr.Body.String(), "keine datei gefunden") {
		t.Errorf("expected missing file error, got: %s", rr.Body.String())
	}
}

func TestHandleImportExcel_InvalidCSV(t *testing.T) {
	handler := &APIHandler{}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.csv")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("titel\nBook A"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler.handleImportExcel(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	if !strings.Contains(rr.Body.String(), "spalte 'isbn' fehlt in der datei") {
		t.Errorf("expected missing isbn error, got: %s", rr.Body.String())
	}
}

func TestHandleImportExcel_TooManyRows(t *testing.T) {
	handler := &APIHandler{}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.csv")
	if err != nil {
		t.Fatal(err)
	}
	// Write header + 100,001 rows
	part.Write([]byte("isbn\n"))
	for i := 0; i < 100001; i++ {
		part.Write([]byte("123\n"))
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler.handleImportExcel(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	if !strings.Contains(rr.Body.String(), "zu viele zeilen") {
		t.Errorf("expected too many rows error, got: %s", rr.Body.String())
	}
}

type MockTransport struct{}

func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       http.NoBody,
	}, nil
}

func TestHandleImportExcel_Success(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	metadaten := &MetadatenClient{httpClient: &http.Client{Transport: &MockTransport{}}}
	handler := &APIHandler{
		repo:      repo,
		metadaten: metadaten,
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.csv")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("isbn,titel,autor,fach,klasse,bestand\n1234567890,Test Book,Test Author,Math,5,10"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	mock.ExpectExec("(?s)INSERT INTO buecher_titel .* ON CONFLICT \\(isbn\\) DO UPDATE SET .*").
		WithArgs(
			pgxmock.AnyArg(), // isbns
			pgxmock.AnyArg(),  // titles
			pgxmock.AnyArg(),// authors
			pgxmock.AnyArg(),           // coverUrls
			pgxmock.AnyArg(),       // subjects
			pgxmock.AnyArg(),             // grades
			pgxmock.AnyArg(),           // tracks
			pgxmock.AnyArg(),            // stocks
			pgxmock.AnyArg(),         // lastCounteds
			pgxmock.AnyArg(),       // medientypen
			pgxmock.AnyArg(),               // jahrgaengeVon
			pgxmock.AnyArg(),               // jahrgaengeBis
			pgxmock.AnyArg(),           // untertitel
			pgxmock.AnyArg(),           // verlage
			pgxmock.AnyArg(),               // erscheinungsjahre
			pgxmock.AnyArg(),           // beschreibungen
			pgxmock.AnyArg(), // erweiterteEigenschaften
			pgxmock.AnyArg(),           // signaturen
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	rr := httptest.NewRecorder()
	handler.handleImportExcel(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Logf("Response body: %s", rr.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err == nil {
		if imported, ok := res["imported"].(float64); ok {
			if imported != 1 {
				t.Errorf("expected 1 imported book, got %v", res["imported"])
			}
		} else {
			t.Errorf("imported key missing or wrong type")
		}
	} else {
		t.Errorf("Failed to parse json: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestHandleImportExcel_FallbackSuccess(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	metadaten := &MetadatenClient{httpClient: &http.Client{Transport: &MockTransport{}}}
	handler := &APIHandler{
		repo:      repo,
		metadaten: metadaten,
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.csv")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("isbn,titel,autor,fach,klasse,bestand\n1234567890,Test Book,Test Author,Math,5,0"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Batch insert fails
	mock.ExpectExec("(?s)INSERT INTO buecher_titel .* ON CONFLICT \\(isbn\\) DO UPDATE SET .*").
		WithArgs(
			pgxmock.AnyArg(), // isbns
			pgxmock.AnyArg(),  // titles
			pgxmock.AnyArg(),// authors
			pgxmock.AnyArg(),           // coverUrls
			pgxmock.AnyArg(),       // subjects
			pgxmock.AnyArg(),             // grades
			pgxmock.AnyArg(),           // tracks
			pgxmock.AnyArg(),            // stocks
			pgxmock.AnyArg(),         // lastCounteds
			pgxmock.AnyArg(),       // medientypen
			pgxmock.AnyArg(),               // jahrgaengeVon
			pgxmock.AnyArg(),               // jahrgaengeBis
			pgxmock.AnyArg(),           // untertitel
			pgxmock.AnyArg(),           // verlage
			pgxmock.AnyArg(),               // erscheinungsjahre
			pgxmock.AnyArg(),           // beschreibungen
			pgxmock.AnyArg(), // erweiterteEigenschaften
			pgxmock.AnyArg(),           // signaturen
		).WillReturnError(errors.New("db batch error"))

	// Fallback single insert succeeds
	mock.ExpectQuery("(?s)INSERT INTO buecher_titel .* ON CONFLICT \\(isbn\\) DO UPDATE SET .* RETURNING id").
		WithArgs(
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(), // stock
			pgxmock.AnyArg(), // lastCounteds
			pgxmock.AnyArg(),
			pgxmock.AnyArg(), // jahrgang_von
			pgxmock.AnyArg(), // jahrgang_bis
			pgxmock.AnyArg(), // untertitel
			pgxmock.AnyArg(), // verlag
			pgxmock.AnyArg(), // erscheinungsjahr
			pgxmock.AnyArg(), // beschreibung
			pgxmock.AnyArg(), // erweiterte Eigenschaften json marshalled
			pgxmock.AnyArg(), // signatur
		).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("test-id-1"))


	rr := httptest.NewRecorder()
	handler.handleImportExcel(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Logf("Response body: %s", rr.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err == nil {
		if imported, ok := res["imported"].(float64); ok {
			if imported != 1 {
				t.Errorf("expected 1 imported book, got %v", res["imported"])
			}
		} else {
			t.Errorf("imported key missing or wrong type")
		}
	} else {
		t.Errorf("Failed to parse json: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestHandleImportExcel_AllFail(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	metadaten := &MetadatenClient{httpClient: &http.Client{Transport: &MockTransport{}}}
	handler := &APIHandler{
		repo:      repo,
		metadaten: metadaten,
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.csv")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("isbn,titel,autor,fach,klasse,bestand\n1234567890,Test Book,Test Author,Math,5,10"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Batch insert fails
	mock.ExpectExec("(?s)INSERT INTO buecher_titel .* ON CONFLICT \\(isbn\\) DO UPDATE SET .*").
		WithArgs(
			pgxmock.AnyArg(), // isbns
			pgxmock.AnyArg(),  // titles
			pgxmock.AnyArg(),// authors
			pgxmock.AnyArg(),           // coverUrls
			pgxmock.AnyArg(),       // subjects
			pgxmock.AnyArg(),             // grades
			pgxmock.AnyArg(),           // tracks
			pgxmock.AnyArg(),            // stocks
			pgxmock.AnyArg(),         // lastCounteds
			pgxmock.AnyArg(),       // medientypen
			pgxmock.AnyArg(),               // jahrgaengeVon
			pgxmock.AnyArg(),               // jahrgaengeBis
			pgxmock.AnyArg(),           // untertitel
			pgxmock.AnyArg(),           // verlage
			pgxmock.AnyArg(),               // erscheinungsjahre
			pgxmock.AnyArg(),           // beschreibungen
			pgxmock.AnyArg(), // erweiterteEigenschaften
			pgxmock.AnyArg(),           // signaturen
		).WillReturnError(errors.New("db batch error"))

	// Fallback single insert fails
	mock.ExpectQuery("(?s)INSERT INTO buecher_titel .* RETURNING id").
		WithArgs(
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			pgxmock.AnyArg(), // stock
			pgxmock.AnyArg(), // lastCounteds
			pgxmock.AnyArg(),
			pgxmock.AnyArg(), // jahrgang_von
			pgxmock.AnyArg(), // jahrgang_bis
			pgxmock.AnyArg(), // untertitel
			pgxmock.AnyArg(), // verlag
			pgxmock.AnyArg(), // erscheinungsjahr
			pgxmock.AnyArg(), // beschreibung
			pgxmock.AnyArg(), // erweiterte Eigenschaften json marshalled
			pgxmock.AnyArg(), // signatur
		).WillReturnError(errors.New("db single error"))

	rr := httptest.NewRecorder()
	handler.handleImportExcel(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		t.Logf("Response body: %s", rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), "keine bücher konnten importiert werden") {
		t.Errorf("expected no books imported error, got: %s", rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
