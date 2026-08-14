package inventur

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleImportExcel(t *testing.T) {
	t.Run("Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/import", nil)
		w := httptest.NewRecorder()

		handler := &APIHandler{}
		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("Missing File", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{}
		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		assert.Equal(t, "keine datei gefunden", resp["error"])
	})

	t.Run("Missing ISBN column", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		_, _ = part.Write([]byte("titel,autor\nBuch1,Autor1"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{}
		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		assert.Contains(t, resp["error"], "spalte 'isbn' fehlt")
	})

	t.Run("Successful Import", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		_, _ = part.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{
			repo: repo,
			metadaten: NeuerMetadatenClient(),
		}

		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))

		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		assert.Equal(t, "1 bücher importiert", resp["message"])
		assert.Equal(t, float64(1), resp["imported"])
		assert.Equal(t, float64(0), resp["failed"])
	})

	t.Run("Import failed completely", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		_, _ = part.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{
			repo: repo,
			metadaten: NeuerMetadatenClient(),
		}

		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(assert.AnError)
		// Let Fallback single Insert fail
		mock.ExpectQuery(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(assert.AnError)

		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		assert.Contains(t, resp["error"], "keine bücher konnten importiert werden")
	})

	t.Run("Partial Failure", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		_, _ = part.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5\n9780987654321,Buch2,Autor2,5"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{
			repo: repo,
			metadaten: NeuerMetadatenClient(),
		}

		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(assert.AnError)
		// Fallback single Insert: first one fails, second one succeeds
		mock.ExpectQuery(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(assert.AnError)
		mock.ExpectQuery(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("test-id"))

		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		assert.Equal(t, "1 bücher importiert, 1 fehlgeschlagen", resp["message"])
		assert.Equal(t, float64(1), resp["imported"])
		assert.Equal(t, float64(1), resp["failed"])
	})
}
