package api

import (
	"bibliothek/db"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMailTemplateHandler_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	server := &Server{
		DB: &db.Database{Pool: mock},
	}

	reqBody := map[string]string{
		"betreff":   "Neuer Betreff",
		"text_body": "Neuer Text",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/mahnung_1", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "mahnung_1")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	mock.ExpectExec(`UPDATE mail_vorlagen SET betreff = \$1, text_body = \$2 WHERE id = \$3`).
		WithArgs("Neuer Betreff", "Neuer Text", "mahnung_1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	server.UpdateMailTemplateHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var res map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Erfolgreich gespeichert", res["message"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMailTemplateHandler_MissingID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	server := &Server{
		DB: &db.Database{Pool: mock},
	}

	reqBody := map[string]string{
		"betreff":   "Neuer Betreff",
		"text_body": "Neuer Text",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/", bytes.NewReader(bodyBytes))
	// Kein PathValue("id") gesetzt oder leer
	req.SetPathValue("id", "")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	server.UpdateMailTemplateHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMailTemplateHandler_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	server := &Server{
		DB: &db.Database{Pool: mock},
	}

	req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/mahnung_1", bytes.NewReader([]byte("{invalid-json}")))
	req.SetPathValue("id", "mahnung_1")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	server.UpdateMailTemplateHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMailTemplateHandler_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	server := &Server{
		DB: &db.Database{Pool: mock},
	}

	reqBody := map[string]string{
		"betreff":   "Neuer Betreff",
		"text_body": "Neuer Text",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/mahnung_1", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "mahnung_1")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	mock.ExpectExec(`UPDATE mail_vorlagen SET betreff = \$1, text_body = \$2 WHERE id = \$3`).
		WithArgs("Neuer Betreff", "Neuer Text", "mahnung_1").
		WillReturnError(errors.New("db error"))

	server.UpdateMailTemplateHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	assert.NoError(t, mock.ExpectationsWereMet())
}
