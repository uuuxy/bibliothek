package inventur

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	t.Run("ValidPayload", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		payload := map[string]string{"message": "success"}

		writeJSON(recorder, http.StatusOK, payload)

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		contentType := recorder.Header().Get("Content-Type")
		expectedContentType := "application/json; charset=utf-8"
		if contentType != expectedContentType {
			t.Errorf("expected Content-Type %q, got %q", expectedContentType, contentType)
		}

		var response map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if response["message"] != "success" {
			t.Errorf("expected message 'success', got %v", response["message"])
		}
	})

	t.Run("NilPayload", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		writeJSON(recorder, http.StatusNoContent, nil)

		if recorder.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
		}

		if recorder.Body.String() != "null\n" {
			t.Errorf("expected body 'null\\n', got %q", recorder.Body.String())
		}
	})

	t.Run("UnencodablePayload", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		// A channel cannot be encoded to JSON
		payload := make(chan int)

		// This should not panic
		writeJSON(recorder, http.StatusInternalServerError, payload)

		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
		}
	})
}
