package system

import (
	"bibliothek/api/tuer"
	"bibliothek/db"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

func TestMailVorlageAendern_Erfolg(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	server := &Tueren{
		Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}},
	}

	reqBody := map[string]string{
		"betreff":   "Neuer Betreff",
		"text_body": "Neuer Text",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Anfrage kodieren: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/mahnung_1", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "mahnung_1")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	mock.ExpectQuery(`UPDATE mail_vorlagen v SET betreff = \$1, text_body = \$2`).
		WithArgs("Neuer Betreff", "Neuer Text", "mahnung_1").
		WillReturnRows(pgxmock.NewRows([]string{"typ", "betreff_neu", "text_neu"}).AddRow("mahnung_1", true, true))

	server.MailVorlageAendern().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusOK)
	}

	var res map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &res)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if res["message"] != "Erfolgreich gespeichert" {
		t.Errorf("res[\"message\"] = %q, erwartet %q", res["message"], "Erfolgreich gespeichert")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestMailVorlageAendern_OhneKennung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	server := &Tueren{
		Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}},
	}

	reqBody := map[string]string{
		"betreff":   "Neuer Betreff",
		"text_body": "Neuer Text",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Anfrage kodieren: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/", bytes.NewReader(bodyBytes))
	// Kein PathValue("id") gesetzt oder leer
	req.SetPathValue("id", "")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	server.MailVorlageAendern().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusBadRequest)
	}
}

func TestMailVorlageAendern_UngueltigesJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	server := &Tueren{
		Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}},
	}

	req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/mahnung_1", bytes.NewReader([]byte("{invalid-json}")))
	req.SetPathValue("id", "mahnung_1")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	server.MailVorlageAendern().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusBadRequest)
	}
}

func TestMailVorlageAendern_Datenbankfehler(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	server := &Tueren{
		Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}},
	}

	reqBody := map[string]string{
		"betreff":   "Neuer Betreff",
		"text_body": "Neuer Text",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Anfrage kodieren: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/mahnung_1", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "mahnung_1")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	mock.ExpectQuery(`UPDATE mail_vorlagen v SET betreff = \$1, text_body = \$2`).
		WithArgs("Neuer Betreff", "Neuer Text", "mahnung_1").
		WillReturnError(errors.New("db error"))

	server.MailVorlageAendern().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusInternalServerError)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
