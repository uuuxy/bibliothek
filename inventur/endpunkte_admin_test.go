package inventur

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleLoginGuest(t *testing.T) {
	jwtSecret := "secret-key-at-least-32-bytes-long-for-security"
	handler := NewAPIHandler(APIHandlerConfig{
		JWTSecret: jwtSecret,
	})

	req, _ := http.NewRequest("POST", "/api/login/guest", strings.NewReader(`{"password":"guest"}`))
	rr := httptest.NewRecorder()

	handler.handleLoginGuest(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestHandleLogin(t *testing.T) {
	jwtSecret := "secret-key-at-least-32-bytes-long-for-security"
	handler := NewAPIHandler(APIHandlerConfig{
		JWTSecret: jwtSecret,
	})

	req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(`{"password":"admin"}`))
	rr := httptest.NewRecorder()

	handler.handleLogin(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
