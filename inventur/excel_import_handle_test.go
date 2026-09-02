package inventur

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// offlineMetadatenClient baut einen MetadatenClient, dessen Transport jede
// Anfrage mit 404 beantwortet. Die Import-Tests duerfen NIE ins echte Netz
// (ergaenzeMetadaten wuerde sonst pro CSV-Zeile DNB/Google/OpenLibrary
// abfragen und gefundene Cover real nach uploads/ schreiben).
func offlineMetadatenClient() *MetadatenClient {
	return &MetadatenClient{httpClient: &http.Client{Transport: &mockTransportCover{
		roundTripFunc: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		},
	}}}
}

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
		err := writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{}
		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "keine datei gefunden", resp["error"])
	})

	t.Run("Missing ISBN column", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.csv")
		require.NoError(t, err)
		_, err = part.Write([]byte("titel,autor\nBuch1,Autor1"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{}
		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "spalte 'isbn' fehlt")
	})

	t.Run("Successful Import", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.csv")
		require.NoError(t, err)
		_, err = part.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{
			repo:      repo,
			metadaten: offlineMetadatenClient(),
		}

		// Der Import registriert das Default-Fach "Unbekannt" (auf dem Pool) vor dem
		// Batch-Upsert; der Batch selbst (Upsert + Exemplare) läuft atomar in einer Tx.
		erwarteFachBekannt(mock, "Unbekannt")
		mock.ExpectBegin()
		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec("CREATE SEQUENCE").WillReturnResult(pgxmock.NewResult("CREATE", 0))
		mock.ExpectExec("INSERT INTO buecher_exemplare").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 0))
		mock.ExpectCommit()

		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
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
		part, err := writer.CreateFormFile("file", "test.csv")
		require.NoError(t, err)
		_, err = part.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{
			repo:      repo,
			metadaten: offlineMetadatenClient(),
		}

		erwarteFachBekannt(mock, "Unbekannt")
		mock.ExpectBegin()
		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(assert.AnError)
		mock.ExpectRollback()
		// Fallback: der Einzel-Upsert scheitert schon beim Fach-Nachschlag (vor der Tx).
		mock.ExpectQuery(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(assert.AnError)

		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "keine bücher konnten importiert werden")
	})

	t.Run("Partial Failure", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.csv")
		require.NoError(t, err)
		_, err = part.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5\n9780987654321,Buch2,Autor2,5"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler := &APIHandler{
			repo:      repo,
			metadaten: offlineMetadatenClient(),
		}

		erwarteFachBekannt(mock, "Unbekannt")
		mock.ExpectBegin()
		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(assert.AnError)
		mock.ExpectRollback()
		// Fallback Einzel-Upsert je Zeile (Fach-Nachschlag auf dem Pool, dann Tx): erste
		// Zeile scheitert im Upsert (Begin→Rollback), zweite gelingt (Begin→…→Commit).
		erwarteFachBekannt(mock, "Unbekannt")
		mock.ExpectBegin()
		mock.ExpectQuery(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(assert.AnError)
		mock.ExpectRollback()
		erwarteFachBekannt(mock, "Unbekannt")
		mock.ExpectBegin()
		mock.ExpectQuery(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("test-id"))
		mock.ExpectExec("CREATE SEQUENCE").WillReturnResult(pgxmock.NewResult("CREATE", 0))
		mock.ExpectExec("INSERT INTO buecher_exemplare").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 5))
		mock.ExpectCommit()

		handler.handleImportExcel(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "1 bücher importiert, 1 fehlgeschlagen", resp["message"])
		assert.Equal(t, float64(1), resp["imported"])
		assert.Equal(t, float64(1), resp["failed"])
	})
}
